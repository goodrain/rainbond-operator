package rainbondvolume

import (
	"context"
	"k8s.io/client-go/util/retry"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_rainbondvolume")

// Add creates a new RainbondVolume Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRainbondVolume{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rainbondvolume-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RainbondVolume
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RainbondVolume{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RainbondVolume
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.RainbondVolume{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRainbondVolume implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRainbondVolume{}

// ReconcileRainbondVolume reconciles a RainbondVolume object
type ReconcileRainbondVolume struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RainbondVolume object and makes changes based on the state read
// and what is in the RainbondVolume.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRainbondVolume) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(6).Info("Reconciling RainbondVolume")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the RainbondVolume instance
	volume := &rainbondv1alpha1.RainbondVolume{}
	err := r.client.Get(ctx, request.NamespacedName, volume)
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

	haveStorageClassName := volume.Spec.StorageClassName != ""
	if haveStorageClassName {
		if err := r.updateVolumeStatus(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	useExternalProvisioner := volume.Spec.CSIPlugin == nil
	if useExternalProvisioner {
		// create StorageClass

		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRainbondVolume) updateVolumeStatus(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) error {
	status := volume.Status.DeepCopy()
	_, condtion := status.GetRainbondVolumeCondition(rainbondv1alpha1.RainbondVolumeReady)
	if condtion == nil {
		condtion = &rainbondv1alpha1.RainbondVolumeCondition{Type: rainbondv1alpha1.RainbondVolumeReady}
	}
	if volume.Spec.StorageClassName == "" {
		condtion.Status = corev1.ConditionFalse
	} else {
		condtion.Status = corev1.ConditionTrue
	}

	volume.Status.UpdateRainbondVolumeCondition(condtion)
	if updated := status.UpdateRainbondVolumeCondition(condtion); updated {
		return r.updateVolumeStatusRetryOnConflict(ctx, volume)
	}
	return nil
}

func (r *ReconcileRainbondVolume) updateVolumeStatusRetryOnConflict(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.client.Status().Update(ctx, volume)
	})
}

func storageClassForRainbondVolume(volume *rainbondv1alpha1.RainbondVolume) *storagev1.StorageClass {
	class := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: volume.Name,
		},
	}
	return class
}
