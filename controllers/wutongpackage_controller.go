/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	cdocker "github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/config"
	"github.com/docker/distribution/reference"
	dtypes "github.com/docker/docker/api/types"
	dclient "github.com/docker/docker/client"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong-operator/util/commonutil"
	"github.com/wutong-paas/wutong-operator/util/constants"
	"github.com/wutong-paas/wutong-operator/util/downloadutil"
	initcontainerd "github.com/wutong-paas/wutong-operator/util/init-containerd"
	"github.com/wutong-paas/wutong-operator/util/retryutil"
	"github.com/wutong-paas/wutong-operator/util/tarutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
)

var errorClusterConfigNotReady = fmt.Errorf("cluster config can not be ready")
var errorClusterConfigNoLocalHub = fmt.Errorf("cluster spec not have local image hub info ")
var pkgDst = "/opt/wutong/pkg/files"

// WutongPackageReconciler reconciles a WutongPackage object
type WutongPackageReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=wutong.io,resources=wutongpackages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wutong.io,resources=wutongpackages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wutong.io,resources=wutongpackages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WutongPackage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *WutongPackageReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("wutongpackage", request.NamespacedName)
	cancleCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Fetch the WutongPackage instance
	pkg := &wutongv1alpha1.WutongPackage{}
	err := r.Client.Get(cancleCtx, request.NamespacedName, pkg)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	cluster := &wutongv1alpha1.WutongCluster{}
	if err := r.Client.Get(cancleCtx, types.NamespacedName{Namespace: pkg.Namespace, Name: constants.WutongClusterName}, cluster); err != nil {
		log.Error(err, "get wutongcluster.")
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	if !cluster.Spec.ConfigCompleted {
		log.V(6).Info("wutongcluster is not completed, waiting!!")
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	// if instsall mode is full online, set package to ready directly
	if cluster.Spec.InstallMode == wutongv1alpha1.InstallationModeFullOnline {
		log.Info("set package to ready directly", "install mode", cluster.Spec.InstallMode)
		pkg.Status = initPackageStatus(wutongv1alpha1.Completed)
		if err := updateCRStatus(r.Client, pkg); err != nil {
			log.Error(err, "update package status")
			return reconcile.Result{RequeueAfter: time.Second * 5}, nil
		}
		return reconcile.Result{}, nil
	}

	updateStatus, re := checkStatusCanReturn(pkg)
	if updateStatus {
		if err := updateCRStatus(r.Client, pkg); err != nil {
			log.Error(err, "update package status failure ")
			return reconcile.Result{RequeueAfter: time.Second * 5}, nil
		}
		return reconcile.Result{}, nil
	}
	if re != nil {
		return *re, nil
	}

	//need handle condition
	p, err := newpkg(cancleCtx, r.Client, pkg, cluster, log)
	if err != nil {
		if p != nil {
			p.updateConditionStatus(wutongv1alpha1.Init, wutongv1alpha1.Failed)
			p.updateConditionResion(wutongv1alpha1.Init, err.Error(), "create package handle failure")
			_ = p.updateCRStatus()
		}
		log.Error(err, "create package handle failure ")
		return reconcile.Result{RequeueAfter: time.Second * 5}, nil
	}

	// handle package
	if err = p.handle(); err != nil {
		if err == errorClusterConfigNoLocalHub {
			log.V(4).Info("waiting local image hub ready")
		} else if err == errorClusterConfigNotReady {
			log.Info("waiting cluster config ready")
		} else {
			log.Error(err, "failed to handle wutong package.")
		}
		return reconcile.Result{RequeueAfter: 8 * time.Second}, nil
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WutongPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wutongv1alpha1.WutongPackage{}).
		Complete(r)
}

func initPackageStatus(status wutongv1alpha1.PackageConditionStatus) wutongv1alpha1.WutongPackageStatus {
	return wutongv1alpha1.WutongPackageStatus{
		Conditions: []wutongv1alpha1.PackageCondition{
			{
				Type:               wutongv1alpha1.Init,
				Status:             status,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               wutongv1alpha1.DownloadPackage,
				Status:             status,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               wutongv1alpha1.UnpackPackage,
				Status:             status,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               wutongv1alpha1.PushImage,
				Status:             status,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               wutongv1alpha1.Ready,
				Status:             status,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
		},
		ImagesPushed: []wutongv1alpha1.WutongPackageImage{},
	}
}

// checkStatusCanReturn if pkg status in the working state, straight back
func checkStatusCanReturn(pkg *wutongv1alpha1.WutongPackage) (updateStatus bool, re *reconcile.Result) {
	if len(pkg.Status.Conditions) == 0 {
		pkg.Status = initPackageStatus(wutongv1alpha1.Waiting)
		return true, &reconcile.Result{}
	}
	completedCount := 0
	for _, cond := range pkg.Status.Conditions {
		if cond.Status == wutongv1alpha1.Running {
			return false, &reconcile.Result{}
		}
		//have failed conditions, retry
		if cond.Status == wutongv1alpha1.Failed {
			return false, nil
		}

		if cond.Status == wutongv1alpha1.Completed {
			completedCount++
		}
	}
	if completedCount == len(pkg.Status.Conditions) {
		return false, &reconcile.Result{}
	}
	return false, nil
}

type pkg struct {
	ctx              context.Context
	client           client.Client
	pkg              *wutongv1alpha1.WutongPackage
	cluster          *wutongv1alpha1.WutongCluster
	log              logr.Logger
	downloadPackage  bool
	localPackagePath string
	// Deprecated: no longer download installation package.
	downloadPackageURL string
	// Deprecated: no longer download installation package.
	downloadPackageMD5  string
	downloadImageDomain string
	pushImageDomain     string
	// Deprecated: no longer download installation package.
	totalImageNum int32
	//need download images
	images        map[string]string
	version       string
	containerdCli *initcontainerd.ContainerdAPI
}

func newpkg(ctx context.Context, client client.Client, p *wutongv1alpha1.WutongPackage, cluster *wutongv1alpha1.WutongCluster, reqLogger logr.Logger) (*pkg, error) {
	containerdCli, err := initcontainerd.InitContainerd()
	if err != nil {
		reqLogger.Error(err, "failed to create docker client")
		return nil, err
	}
	pkg := &pkg{
		ctx:    ctx,
		client: client,
		pkg:    p.DeepCopy(),
		// Deprecated: no longer download installation package.
		totalImageNum: 23,
		images:        make(map[string]string, 23),
		log:           reqLogger,
		version:       cluster.Spec.InstallVersion,
		cluster:       cluster,
		containerdCli: containerdCli,
	}
	return pkg, nil
}

func (p *pkg) configByCluster(c *wutongv1alpha1.WutongCluster) error {
	if !c.Spec.ConfigCompleted {
		return errorClusterConfigNotReady
	}

	// check if image repository is ready
	if !p.isImageRepositoryReady() {
		return errorClusterConfigNoLocalHub
	}

	if c.Spec.InstallVersion != "" {
		p.version = c.Spec.InstallVersion
	}
	p.localPackagePath = p.pkg.Spec.PkgPath

	p.images = map[string]string{}
	return nil
}

func (p *pkg) updateCRStatus() error {
	return updateCRStatus(p.client, p.pkg)
}

func updateCRStatus(client client.Client, pkg *wutongv1alpha1.WutongPackage) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &wutongv1alpha1.WutongPackage{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: pkg.Namespace, Name: pkg.Name}, latest); err != nil {
			return fmt.Errorf("getting latest wutong package: %v", err)
		}
		pkg.ResourceVersion = latest.ResourceVersion
		return client.Status().Update(ctx, pkg)
	}); err != nil {
		return fmt.Errorf("failed to update wutongpackage status: %v", err)
	}
	return nil
}

