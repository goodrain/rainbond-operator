package rainbondpackage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/pkg/jsonmessage"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/downloadutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	"github.com/goodrain/rainbond-operator/pkg/util/retryutil"
	"github.com/goodrain/rainbond-operator/pkg/util/tarutil"

	"github.com/docker/distribution/reference"
	dtype "github.com/docker/docker/api/types"
	dtypes "github.com/docker/docker/api/types"
	dclient "github.com/docker/docker/client"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_rainbondpackage")
var errorClusterConfigNotReady = fmt.Errorf("cluster config can not be ready")
var errorClusterConfigNoLocalHub = fmt.Errorf("cluster spec not have local image hub info ")
var pkgDst = "/opt/rainbond/pkg/files"

// Add creates a new RainbondPackage Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRainbondPackage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rainbondpackage-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RainbondPackage
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondpackage",
			Namespace: "rbd-system",
		},
	}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRainbondPackage implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRainbondPackage{}

// ReconcileRainbondPackage reconciles a RainbondPackage object
type ReconcileRainbondPackage struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RainbondPackage object and makes changes based on the state read
// and what is in the RainbondPackage.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRainbondPackage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RainbondPackage")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the RainbondPackage instance
	pkg := &rainbondv1alpha1.RainbondPackage{}
	err := r.client.Get(ctx, request.NamespacedName, pkg)
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
	updateStatus, re := checkStatusCanReturn(pkg)
	if updateStatus {
		if err := updateCRStatus(r.client, pkg); err != nil {
			reqLogger.Error(err, "update package status failure ")
			return reconcile.Result{RequeueAfter: time.Second * 5}, nil
		}
		return reconcile.Result{}, nil
	}
	if re != nil {
		return *re, nil
	}
	//need handle condition
	p, err := newpkg(ctx, r.client, pkg, reqLogger)
	if err != nil {
		p.updateConditionStatus(rainbondv1alpha1.Init, rainbondv1alpha1.Failed)
		p.updateConditionResion(rainbondv1alpha1.Init, err.Error(), "create package handle failure")
		p.updateCRStatus()
		reqLogger.Error(err, "create package handle failure ")
		return reconcile.Result{RequeueAfter: time.Second * 5}, nil
	}
	// handle package
	if err = p.handle(); err != nil {
		if err == errorClusterConfigNoLocalHub {
			reqLogger.Info("waiting local image hub ready")
		} else if err == errorClusterConfigNotReady {
			reqLogger.Info("waiting cluster config ready")
		} else {
			reqLogger.Error(err, "failed to handle rainbond package.")
		}
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return reconcile.Result{}, nil
}
func initPackageStatus() *rainbondv1alpha1.RainbondPackageStatus {
	return &rainbondv1alpha1.RainbondPackageStatus{
		Conditions: []rainbondv1alpha1.PackageCondition{
			rainbondv1alpha1.PackageCondition{
				Type:               rainbondv1alpha1.Init,
				Status:             rainbondv1alpha1.Waiting,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			rainbondv1alpha1.PackageCondition{
				Type:               rainbondv1alpha1.DownloadPackage,
				Status:             rainbondv1alpha1.Waiting,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			rainbondv1alpha1.PackageCondition{
				Type:               rainbondv1alpha1.UnpackPackage,
				Status:             rainbondv1alpha1.Waiting,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			rainbondv1alpha1.PackageCondition{
				Type:               rainbondv1alpha1.PushImage,
				Status:             rainbondv1alpha1.Waiting,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
			rainbondv1alpha1.PackageCondition{
				Type:               rainbondv1alpha1.Ready,
				Status:             rainbondv1alpha1.Waiting,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			},
		},
		ImagesPushed: []rainbondv1alpha1.RainbondPackageImage{},
	}
}

//checkStatusCanReturn if pkg status in the working state, straight back
func checkStatusCanReturn(pkg *rainbondv1alpha1.RainbondPackage) (updateStatus bool, re *reconcile.Result) {
	if pkg.Status == nil {
		pkg.Status = initPackageStatus()
		return true, &reconcile.Result{}
	}
	completedCount := 0
	for _, cond := range pkg.Status.Conditions {
		if cond.Status == rainbondv1alpha1.Running {
			return false, &reconcile.Result{}
		}
		//TODO: if have Failed condition, Need to try again?
		if cond.Status == rainbondv1alpha1.Failed {
			return false, &reconcile.Result{}
		}
		if cond.Status == rainbondv1alpha1.Completed {
			completedCount++
		}
	}
	if completedCount == len(pkg.Status.Conditions) {
		return false, &reconcile.Result{}
	}
	return false, nil
}

type pkg struct {
	ctx                 context.Context
	client              client.Client
	dcli                *dclient.Client
	pkg                 *rainbondv1alpha1.RainbondPackage
	cluster             *rainbondv1alpha1.RainbondCluster
	log                 logr.Logger
	downloadPackage     bool
	localPackagePath    string
	downloadPackageURL  string
	downloadPackageMD5  string
	downloadImageDomain string
	pushImageDomain     string
	totalImageNum       int32
	//need download images
	images  map[string]string
	version string
}

func newpkg(ctx context.Context, client client.Client, p *rainbondv1alpha1.RainbondPackage, reqLogger logr.Logger) (*pkg, error) {
	dcli, err := newDockerClient(ctx)
	if err != nil {
		reqLogger.Error(err, "failed to create docker client")
		return nil, err
	}
	version := os.Getenv("RAINBOND_VERSION")
	if version == "" {
		return nil, fmt.Errorf("RAINBOND_VERSION not found")
	}
	pkg := &pkg{
		ctx:           ctx,
		client:        client,
		pkg:           p.DeepCopy(),
		dcli:          dcli,
		totalImageNum: 23,
		images:        make(map[string]string, 23),
		log:           reqLogger,
		version:       version,
	}
	return pkg, nil
}

func (p *pkg) setCluster(c *rainbondv1alpha1.RainbondCluster) error {
	if !c.Spec.ConfigCompleted {
		return errorClusterConfigNotReady
	}
	if c.Spec.ImageHub == nil || c.Spec.ImageHub.Domain == "" {
		return errorClusterConfigNoLocalHub
	}
	if c.Spec.InstallVersion != "" {
		p.version = c.Spec.InstallVersion
	}
	p.localPackagePath = p.pkg.Spec.PkgPath
	p.downloadPackageURL = c.Spec.InstallPackageConfig.URL
	p.downloadPackageMD5 = c.Spec.InstallPackageConfig.MD5
	p.cluster = c
	p.images = map[string]string{
		"/rbd-api:" + p.version:             "/rbd-api:" + p.version,
		"/rbd-app-ui:" + p.version:          "/rbd-app-ui:" + p.version,
		"/rbd-eventlog:" + p.version:        "/rbd-eventlog:" + p.version,
		"/rbd-chaos:" + p.version:           "/rbd-chaos:" + p.version,
		"/rbd-mq:" + p.version:              "/rbd-mq:" + p.version,
		"/rbd-webcli:" + p.version:          "/rbd-webcli:" + p.version,
		"/rbd-worker:" + p.version:          "/rbd-worker:" + p.version,
		"/rbd-monitor:" + p.version:         "/rbd-monitor:" + p.version,
		"/rbd-grctl:" + p.version:           "/rbd-grctl:" + p.version,
		"/rbd-node:" + p.version:            "/rbd-node:" + p.version,
		"/rbd-gateway:" + p.version:         "/rbd-gateway:" + p.version,
		"/builder:5.2.0":                    "/builder",
		"/runner":                           "/runner",
		"/kube-state-metrics":               "/kube-state-metrics",
		"/mysqld-exporter":                  "/mysqld-exporter",
		"/rbd-repo:6.16.0":                  "/rbd-repo:6.16.0",
		"/rbd-registry:2.6.2":               "/rbd-registry:2.6.2",
		"/rbd-db:8.0.12":                    "/rbd-db:8.0.12",
		"/metrics-server:v0.3.6":            "/metrics-server:v0.3.6",
		"/rbd-init-probe:" + p.version:      "/rbd-init-probe",
		"/rbd-mesh-data-panel:" + p.version: "/rbd-mesh-data-panel",
		"/plugins-tcm:5.1.7":                "/tcm",
		"/nfs-provisioner:v2.3.0":           "/nfs-provisioner:v2.3.0",
		"/etcd:v3.3.18":                     "/etcd:v3.3.18",
	}
	return nil
}

func (p *pkg) updateCRStatus() error {
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return updateCRStatus(p.client, p.pkg)
	}); err != nil {
		log.Error(err, "update rainbondpackage")
		return err
	}
	return nil
}

func updateCRStatus(client client.Client, pkg *rainbondv1alpha1.RainbondPackage) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return client.Status().Update(ctx, pkg)
	}); err != nil {
		return fmt.Errorf("failed to update rainbondpackage status: %v", err)
	}
	return nil
}

