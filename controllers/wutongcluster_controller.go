package controllers

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong-operator/util/commonutil"
	"github.com/wutong-paas/wutong-operator/util/retryutil"
	"github.com/wutong-paas/wutong-operator/util/suffixdomain"
	"github.com/wutong-paas/wutong-operator/util/wtutil"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"github.com/juju/errors"
	"github.com/wutong-paas/wutong-operator/api/v1alpha1"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	clustermgr "github.com/wutong-paas/wutong-operator/controllers/cluster-mgr"
	"github.com/wutong-paas/wutong-operator/util/constants"
	"github.com/wutong-paas/wutong-operator/util/uuidutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// RegionArchAmd64 -
	RegionArchAmd64 = "amd64" // default
	// RegionArchArm64 -
	RegionArchArm64 = "arm64"
)

// WutongClusterReconciler reconciles a WutongCluster object
type WutongClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=wutong.io,resources=wutongclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wutong.io,resources=wutongclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wutong.io,resources=wutongclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WutongCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *WutongClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("wutongcluster", request.NamespacedName)

	// Fetch the WutongCluster instance
	wutongcluster := &wutongv1alpha1.WutongCluster{}
	err := r.Client.Get(ctx, request.NamespacedName, wutongcluster)
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

	mgr := clustermgr.NewClusterMgr(ctx, r.Client, reqLogger, wutongcluster, r.Scheme)

	// generate status for wutong cluster
	reqLogger.V(6).Info("start generate status")
	status, err := mgr.GenerateWutongClusterStatus()
	if err != nil {
		reqLogger.Error(err, "failed to generate wutongcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		rc := &wutongv1alpha1.WutongCluster{}
		if err := r.Client.Get(ctx, request.NamespacedName, rc); err != nil {
			return err
		}
		rc.Status = *status
		return r.Client.Status().Update(ctx, rc)
	}); err != nil {
		reqLogger.Error(err, "update wutongcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}
	reqLogger.V(6).Info("update status success")

	// setup imageHub if empty
	if wutongcluster.Spec.ImageHub == nil {
		reqLogger.V(6).Info("create new image hub info")
		imageHub, err := r.getImageHub(wutongcluster)
		if err != nil {
			reqLogger.V(6).Info(fmt.Sprintf("set image hub info: %v", err))
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			rc := &wutongv1alpha1.WutongCluster{}
			if err := r.Client.Get(ctx, request.NamespacedName, rc); err != nil {
				return err
			}
			rc.Spec.ImageHub = imageHub
			wutongcluster = rc
			return r.Client.Update(ctx, rc)
		}); err != nil {
			reqLogger.Error(err, "update wutongcluster")
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
		reqLogger.V(6).Info("create new image hub info success")
		// Put it back in the queue.
		return reconcile.Result{Requeue: true}, err
	}

	if wutongcluster.Spec.SuffixHTTPHost == "" {
		var ip string
		if len(wutongcluster.Spec.NodesForGateway) > 0 {
			ip = wutongcluster.Spec.NodesForGateway[0].InternalIP
		}
		if len(wutongcluster.Spec.GatewayIngressIPs) > 0 && wutongcluster.Spec.GatewayIngressIPs[0] != "" {
			ip = wutongcluster.Spec.GatewayIngressIPs[0]
		}
		if ip != "" {
			err := retryutil.Retry(1*time.Second, 3, func() (bool, error) {
				domain, err := r.genSuffixHTTPHost(ip, wutongcluster)
				if err != nil {
					return false, err
				}
				wutongcluster.Spec.SuffixHTTPHost = domain
				if !strings.HasSuffix(domain, constants.DefHTTPDomainSuffix) {
					wutongcluster.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
				}
				return true, nil
			})
			if err != nil {
				logrus.Warningf("generate suffix http host: %v", err)
				wutongcluster.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
			}
			return reconcile.Result{}, r.Client.Update(ctx, wutongcluster)
		}
		logrus.Infof("wutongcluster.Spec.SuffixHTTPHost ip is empty %s", ip)
		wutongcluster.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
		return reconcile.Result{}, r.Client.Update(ctx, wutongcluster)
	}

	// create secret for pulling images.
	if wutongcluster.Spec.ImageHub != nil && wutongcluster.Spec.ImageHub.Username != "" && wutongcluster.Spec.ImageHub.Password != "" {
		err := mgr.CreateImagePullSecret()
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, con := range wutongcluster.Status.Conditions {
		if con.Status != corev1.ConditionTrue {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	r.ReconcileLightweightInstall(ctx, wutongcluster)

	if err := r.createWutongVolumes(wutongcluster); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("create wutong volume failure %s", err.Error())
	}
	if err := r.createWutongPackage(); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("create wutong package failure %s", err.Error())
	}
	if err := r.createComponents(wutongcluster); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("create components failure %s", err.Error())
	}

	return ctrl.Result{}, nil
}

// ReconcileLightweightInstall -
func (r *WutongClusterReconciler) ReconcileLightweightInstall(ctx context.Context, wutongcluster *wutongv1alpha1.WutongCluster) {
	if wutongcluster.Spec.Lightweight {
		if !wutongcluster.Spec.OptionalComponent.MetricsServer {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "metrics-server", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}

		if !wutongcluster.Spec.OptionalComponent.WutongGateway {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "wt-gateway", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}

		if !wutongcluster.Spec.OptionalComponent.WutongMonitor {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "wt-monitor", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}

		if !wutongcluster.Spec.OptionalComponent.WutongNode {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "wt-node", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}

		if !wutongcluster.Spec.OptionalComponent.WutongResourceProxy {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "wt-resource-proxy", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}

		if !wutongcluster.Spec.OptionalComponent.WutongEventLog {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "wt-eventlog", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}

		if !wutongcluster.Spec.OptionalComponent.WutongWebcli {
			var comp wutongv1alpha1.WutongComponent
			err := r.Client.Get(ctx, types.NamespacedName{Name: "wt-webcli", Namespace: constants.WutongSystemNamespace}, &comp)
			if err == nil {
				_ = r.Client.Delete(ctx, &comp, &client.DeleteOptions{})
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WutongClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wutongv1alpha1.WutongCluster{}).
		Complete(r)
}

func (r *WutongClusterReconciler) getImageHub(cluster *wutongv1alpha1.WutongCluster) (*wutongv1alpha1.ImageHub, error) {
	return &wutongv1alpha1.ImageHub{
		Domain:   constants.DefImageRepository,
		Username: "admin",
		Password: uuidutil.NewUUID()[0:8],
	}, nil
}

func (r *WutongClusterReconciler) genSuffixHTTPHost(ip string, wutongcluster *wutongv1alpha1.WutongCluster) (domain string, err error) {
	id, auth, err := r.getOrCreateUUIDAndAuth(wutongcluster)
	if err != nil {
		return "", err
	}
	domain, err = suffixdomain.GenerateDomain(ip, id, auth)
	if err != nil {
		return "", err
	}
	return domain, nil
}

func (r *WutongClusterReconciler) getOrCreateUUIDAndAuth(wutongcluster *wutongv1alpha1.WutongCluster) (id, auth string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	cm := &corev1.ConfigMap{}
	err = r.Client.Get(context.Background(), types.NamespacedName{Name: "wt-suffix-host", Namespace: wutongcluster.Namespace}, cm)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return "", "", err
	}
	if err != nil && strings.Contains(err.Error(), "not found") {
		logrus.Info("not found configmap wt-suffix-host, create it")
		cm = suffixdomain.GenerateSuffixConfigMap("wt-suffix-host", wutongcluster.Namespace)
		if err = r.Client.Create(ctx, cm); err != nil {
			return "", "", err
		}
	}
	return cm.Data["uuid"], cm.Data["auth"], nil
}

type componentClaim struct {
	namespace       string
	name            string
	version         string
	imageRepository string
	imageName       string
	Configs         map[string]string
	envs            map[string]string
	isInit          bool
	replicas        *int32
	limitCPU        string
	limitMemory     string
}

func (c *componentClaim) image() string {
	return path.Join(c.imageRepository, c.imageName) + ":" + c.version
}

func (r *WutongClusterReconciler) createComponents(cluster *wutongv1alpha1.WutongCluster) error {
	claims := r.genComponentClaims(cluster)
	for _, claim := range claims {
		// update image repository for priority components
		claim.imageRepository = cluster.Spec.WutongImageRepository
		data := r.parseComponentClaim(claim)
		// init component
		data.Namespace = constants.WutongSystemNamespace

		err := retryutil.Retry(time.Second*2, 3, func() (bool, error) {
			if err := r.createResourceIfNotExists(data); err != nil {
				return false, err
			}
			return true, nil
		})
		if err != nil {
			return fmt.Errorf("create wutong component %s failure %s", data.GetName(), err.Error())
		}
	}
	return nil
}

func (r *WutongClusterReconciler) parseComponentClaim(claim *componentClaim) *wutongv1alpha1.WutongComponent {
	component := &v1alpha1.WutongComponent{}
	component.Namespace = claim.namespace
	component.Name = claim.name
	component.Spec.Image = claim.image()
	component.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	component.Spec.Replicas = claim.replicas
	if claim.envs != nil {
		for k, v := range claim.envs {
			component.Spec.Env = append(component.Spec.Env, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	if claim.limitCPU != "" {
		limitCPU, err := resource.ParseQuantity(claim.limitCPU)
		if err != nil {
			logrus.Errorf("parse cpu limit %s failure %s", claim.limitCPU, err.Error())
		}
		if component.Spec.Resources.Limits == nil {
			component.Spec.Resources.Limits = corev1.ResourceList{}
		}
		component.Spec.Resources.Limits[corev1.ResourceCPU] = limitCPU
	}
	if claim.limitMemory != "" {
		limitMemory, err := resource.ParseQuantity(claim.limitMemory)
		if err != nil {
			logrus.Errorf("parse memory limit %s failure %s", claim.limitMemory, err.Error())
		}
		if component.Spec.Resources.Limits == nil {
			component.Spec.Resources.Limits = corev1.ResourceList{}
		}
		component.Spec.Resources.Limits[corev1.ResourceMemory] = limitMemory
	}
	labels := wtutil.LabelsForWutong(map[string]string{"name": claim.name})
	if claim.isInit {
		component.Spec.PriorityComponent = true
		labels["priorityComponent"] = "true"
	}
	component.Labels = labels
	return component
}

func (r *WutongClusterReconciler) genComponentClaims(cluster *v1alpha1.WutongCluster) map[string]*componentClaim {
	var defReplicas = commonutil.Int32(1)
	if cluster.Spec.EnableHA {
		defReplicas = commonutil.Int32(2)
	}

	var isInit bool
	imageRepository := constants.DefImageRepository
	if cluster.Spec.ImageHub == nil || cluster.Spec.ImageHub.Domain == constants.DefImageRepository {
		isInit = true
	} else {
		imageRepository = path.Join(cluster.Spec.ImageHub.Domain, cluster.Spec.ImageHub.Namespace)
	}

	newClaim := func(name string) *componentClaim {
		defClaim := componentClaim{name: name, imageRepository: imageRepository, version: cluster.Spec.InstallVersion, replicas: defReplicas}
		defClaim.imageName = name
		return &defClaim
	}
	name2Claim := map[string]*componentClaim{
		"wt-api":            newClaim("wt-api"),
		"wt-chaos":          newClaim("wt-chaos"),
		"wt-eventlog":       newClaim("wt-eventlog"),
		"wt-monitor":        newClaim("wt-monitor"),
		"wt-mq":             newClaim("wt-mq"),
		"wt-worker":         newClaim("wt-worker"),
		"wt-webcli":         newClaim("wt-webcli"),
		"wt-resource-proxy": newClaim("wt-resource-proxy"),
	}

	name2Claim["wt-eventlog"].limitCPU = "500m"
	name2Claim["wt-eventlog"].limitMemory = "4Gi"

	name2Claim["wt-monitor"].limitCPU = "500m"
	name2Claim["wt-monitor"].limitMemory = "4Gi"

	name2Claim["wt-chaos"].envs = map[string]string{
		"CI_VERSION": cluster.Spec.InstallVersion,
	}
	name2Claim["wt-worker"].envs = map[string]string{
		"CI_VERSION": cluster.Spec.InstallVersion,
	}
	name2Claim["metrics-server"] = newClaim("metrics-server")
	name2Claim["metrics-server"].version = "v0.6.1"

	if cluster.Spec.RegionDatabase == nil {
		claim := newClaim("wt-db")
		claim.imageName = "mysql"
		claim.version = "8.0"
		claim.replicas = commonutil.Int32(1)
		name2Claim["wt-db"] = claim
	}

	if cluster.Spec.ImageHub == nil || cluster.Spec.ImageHub.Domain == constants.DefImageRepository {
		claim := newClaim("wt-hub")
		claim.imageName = "registry"
		claim.version = "2.6.2"
		claim.isInit = isInit
		name2Claim["wt-hub"] = claim
	}

	name2Claim["wt-gateway"] = newClaim("wt-gateway")
	name2Claim["wt-gateway"].isInit = isInit
	name2Claim["wt-node"] = newClaim("wt-node")
	name2Claim["wt-node"].isInit = isInit

	name2Claim["wt-node"].limitCPU = "250m"
	name2Claim["wt-node"].limitMemory = "2Gi"

	if cluster.Spec.EtcdConfig == nil || len(cluster.Spec.EtcdConfig.Endpoints) == 0 {
		claim := newClaim("wt-etcd")
		claim.imageName = "etcd"
		claim.version = imageFitArch("v3.3.18", cluster.Spec.Arch)
		claim.isInit = isInit
		if cluster.Spec.EnableHA {
			claim.replicas = commonutil.Int32(3)
		}
		name2Claim["wt-etcd"] = claim
	}

	if rwx := cluster.Spec.WutongVolumeSpecRWX; rwx != nil && rwx.CSIPlugin != nil {
		if rwx.CSIPlugin.NFS != nil {
			name2Claim["nfs-provisioner"] = newClaim("nfs-provisioner")
			name2Claim["nfs-provisioner"].version = "v3.0.0"
			name2Claim["nfs-provisioner"].replicas = commonutil.Int32(1)
			name2Claim["nfs-provisioner"].isInit = isInit
		}
		if rwx.CSIPlugin.AliyunNas != nil {
			name2Claim[constants.AliyunCSINasPlugin] = newClaim(constants.AliyunCSINasPlugin)
			name2Claim[constants.AliyunCSINasPlugin].isInit = isInit
			name2Claim[constants.AliyunCSINasProvisioner] = newClaim(constants.AliyunCSINasProvisioner)
			name2Claim[constants.AliyunCSINasProvisioner].isInit = isInit
			name2Claim[constants.AliyunCSINasProvisioner].replicas = commonutil.Int32(1)
		}
	}
	if rwo := cluster.Spec.WutongVolumeSpecRWO; rwo != nil && rwo.CSIPlugin != nil {
		if rwo.CSIPlugin.AliyunCloudDisk != nil {
			name2Claim[constants.AliyunCSIDiskPlugin] = newClaim(constants.AliyunCSIDiskPlugin)
			name2Claim[constants.AliyunCSIDiskPlugin].isInit = isInit
			name2Claim[constants.AliyunCSIDiskProvisioner] = newClaim(constants.AliyunCSIDiskProvisioner)
			name2Claim[constants.AliyunCSIDiskProvisioner].isInit = isInit
			name2Claim[constants.AliyunCSIDiskProvisioner].replicas = commonutil.Int32(1)
		}
	}

	if cluster.Spec.Lightweight {
		if !cluster.Spec.OptionalComponent.WutongMonitor {
			delete(name2Claim, "wt-monitor")
		}
		if !cluster.Spec.OptionalComponent.WutongNode {
			delete(name2Claim, "wt-node")
		}
		if !cluster.Spec.OptionalComponent.WutongWebcli {
			delete(name2Claim, "wt-webcli")
		}
		if !cluster.Spec.OptionalComponent.MetricsServer {
			delete(name2Claim, "metrics-server")
		}
		if !cluster.Spec.OptionalComponent.WutongResourceProxy {
			delete(name2Claim, "wt-resource-proxy")
		}
		if !cluster.Spec.OptionalComponent.WutongEventLog {
			delete(name2Claim, "wt-eventlog")
		}
		if !cluster.Spec.OptionalComponent.WutongGateway {
			delete(name2Claim, "wt-gateway")
		}
	}

	return name2Claim
}

func (r *WutongClusterReconciler) createWutongPackage() error {
	pkg := &v1alpha1.WutongPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.WutongPackageName,
			Namespace: constants.WutongSystemNamespace,
		},
		Spec: v1alpha1.WutongPackageSpec{
			PkgPath: "/opt/wutong/pkg/tgz/wutong.tgz",
		},
	}
	return r.createResourceIfNotExists(pkg)
}

func (r *WutongClusterReconciler) createWutongVolumes(cluster *v1alpha1.WutongCluster) error {
	if cluster.Spec.WutongVolumeSpecRWX != nil {
		rwx := setWutongVolume("wutongvolumerwx", constants.WutongSystemNamespace, wtutil.LabelsForAccessModeRWX(), cluster.Spec.WutongVolumeSpecRWX)
		rwx.Spec.ImageRepository = constants.InstallImageRepo
		if err := r.createResourceIfNotExists(rwx); err != nil {
			return err
		}
	}
	if cluster.Spec.WutongVolumeSpecRWO != nil {
		rwo := setWutongVolume("wutongvolumerwo", constants.WutongSystemNamespace, wtutil.LabelsForAccessModeRWO(), cluster.Spec.WutongVolumeSpecRWO)
		rwo.Spec.ImageRepository = constants.InstallImageRepo
		if err := r.createResourceIfNotExists(rwo); err != nil {
			return err
		}
	}
	return nil
}

func (r *WutongClusterReconciler) createResourceIfNotExists(resource client.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(resource), resource)
	if err == nil {
		return nil
	}
	err = r.Client.Create(ctx, resource)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("create resource %s/%s failure %s", resource.GetObjectKind(), resource.GetName(), err.Error())
	}
	return nil
}

func setWutongVolume(name, namespace string, labels map[string]string, spec *v1alpha1.WutongVolumeSpec) *v1alpha1.WutongVolume {
	volume := &v1alpha1.WutongVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    wtutil.LabelsForWutong(labels),
		},
		Spec: *spec,
	}
	return volume
}

func imageFitArch(image string, arch string) string {
	if arch == "" || arch == RegionArchAmd64 {
		return image
	}
	return fmt.Sprintf("%s-%s", image, arch)
}