func (p *pkg) checkClusterConfig() error {
	cluster := p.cluster
	if err := p.configByCluster(cluster); err != nil {
		return err
	}
	switch cluster.Spec.InstallMode {
	default:
		p.downloadImageDomain = cluster.Spec.WutongImageRepository
		if p.downloadImageDomain == "" {
			p.downloadImageDomain = "wutong"
		}
		if cluster.Spec.ImageHub != nil {
			p.pushImageDomain = cluster.Spec.ImageHub.Domain
			if cluster.Spec.ImageHub.Namespace != "" {
				p.pushImageDomain += "/" + cluster.Spec.ImageHub.Namespace
			}
		}
		if p.pushImageDomain == "" {
			p.pushImageDomain = constants.DefImageRepository
		}
	}
	return nil
}

func (p *pkg) findCondition(typ3 wutongv1alpha1.PackageConditionType) *wutongv1alpha1.PackageCondition {
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			return &p.pkg.Status.Conditions[i]
		}
	}
	return nil
}
func (p *pkg) updateConditionStatus(typ3 wutongv1alpha1.PackageConditionType, status wutongv1alpha1.PackageConditionStatus) {
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			if p.pkg.Status.Conditions[i].Status != status {
				p.pkg.Status.Conditions[i].LastTransitionTime = metav1.Now()
			}
			p.pkg.Status.Conditions[i].LastHeartbeatTime = metav1.Now()
			p.pkg.Status.Conditions[i].Status = status
			if status == wutongv1alpha1.Completed {
				p.pkg.Status.Conditions[i].Progress = 100
				p.pkg.Status.Conditions[i].Reason = ""
				p.pkg.Status.Conditions[i].Message = ""
			}
			break
		}
	}
}
func (p *pkg) updateConditionResion(typ3 wutongv1alpha1.PackageConditionType, resion, message string) {
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			p.pkg.Status.Conditions[i].LastHeartbeatTime = metav1.Now()
			p.pkg.Status.Conditions[i].Reason = resion
			p.pkg.Status.Conditions[i].Message = message
			break
		}
	}
}
func (p *pkg) updateConditionProgress(typ3 wutongv1alpha1.PackageConditionType, progress int32) bool {
	if progress > 100 {
		progress = 100
	}
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			p.pkg.Status.Conditions[i].LastHeartbeatTime = metav1.Now()
			if p.pkg.Status.Conditions[i].Progress != int(progress) {
				p.pkg.Status.Conditions[i].Progress = int(progress)
				return true
			}
		}
	}
	return false
}
func (p *pkg) completeCondition(con *wutongv1alpha1.PackageCondition) error {
	if con == nil {
		return nil
	}
	p.updateConditionStatus(con.Type, wutongv1alpha1.Completed)
	return p.updateCRStatus()
}
func (p *pkg) runningCondition(con *wutongv1alpha1.PackageCondition) error {
	if con == nil {
		return nil
	}
	p.updateConditionStatus(con.Type, wutongv1alpha1.Running)
	return p.updateCRStatus()
}
func (p *pkg) canDownload() bool {
	if con := p.findCondition(wutongv1alpha1.DownloadPackage); con != nil {
		if con.Status != wutongv1alpha1.Completed {
			if !p.downloadPackage {
				p.log.Info("not need download package")
				if err := p.completeCondition(con); err != nil {
					p.log.Error(err, "complete download condition because of not need download failure %s")
				}
				return false
			}
			if err := p.runningCondition(con); err != nil {
				p.log.Error(err, "complete download condition because of not need download failure %s")
			}
			return true
		}
		return false
	}
	return false
}