func (p *pkg) checkClusterConfig() error {
	cluster := &rainbondv1alpha1.RainbondCluster{}
	ctx, cancel := context.WithTimeout(p.ctx, time.Second*5)
	defer cancel()
	if err := p.client.Get(ctx, types.NamespacedName{Namespace: p.pkg.Namespace, Name: "rainbondcluster"}, cluster); err != nil {
		p.log.Error(err, "failed to get rainbondcluster.")
		return err
	}
	if err := p.setCluster(cluster); err != nil {
		return err
	}
	switch cluster.Spec.InstallMode {
	case rainbondv1alpha1.InstallationModeWithPackage:
		p.downloadPackage = true
	default:
		p.downloadPackage = false
		p.downloadImageDomain = cluster.Spec.RainbondImageRepository
		if p.downloadImageDomain == "" {
			p.downloadImageDomain = "rainbond"
		}
		if cluster.Spec.ImageHub != nil {
			p.pushImageDomain = cluster.Spec.ImageHub.Domain
			if cluster.Spec.ImageHub.Namespace != "" {
				p.pushImageDomain += "/" + cluster.Spec.ImageHub.Namespace
			}
		}
		if p.pushImageDomain == "" {
			p.pushImageDomain = "goodrain.me"
		}
	}
	return nil
}

