package controllers

import (
	"context"
	"fmt"
	"github.com/wutong-paas/wutong-operator/util/retryutil"
	"github.com/wutong-paas/wutong-operator/util/suffixdomain"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"time"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	clustermgr "github.com/wutong-paas/wutong-operator/controllers/cluster-mgr"
	"github.com/wutong-paas/wutong-operator/util/constants"
	"github.com/wutong-paas/wutong-operator/util/uuidutil"
	"github.com/juju/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// WutongClusterReconciler reconciles a WutongCluster object
type WutongClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=wutong.io,resources=WutongClusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wutong.io,resources=WutongClusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wutong.io,resources=WutongClusters/finalizers,verbs=update

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
	reqLogger := r.Log.WithValues("WutongCluster", request.NamespacedName)

	// Fetch the WutongCluster instance
	WutongCluster := &wutongv1alpha1.WutongCluster{}
	err := r.Get(ctx, request.NamespacedName, WutongCluster)
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

	mgr := clustermgr.NewClusterMgr(ctx, r.Client, reqLogger, WutongCluster, r.Scheme)

	// generate status for wutong cluster
	reqLogger.V(6).Info("start generate status")
	status, err := mgr.GenerateWutongClusterStatus()
	if err != nil {
		reqLogger.Error(err, "failed to generate WutongCluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		rc := &wutongv1alpha1.WutongCluster{}
		if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
			return err
		}
		rc.Status = *status
		return r.Status().Update(ctx, rc)
	}); err != nil {
		reqLogger.Error(err, "update WutongCluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}
	reqLogger.V(6).Info("update status success")

	// setup imageHub if empty
	if WutongCluster.Spec.ImageHub == nil {
		reqLogger.V(6).Info("create new image hub info")
		imageHub, err := r.getImageHub(WutongCluster)
		if err != nil {
			reqLogger.V(6).Info(fmt.Sprintf("set image hub info: %v", err))
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			rc := &wutongv1alpha1.WutongCluster{}
			if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
				return err
			}
			rc.Spec.ImageHub = imageHub
			WutongCluster = rc
			return r.Update(ctx, rc)
		}); err != nil {
			reqLogger.Error(err, "update WutongCluster")
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
		reqLogger.V(6).Info("create new image hub info success")
		// Put it back in the queue.
		return reconcile.Result{Requeue: true}, err
	}

	if WutongCluster.Spec.SuffixHTTPHost == "" {
		var ip string
		if len(WutongCluster.Spec.NodesForGateway) > 0 {
			ip = WutongCluster.Spec.NodesForGateway[0].InternalIP
		}
		if len(WutongCluster.Spec.GatewayIngressIPs) > 0 && WutongCluster.Spec.GatewayIngressIPs[0] != "" {
			ip = WutongCluster.Spec.GatewayIngressIPs[0]
		}
		if ip != "" {
			err := retryutil.Retry(1*time.Second, 3, func() (bool, error) {
				domain, err := r.genSuffixHTTPHost(ip, WutongCluster)
				if err != nil {
					return false, err
				}
				WutongCluster.Spec.SuffixHTTPHost = domain
				if !strings.HasSuffix(domain, constants.DefHTTPDomainSuffix) {
					WutongCluster.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
				}
				return true, nil
			})
			if err != nil {
				logrus.Warningf("generate suffix http host: %v", err)
				WutongCluster.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
			}
			return reconcile.Result{}, r.Update(ctx, WutongCluster)
		}
		logrus.Infof("WutongCluster.Spec.SuffixHTTPHost ip is empty %s", ip)
		WutongCluster.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
		return reconcile.Result{}, r.Update(ctx, WutongCluster)
	}

	// create secret for pulling images.
	if WutongCluster.Spec.ImageHub != nil && WutongCluster.Spec.ImageHub.Username != "" && WutongCluster.Spec.ImageHub.Password != "" {
		err := mgr.CreateImagePullSecret()
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// create pvc for grdata if not exists
	if err := mgr.CreateFoobarPVCIfNotExists(); err != nil {
		return reconcile.Result{}, err
	}

	for _, con := range WutongCluster.Status.Conditions {
		if con.Status != corev1.ConditionTrue {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	return ctrl.Result{}, nil
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

func (r *WutongClusterReconciler) genSuffixHTTPHost(ip string, WutongCluster *wutongv1alpha1.WutongCluster) (domain string, err error) {
	id, auth, err := r.getOrCreateUUIDAndAuth(WutongCluster)
	if err != nil {
		return "", err
	}
	domain, err = suffixdomain.GenerateDomain(ip, id, auth)
	if err != nil {
		return "", err
	}
	return domain, nil
}

func (r *WutongClusterReconciler) getOrCreateUUIDAndAuth(WutongCluster *wutongv1alpha1.WutongCluster) (id, auth string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	cm := &corev1.ConfigMap{}
	err = r.Client.Get(context.Background(), types.NamespacedName{Name: "wt-suffix-host", Namespace: WutongCluster.Namespace}, cm)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return "", "", err
	}
	if err != nil && strings.Contains(err.Error(), "not found") {
		logrus.Info("not found configmap wt-suffix-host, create it")
		cm = suffixdomain.GenerateSuffixConfigMap("wt-suffix-host", WutongCluster.Namespace)
		if err = r.Client.Create(ctx, cm); err != nil {
			return "", "", err
		}
	}
	return cm.Data["uuid"], cm.Data["auth"], nil
}