func (p *pkg) canUnpack() bool {
	if con := p.findCondition(wutongv1alpha1.DownloadPackage); con != nil {
		//Must conditions are not met
		if con.Status != wutongv1alpha1.Completed {
			return false
		}
		if uncon := p.findCondition(wutongv1alpha1.UnpackPackage); uncon != nil {
			//status is waiting
			if uncon.Status != wutongv1alpha1.Completed {
				if !p.downloadPackage {
					if err := p.completeCondition(uncon); err != nil {
						p.log.Error(err, "complete unpack package condition because of not need download failure %s")
					}
					return false
				}
				if err := p.runningCondition(uncon); err != nil {
					p.log.Error(err, "running unpack package condition failure %s")
				}
				return true
			}
		}
	}
	return false
}
func (p *pkg) canPushImage() bool {
	if uncon := p.findCondition(wutongv1alpha1.UnpackPackage); uncon != nil {
		if uncon.Status != wutongv1alpha1.Completed {
			return false
		}
		if pcon := p.findCondition(wutongv1alpha1.PushImage); pcon != nil {
			if pcon.Status != wutongv1alpha1.Completed {
				return true
			}
		}
	}
	return false
}

func (p *pkg) canReady() bool {
	if pcon := p.findCondition(wutongv1alpha1.PushImage); pcon != nil && pcon.Status == wutongv1alpha1.Completed {
		return true
	}
	return false
}
func (p *pkg) setInitStatus() error {
	if con := p.findCondition(wutongv1alpha1.Init); con != nil {
		if con.Status != wutongv1alpha1.Completed {
			con.Status = wutongv1alpha1.Completed
			if err := p.updateCRStatus(); err != nil {
				p.log.Error(err, "failed to update wutongpackage status.")
				return err
			}
		}
	}
	return nil
}