func (p *pkg) findCondition(typ3 rainbondv1alpha1.PackageConditionType) *rainbondv1alpha1.PackageCondition {
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			return &p.pkg.Status.Conditions[i]
		}
	}
	return nil
}
func (p *pkg) updateConditionStatus(typ3 rainbondv1alpha1.PackageConditionType, status rainbondv1alpha1.PackageConditionStatus) {
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			if p.pkg.Status.Conditions[i].Status != status {
				p.pkg.Status.Conditions[i].LastTransitionTime = metav1.Now()
			}
			p.pkg.Status.Conditions[i].LastHeartbeatTime = metav1.Now()
			p.pkg.Status.Conditions[i].Status = status
			if status == rainbondv1alpha1.Completed {
				p.pkg.Status.Conditions[i].Progress = 100
				p.pkg.Status.Conditions[i].Reason = ""
				p.pkg.Status.Conditions[i].Message = ""
			}
			break
		}
	}
}
func (p *pkg) updateConditionResion(typ3 rainbondv1alpha1.PackageConditionType, resion, message string) {
	for i, condition := range p.pkg.Status.Conditions {
		if condition.Type == typ3 {
			p.pkg.Status.Conditions[i].LastHeartbeatTime = metav1.Now()
			p.pkg.Status.Conditions[i].Reason = resion
			p.pkg.Status.Conditions[i].Message = message
			break
		}
	}
}
func (p *pkg) updateConditionProgress(typ3 rainbondv1alpha1.PackageConditionType, progress int32) bool {
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
func (p *pkg) completeCondition(con *rainbondv1alpha1.PackageCondition) error {
	if con == nil {
		return nil
	}
	p.updateConditionStatus(con.Type, rainbondv1alpha1.Completed)
	return p.updateCRStatus()
}
func (p *pkg) runningCondition(con *rainbondv1alpha1.PackageCondition) error {
	if con == nil {
		return nil
	}
	p.updateConditionStatus(con.Type, rainbondv1alpha1.Running)
	return p.updateCRStatus()
}
func (p *pkg) canDownload() bool {
	if con := p.findCondition(rainbondv1alpha1.DownloadPackage); con != nil {
		if con.Status == rainbondv1alpha1.Waiting {
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
	if con := p.findCondition(rainbondv1alpha1.DownloadPackage); con != nil {
		//Must conditions are not met
		if con.Status != rainbondv1alpha1.Completed {
			return false
		}
		if uncon := p.findCondition(rainbondv1alpha1.UnpackPackage); uncon != nil {
			//status is waiting
			if uncon.Status == rainbondv1alpha1.Waiting {
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
	if uncon := p.findCondition(rainbondv1alpha1.UnpackPackage); uncon != nil {
		if uncon.Status != rainbondv1alpha1.Completed {
			return false
		}
		if pcon := p.findCondition(rainbondv1alpha1.PushImage); pcon != nil {
			if pcon.Status == rainbondv1alpha1.Waiting {
				if err := p.runningCondition(pcon); err != nil {
					p.log.Error(err, "running push image condition failure %s")
				}
				return true
			}
		}
	}
	return false
}

func (p *pkg) canReady() bool {
	if pcon := p.findCondition(rainbondv1alpha1.PushImage); pcon != nil && pcon.Status == rainbondv1alpha1.Completed {
		return true
	}
	return false
}
func (p *pkg) setInitStatus() error {
	if con := p.findCondition(rainbondv1alpha1.Init); con != nil {
		if con.Status != rainbondv1alpha1.Completed {
			con.Status = rainbondv1alpha1.Completed
			if err := p.updateCRStatus(); err != nil {
				p.log.Error(err, "failed to update rainbondpackage status.")
				return err
			}
		}
	}
	return nil
}

//donwnloadPackage download package
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
			p.log.Info("rainbond package file is exists")
			return nil
		}
	}
	p.log.Info("rainbond package file does not exists, downloading background ...")
	var stop = make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-ticker.C:
				progress := downloadListener.Percent
				//Make time for later in the download process
				realProgress := int32(progress) - int32(float64(progress)*0.05)
				if p.updateConditionProgress(rainbondv1alpha1.DownloadPackage, realProgress) {
					if err := p.updateCRStatus(); err != nil {
						// ignore error
						log.Info("update number extracted: %v", err)
					}
				}
			case <-stop:
				return
			}
		}
	}()
	if err := downloadListener.Download(); err != nil {
		p.log.Error(err, "download rainbond package error, will retry")
		p.updateConditionResion(rainbondv1alpha1.Init, err.Error(), "download rainbond package error, will retry")
		p.updateCRStatus()
		err = downloadListener.Download()
		if err != nil {
			logrus.Error(err, "download rainbond package error, not retry")
			return err
		}
	}
	//stop watch progress
	stop <- struct{}{}
	p.log.Info(fmt.Sprintf("success download package from %s", p.downloadPackageURL))
	return nil
}

