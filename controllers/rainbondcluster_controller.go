package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	clustermgr "github.com/goodrain/rainbond-operator/controllers/cluster-mgr"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/uuidutil"
	"github.com/juju/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RainbondClusterReconciler reconciles a RainbondCluster object
type RainbondClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RainbondCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *RainbondClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("rainbondcluster", request.NamespacedName)

	// Fetch the RainbondCluster instance
	rainbondcluster := &rainbondv1alpha1.RainbondCluster{}
	err := r.Get(ctx, request.NamespacedName, rainbondcluster)
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

	mgr := clustermgr.NewClusterMgr(ctx, r.Client, reqLogger, rainbondcluster, r.Scheme)

	// generate status for rainbond cluster
	status, err := mgr.GenerateRainbondClusterStatus()
	if err != nil {
		reqLogger.Error(err, "failed to generate rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		rc := &rainbondv1alpha1.RainbondCluster{}
		if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
			return err
		}
		rc.Status = *status
		return r.Status().Update(ctx, rc)
	}); err != nil {
		reqLogger.Error(err, "update rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	// setup imageHub if empty
	if rainbondcluster.Spec.ImageHub == nil {
		imageHub, err := r.getImageHub(rainbondcluster)
		if err != nil {
			reqLogger.V(6).Info(fmt.Sprintf("set image hub info: %v", err))
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			rc := &rainbondv1alpha1.RainbondCluster{}
			if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
				return err
			}
			rc.Spec.ImageHub = imageHub
			rainbondcluster = rc
			return r.Update(ctx, rc)
		}); err != nil {
			reqLogger.Error(err, "update rainbondcluster")
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
		// Put it back in the queue.
		return reconcile.Result{Requeue: true}, err
	}

	// create secret for pulling images.
	if rainbondcluster.Spec.ImageHub != nil && rainbondcluster.Spec.ImageHub.Username != "" && rainbondcluster.Spec.ImageHub.Password != "" {
		err := mgr.CreateImagePullSecret()
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// create pvc for grdata if not exists
	if err := mgr.CreateFoobarPVCIfNotExists(); err != nil {
		return reconcile.Result{}, err
	}

	for _, con := range rainbondcluster.Status.Conditions {
		if con.Status != corev1.ConditionTrue {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RainbondClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rainbondv1alpha1.RainbondCluster{}).
		Complete(r)
}

func (r *RainbondClusterReconciler) getImageHub(cluster *rainbondv1alpha1.RainbondCluster) (*rainbondv1alpha1.ImageHub, error) {
	return &rainbondv1alpha1.ImageHub{
		Domain:   constants.DefImageRepository,
		Username: "admin",
		Password: uuidutil.NewUUID()[0:8],
	}, nil
}