// donwnloadPackage download package
func (p *pkg) donwnloadPackage() error {
	p.log.Info(fmt.Sprintf("start download package from %s", p.downloadPackageURL))
	downloadListener := &downloadutil.DownloadWithProgress{
		URL:       p.downloadPackageURL,
		SavedPath: p.localPackagePath,
		Wanted:    p.downloadPackageMD5,
	}
	// first chack exist file md5
	file, _ := os.Open(p.localPackagePath)
	if file != nil {
		err := downloadListener.CheckMD5(file)
		_ = file.Close()
		if err == nil {
			p.log.Info("wutong package file is exists")
			return nil
		}
	}
	p.log.Info("wutong package file does not exists, downloading background ...")
	var stop = make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-ticker.C:
				progress := downloadListener.Percent
				//Make time for later in the download process
				realProgress := int32(progress) - int32(float64(progress)*0.05)
				if p.updateConditionProgress(wutongv1alpha1.DownloadPackage, realProgress) {
					if err := p.updateCRStatus(); err != nil {
						// ignore error
						p.log.Info("update number extracted: %v", err)
					}
				}
			case <-stop:
				return
			}
		}
	}()
	if err := downloadListener.Download(); err != nil {
		p.log.Error(err, "download wutong package error, will retry")
		p.updateConditionResion(wutongv1alpha1.Init, err.Error(), "download wutong package error, will retry")
		_ = p.updateCRStatus()
		err = downloadListener.Download()
		if err != nil {
			p.log.Error(err, "download wutong package error, not retry")
			return err
		}
	}
	//stop watch progress
	stop <- struct{}{}
	p.log.Info(fmt.Sprintf("success download package from %s", p.downloadPackageURL))
	return nil
}