//handle
func (p *pkg) handle() error {
	p.log.Info("start handling rainbond package.")
	// check prerequisites
	if err := p.checkClusterConfig(); err != nil {
		p.log.V(6).Info(fmt.Sprintf("check cluster config: %v", err))
		//To continue waiting
		if err == errorClusterConfigNotReady || err == errorClusterConfigNoLocalHub {
			return err
		}
		p.updateConditionStatus(rainbondv1alpha1.Init, rainbondv1alpha1.Waiting)
		p.updateConditionResion(rainbondv1alpha1.Init, err.Error(), "get rainbond cluster config failure")
		p.updateCRStatus()
		return err
	}
	//update init condition status is complete
	if err := p.setInitStatus(); err != nil {
		p.log.Error(err, "set init status")
		p.updateConditionStatus(rainbondv1alpha1.Init, rainbondv1alpha1.Failed)
		p.updateConditionResion(rainbondv1alpha1.Init, err.Error(), "set init status failure")
		p.updateCRStatus()
		return err
	}
	if p.canDownload() {
		p.updateConditionStatus(rainbondv1alpha1.DownloadPackage, rainbondv1alpha1.Running)
		p.updateCRStatus()
		//download pkg
		if err := p.donwnloadPackage(); err != nil {
			p.log.Error(err, "download package")
			p.updateConditionStatus(rainbondv1alpha1.DownloadPackage, rainbondv1alpha1.Failed)
			p.updateConditionResion(rainbondv1alpha1.DownloadPackage, err.Error(), "download package failure")
			p.updateCRStatus()
			return fmt.Errorf("failed to download package %s", err.Error())
		}
		p.log.Info("handle downlaod package success")
		p.updateConditionStatus(rainbondv1alpha1.DownloadPackage, rainbondv1alpha1.Completed)
		return p.updateCRStatus()
	}

	if p.canUnpack() {
		p.updateConditionStatus(rainbondv1alpha1.UnpackPackage, rainbondv1alpha1.Running)
		p.updateCRStatus()
		//unstar the installation package
		if err := p.untartar(); err != nil {
			p.updateConditionStatus(rainbondv1alpha1.UnpackPackage, rainbondv1alpha1.Failed)
			p.updateConditionResion(rainbondv1alpha1.UnpackPackage, err.Error(), "unpack package failure")
			p.updateCRStatus()
			return fmt.Errorf("failed to untar %s: %v", p.pkg.Spec.PkgPath, err)
		}
		p.log.Info("handle package unpack success")
		p.updateConditionStatus(rainbondv1alpha1.UnpackPackage, rainbondv1alpha1.Completed)
		return p.updateCRStatus()
	}

	if p.canPushImage() {
		p.updateConditionStatus(rainbondv1alpha1.PushImage, rainbondv1alpha1.Running)
		p.updateCRStatus()
		if p.downloadPackage {
			p.log.Info("start load and push images")
			if err := p.imagesLoadAndPush(); err != nil {
				p.updateConditionStatus(rainbondv1alpha1.PushImage, rainbondv1alpha1.Failed)
				p.updateConditionResion(rainbondv1alpha1.PushImage, err.Error(), "load and push images failure")
				p.updateCRStatus()
				return fmt.Errorf("failed to load and push images: %v", err)
			}
		} else {
			p.log.Info("start pull and push images")
			if err := p.imagePullAndPush(); err != nil {
				p.updateConditionStatus(rainbondv1alpha1.PushImage, rainbondv1alpha1.Failed)
				p.updateConditionResion(rainbondv1alpha1.PushImage, err.Error(), "pull and push images failure")
				p.updateCRStatus()
				return fmt.Errorf("failed to pull and push images: %v", err)
			}
		}
		p.log.Info("handle images success")
		p.updateConditionStatus(rainbondv1alpha1.PushImage, rainbondv1alpha1.Completed)
		return p.updateCRStatus()
	}

	if p.canReady() {
		p.updateConditionStatus(rainbondv1alpha1.Ready, rainbondv1alpha1.Completed)
		return p.updateCRStatus()
	}

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
				if p.updateConditionProgress(rainbondv1alpha1.UnpackPackage, progress) {
					if err := p.updateCRStatus(); err != nil {
						// ignore error
						log.Info("update number extracted: %v", err)
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
		if err := p.imagePull(remoteImage); err != nil {
			return fmt.Errorf("pull image %s failure %s", remoteImage, err.Error())
		}
		if err := p.dcli.ImageTag(p.ctx, remoteImage, localImage); err != nil {
			return fmt.Errorf("change image tag(%s => %s) failure: %v", remoteImage, localImage, err)
		}
		if err := p.imagePush(localImage); err != nil {
			return fmt.Errorf("push image %s failure %s", localImage, err.Error())
		}
		return nil
	}

	for old, new := range p.images {
		remoteImage := path.Join(p.downloadImageDomain, old)
		localImage := path.Join(p.pushImageDomain, new)
		if err := handleImgae(remoteImage, localImage); err != nil {
			err = handleImgae(remoteImage, localImage)
			if err != nil {
				return err
			}
		}
		count++
		p.pkg.Status.ImagesPushed = append(p.pkg.Status.ImagesPushed, rainbondv1alpha1.RainbondPackageImage{Name: localImage})
		progress := count * 100 / p.pkg.Status.ImagesNumber
		if p.updateConditionProgress(rainbondv1alpha1.PushImage, progress) {
			if err := p.updateCRStatus(); err != nil {
				return fmt.Errorf("update cr status: %v", err)
			}
		}
		p.log.Info("successfully load image", "image", localImage)
	}
	return nil
}
func (p *pkg) imagesLoadAndPush() error {
	p.pkg.Status.ImagesNumber = countImages(pkgDst)
	p.pkg.Status.ImagesPushed = nil
	var count int32
	walkFn := func(pstr string, info os.FileInfo, err error) error {
		l := log.WithValues("file", pstr)
		if err != nil {
			l.Info(fmt.Sprintf("prevent panic by handling failure accessing a path %q: %v\n", pstr, err))
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v", pstr, err)
		}
		if !commonutil.IsFile(pstr) {
			return nil
		}
		if !validateFile(pstr) {
			l.Info("invalid file, skip it1")
			return nil
		}

		f := func() (bool, error) {
			image, err := p.imageLoad(pstr)
			if err != nil {
				l.Error(err, "load image")
				return false, fmt.Errorf("load image: %v", err)
			}

			newImage := newImageWithNewDomain(image, rbdutil.GetImageRepository(p.cluster))
			if newImage == "" {
				return false, fmt.Errorf("parse image name failure")
			}

			if err := p.dcli.ImageTag(p.ctx, image, newImage); err != nil {
				l.Error(err, "tag image", "source", image, "target", newImage)
				return false, fmt.Errorf("tag image: %v", err)
			}

			if err = p.imagePush(newImage); err != nil {
				l.Error(err, "push image", "image", newImage)
				return false, fmt.Errorf("push image %s: %v", newImage, err)
			}
			count++
			p.pkg.Status.ImagesPushed = append(p.pkg.Status.ImagesPushed, rainbondv1alpha1.RainbondPackageImage{Name: newImage})
			progress := count * 100 / p.pkg.Status.ImagesNumber
			if p.updateConditionProgress(rainbondv1alpha1.PushImage, progress) {
				if err := p.updateCRStatus(); err != nil {
					return false, fmt.Errorf("update cr status: %v", err)
				}
			}
			l.Info("successfully load image", "image", newImage)
			return true, nil
		}

		return retryutil.Retry(1*time.Second, 3, f)
	}

	return filepath.Walk(pkgDst, walkFn)
}

func (p *pkg) imageLoad(file string) (string, error) {
	log.Info("start loading image", "file", file)
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open file %s: %v", file, err)
	}
	defer f.Close()
	res, err := p.dcli.ImageLoad(p.ctx, f, true) // load one, push one.
	if err != nil {
		return "", fmt.Errorf("path: %s; failed to load images: %v", file, err)
	}
	var imageName string
	if res.Body != nil {
		defer res.Body.Close()
		dec := json.NewDecoder(res.Body)
		for {
			select {
			case <-p.ctx.Done():
				log.Error(p.ctx.Err(), "error form context")
				return "", p.ctx.Err()
			default:
			}
			var jm jsonmessage.JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				return "", fmt.Errorf("failed to decode json message: %v", err)
			}
			if jm.Error != nil {
				return "", fmt.Errorf("error detail: %v", jm.Error)
			}
			msg := jm.Stream
			//the domain of image in package is goodrain.me,not need change tag
			image := parseImageName(msg)
			if image != "" {
				imageName = image
			}
			log.Info("response from image loading", "msg", msg)
		}
	}
	if imageName == "" {
		return "", fmt.Errorf("not parse image name")
	}
	p.log.Info("success loading image", "image", imageName)
	return imageName, nil
}

func (p *pkg) imagePush(image string) error {
	p.log.Info("start push image", "image", image)
	var opts dtypes.ImagePushOptions
	authConfig := dtypes.AuthConfig{
		ServerAddress: rbdutil.GetImageRepository(p.cluster),
	}
	authConfig.Username = p.cluster.Spec.ImageHub.Username
	authConfig.Password = p.cluster.Spec.ImageHub.Password

	registryAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("failed to encode auth config: %v", err)
	}
	opts.RegistryAuth = registryAuth
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()
	var res io.ReadCloser
	for i := 0; i < 2; i++ {
		res, err = p.dcli.ImagePush(ctx, image, opts)
		if err != nil {
			p.log.Error(err, "failed to push image, retry after 5 second", "image", image)
			//retry after 5 second
			time.Sleep(time.Second * 5)
			continue
		}
	}
	if err != nil {
		return err
	}
	if res != nil {
		defer res.Close()

		dec := json.NewDecoder(res)
		for {
			select {
			case <-ctx.Done():
				log.Error(p.ctx.Err(), "error form context")
				return p.ctx.Err()
			default:
			}
			var jm jsonmessage.JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("failed to decode json message: %v", err)
			}
			if jm.Error != nil {
				return fmt.Errorf("error detail: %v", jm.Error)
			}
			log.V(5).Info("response from image pushing", "msg", jm.Stream)
		}
	}
	p.log.Info("success push image", "image", image)
	return nil
}