// handle
func (p *pkg) handle() error {
	p.log.V(5).Info("start handling wutong package.")
	// check prerequisites
	if err := p.checkClusterConfig(); err != nil {
		p.log.V(6).Info(fmt.Sprintf("check cluster config: %v", err))
		//To continue waiting
		if err == errorClusterConfigNotReady || err == errorClusterConfigNoLocalHub {
			return err
		}
		p.updateConditionStatus(wutongv1alpha1.Init, wutongv1alpha1.Waiting)
		p.updateConditionResion(wutongv1alpha1.Init, err.Error(), "get wutong cluster config failure")
		_ = p.updateCRStatus()
		return err
	}
	//update init condition status is complete
	if err := p.setInitStatus(); err != nil {
		p.log.Error(err, "set init status")
		p.updateConditionStatus(wutongv1alpha1.Init, wutongv1alpha1.Failed)
		p.updateConditionResion(wutongv1alpha1.Init, err.Error(), "set init status failure")
		_ = p.updateCRStatus()
		return err
	}
	if p.canDownload() {
		//download pkg
		if err := p.donwnloadPackage(); err != nil {
			p.log.Error(err, "download package")
			p.updateConditionStatus(wutongv1alpha1.DownloadPackage, wutongv1alpha1.Failed)
			p.updateConditionResion(wutongv1alpha1.DownloadPackage, err.Error(), "download package failure")
			_ = p.updateCRStatus()
			return fmt.Errorf("failed to download package %s", err.Error())
		}
		p.log.Info("handle downlaod package success")
		p.updateConditionStatus(wutongv1alpha1.DownloadPackage, wutongv1alpha1.Completed)
		return p.updateCRStatus()
	}

	if p.canUnpack() {
		//unstar the installation package
		if err := p.untartar(); err != nil {
			p.updateConditionStatus(wutongv1alpha1.UnpackPackage, wutongv1alpha1.Failed)
			p.updateConditionResion(wutongv1alpha1.UnpackPackage, err.Error(), "unpack package failure")
			_ = p.updateCRStatus()
			return fmt.Errorf("failed to untar %s: %v", p.pkg.Spec.PkgPath, err)
		}
		p.log.Info("handle package unpack success")
		p.updateConditionStatus(wutongv1alpha1.UnpackPackage, wutongv1alpha1.Completed)
		return p.updateCRStatus()
	}

	if p.canPushImage() {
		// Deprecated: No longer download the installation package
		if p.downloadPackage {
			p.log.Info("start load and push images")
			if err := p.imagesLoadAndPush(); err != nil {
				p.updateConditionStatus(wutongv1alpha1.PushImage, wutongv1alpha1.Failed)
				p.updateConditionResion(wutongv1alpha1.PushImage, err.Error(), "load and push images failure")
				_ = p.updateCRStatus()
				return fmt.Errorf("failed to load and push images: %v", err)
			}
		} else {
			p.log.Info("start pull and push images")
			if err := p.imagePullAndPush(); err != nil {
				p.updateConditionStatus(wutongv1alpha1.PushImage, wutongv1alpha1.Failed)
				p.updateConditionResion(wutongv1alpha1.PushImage, err.Error(), "pull and push images failure")
				_ = p.updateCRStatus()
				return fmt.Errorf("failed to pull and push images: %v", err)
			}
		}
		p.log.Info("handle images success")
		p.updateConditionStatus(wutongv1alpha1.PushImage, wutongv1alpha1.Completed)
		return p.updateCRStatus()
	}

	if p.canReady() {
		p.updateConditionStatus(wutongv1alpha1.Ready, wutongv1alpha1.Completed)
		return p.updateCRStatus()
	}
	p.log.V(5).Info("no event can be handle about package")
	return nil
}

func (p *pkg) untartar() error {
	p.log.Info(fmt.Sprintf("start untartaring %s", p.pkg.Spec.PkgPath))
	f, err := os.Open(p.pkg.Spec.PkgPath)
	if f != nil {
		f.Close()
	}
	if err != nil {
		return err
	}
	stop := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(time.Second * 2)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				num := countImages(pkgDst)
				progress := num * 100 / p.totalImageNum
				if p.updateConditionProgress(wutongv1alpha1.UnpackPackage, progress) {
					if err := p.updateCRStatus(); err != nil {
						// ignore error
						p.log.Info(fmt.Sprintf("update number extracted: %v", err))
					}
				}
			case <-stop:
				return
			}
		}
	}()
	_ = os.MkdirAll(pkgDst, os.ModePerm)
	if err := tarutil.Untartar(p.pkg.Spec.PkgPath, pkgDst); err != nil {
		return err
	}
	stop <- struct{}{}
	return nil
}
func (p *pkg) imagePullAndPush() error {
	p.pkg.Status.ImagesNumber = int32(len(p.images))
	p.pkg.Status.ImagesPushed = nil
	var count int32
	handleImgae := func(remoteImage, localImage string) error {
		return retryutil.Retry(time.Second*2, 3, func() (bool, error) {
			image, err := p.containerdCli.ImageService.Get(p.containerdCli.CCtx, remoteImage)
			if err != nil {
				return false, fmt.Errorf("get image %s failure: %v", remoteImage, err)
			}
			image.Name = localImage
			if _, err = p.containerdCli.ImageService.Create(p.containerdCli.CCtx, image); err != nil {
				// If user has specified force and the image already exists then
				// delete the original image and attempt to create the new one
				if errdefs.IsAlreadyExists(err) {
					if err = p.containerdCli.ImageService.Delete(p.containerdCli.CCtx, localImage); err != nil {
						return false, fmt.Errorf("delete image %s failure: %v", localImage, err)
					}
					if _, err = p.containerdCli.ImageService.Create(p.containerdCli.CCtx, image); err != nil {
						return false, fmt.Errorf("create image %s failure: %v", localImage, err)
					}
				} else {
					return false, fmt.Errorf("create image %s failure: %v", localImage, err)
				}
			}
			if err := p.imagePush(image); err != nil {
				return false, fmt.Errorf("push image %s failure %s", localImage, err.Error())
			}
			return true, nil
		})
	}

	for old, new := range p.images {
		remoteImage := path.Join(p.downloadImageDomain, old)
		localImage := path.Join(p.pushImageDomain, new)
		if err := handleImgae(remoteImage, localImage); err != nil {
			return err
		}
		count++
		p.pkg.Status.ImagesPushed = append(p.pkg.Status.ImagesPushed, wutongv1alpha1.WutongPackageImage{Name: localImage})
		progress := count * 100 / p.pkg.Status.ImagesNumber
		if p.updateConditionProgress(wutongv1alpha1.PushImage, progress) {
			if err := p.updateCRStatus(); err != nil {
				return fmt.Errorf("update cr status: %v", err)
			}
		}
		p.log.Info("successfully load image", "image", localImage)
	}
	return nil
}