func (p *pkg) imagePull(image string) error {
	p.log.Info("start pull image", "image", image)
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()
	rf, err := reference.ParseAnyReference(image)
	if err != nil {
		logrus.Errorf("reference image error: %s", err.Error())
		return err
	}
	res, err := p.dcli.ImagePull(ctx, rf.String(), dtype.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s failure %s", image, err.Error())
	}
	if res != nil {
		defer res.Close()
		dec := json.NewDecoder(res)
		for {
			select {
			case <-ctx.Done():
				p.log.Error(ctx.Err(), "error form context")
				return ctx.Err()
			default:
			}
			var jm jsonmessage.JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("failed to decode json message: %v", err)
			}
			if jm.Error != nil {
				return fmt.Errorf("error detail: %v", jm.Error)
			}
			p.log.V(5).Info("response from image pushing", "msg", jm.Stream)
		}
	}
	p.log.Info("success pull image", "image", image)
	return nil
}

func newDockerClient(ctx context.Context) (*dclient.Client, error) {
	cli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		log.Error(err, "create new docker client")
		return nil, fmt.Errorf("create new docker client: %v", err)
	}
	cli.NegotiateAPIVersion(ctx)

	return cli, nil
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

func encodeAuthToBase64(authConfig dtypes.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func trimLatest(str string) string {
	if !strings.HasSuffix(str, ":latest") {
		return str
	}
	return str[:len(str)-len(":latest")]
}

func countImages(dir string) int32 {
	l := log.WithName("count images")
	var count int32
	_ = filepath.Walk(dir, func(pstr string, info os.FileInfo, err error) error {
		if err != nil {
			l.Info(fmt.Sprintf("walk path %s: %v", pstr, err))
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
		log.Error(err, "parse image name failure", "imagename", image)
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