func (p *pkg) imagesLoadAndPush() error {
	return nil
	// p.pkg.Status.ImagesNumber = countImages(pkgDst)
	// p.pkg.Status.ImagesPushed = nil
	// var count int32
	// walkFn := func(pstr string, info os.FileInfo, err error) error {
	// 	l := p.log.WithValues("file", pstr)
	// 	if err != nil {
	// 		l.Info(fmt.Sprintf("prevent panic by handling failure accessing a path %q: %v\n", pstr, err))
	// 		return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v", pstr, err)
	// 	}
	// 	if !commonutil.IsFile(pstr) {
	// 		return nil
	// 	}
	// 	if !validateFile(pstr) {
	// 		l.Info("invalid file, skip it1")
	// 		return nil
	// 	}

	// 	f := func() (bool, error) {
	// 		image, err := p.imageLoad(pstr)
	// 		if err != nil {
	// 			l.Error(err, "load image")
	// 			return false, fmt.Errorf("load image: %v", err)
	// 		}

	// 		newImage := newImageWithNewDomain(image, wtutil.GetImageRepository(p.cluster))
	// 		if newImage == "" {
	// 			return false, fmt.Errorf("parse image name failure")
	// 		}

	// 		if err := p.dcli.ImageTag(p.ctx, image, newImage); err != nil {
	// 			l.Error(err, "tag image", "source", image, "target", newImage)
	// 			return false, fmt.Errorf("tag image: %v", err)
	// 		}

	// 		if err = p.imagePush(newImage); err != nil {
	// 			l.Error(err, "push image", "image", newImage)
	// 			return false, fmt.Errorf("push image %s: %v", newImage, err)
	// 		}
	// 		count++
	// 		p.pkg.Status.ImagesPushed = append(p.pkg.Status.ImagesPushed, wutongv1alpha1.WutongPackageImage{Name: newImage})
	// 		progress := count * 100 / p.pkg.Status.ImagesNumber
	// 		if p.updateConditionProgress(wutongv1alpha1.PushImage, progress) {
	// 			if err := p.updateCRStatus(); err != nil {
	// 				return false, fmt.Errorf("update cr status: %v", err)
	// 			}
	// 		}
	// 		l.Info("successfully load image", "image", newImage)
	// 		return true, nil
	// 	}

	// 	return retryutil.Retry(1*time.Second, 3, f)
	// }

	// return filepath.Walk(pkgDst, walkFn)
}

func (p *pkg) imageLoad(file string) (string, error) {
	p.log.Info("start loading image", "file", file)
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open file %s: %v", file, err)
	}
	defer f.Close()
	var imageNames []images.Image
	if imageNames, err = p.containerdCli.ContainerdClient.Import(p.containerdCli.CCtx, f); err != nil {
		logrus.Errorf("load image from file %s failure %s", f, err.Error())
	}
	if err != nil {
		return "", fmt.Errorf("path: %s; failed to load images: %v", file, err)
	}
	var imageName string
	imageName = imageNames[0].Name
	if imageName == "" {
		return "", fmt.Errorf("not parse image name")
	}
	p.log.Info("success loading image", "image", imageName)
	return imageName, nil
}

func (p *pkg) imagePush(image images.Image) error {
	p.log.Info("start push image", "image", image.Name)
	defaultTLS := &tls.Config{
		InsecureSkipVerify: true,
	}

	hostOpt := config.HostOptions{}
	hostOpt.DefaultTLS = defaultTLS
	hostOpt.Credentials = func(host string) (string, string, error) {
		return p.cluster.Spec.ImageHub.Username, p.cluster.Spec.ImageHub.Password, nil
	}
	options := cdocker.ResolverOptions{
		Tracker: cdocker.NewInMemoryTracker(),
		Hosts:   config.ConfigureHosts(p.containerdCli.CCtx, hostOpt),
	}
	err := p.containerdCli.ContainerdClient.Push(p.containerdCli.CCtx, image.Name, image.Target, containerd.WithResolver(cdocker.NewResolver(options)))
	if err != nil {
		p.log.Error(err, "failed to push image", "image", image)
		return err
	}
	p.log.Info("success push image", "image", image)
	return nil
}

// EncodeAuthToBase64 serializes the auth configuration as JSON base64 payload
func EncodeAuthToBase64(authConfig dtypes.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func newDockerClient(ctx context.Context) (*dclient.Client, error) {
	// cli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	// if err != nil {
	// 	return nil, fmt.Errorf("create new docker client: %v", err)
	// }
	// cli.NegotiateAPIVersion(ctx)

	// return cli, nil
	return nil, nil
}

func parseImageName(str string) string {
	if !strings.Contains(str, "Loaded image: ") {
		return ""
	}
	str = strings.Replace(str, "Loaded image: ", "", -1)
	str = strings.Replace(str, "\n", "", -1)
	str = trimLatest(str)
	return str
}

func trimLatest(str string) string {
	if !strings.HasSuffix(str, ":latest") {
		return str
	}
	return str[:len(str)-len(":latest")]
}

func countImages(dir string) int32 {
	var count int32
	_ = filepath.Walk(dir, func(pstr string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !commonutil.IsFile(pstr) {
			return nil
		}
		if !validateFile(pstr) {
			return nil
		}
		count++
		return nil
	})

	return count
}

func validateFile(file string) bool {
	base := path.Base(file)
	if path.Ext(base) != ".tgz" || strings.HasPrefix(base, "._") {
		return false
	}
	return true
}

func newImageWithNewDomain(image string, newDomain string) string {
	repo, err := reference.Parse(image)
	if err != nil {
		return ""
	}
	named := repo.(reference.Named)
	remoteName := reference.Path(named)
	tag := "latest"
	if t, ok := repo.(reference.Tagged); ok {
		tag = t.Tag()
	}
	return path.Join(newDomain, remoteName+":"+tag)
}

func (p *pkg) checkIfImageExists(image string) (bool, error) {
	return true, nil
	// repo, err := reference.Parse(image)
	// if err != nil {
	// 	p.log.V(6).Info("parse image", "image", image, "error", err)
	// 	return false, fmt.Errorf("parse image %s: %v", image, err)
	// }
	// named := repo.(reference.Named)
	// tag := "latest"
	// if t, ok := repo.(reference.Tagged); ok {
	// 	tag = t.Tag()
	// }
	// imageFullName := named.Name() + ":" + tag

	// ctx, cancel := context.WithCancel(p.ctx)
	// defer cancel()

	// imageSummarys, err := p.dcli.ImageList(ctx, dtypes.ImageListOptions{
	// 	Filters: filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: imageFullName}),
	// })
	// if err != nil {
	// 	return false, fmt.Errorf("list images: %v", err)
	// }
	// for _, imageSummary := range imageSummarys {
	// 	fmt.Printf("%#v", imageSummary.RepoTags)
	// }

	// _ = imageSummarys

	// return len(imageSummarys) > 0, nil
}

func (p *pkg) isImageRepositoryReady() bool {

	idx, condition := p.cluster.Status.GetCondition(wutongv1alpha1.WutongClusterConditionTypeImageRepository)
	if idx == -1 {
		return false
	}

	if condition.Status != corev1.ConditionTrue {
		return false
	}

	return true
}
