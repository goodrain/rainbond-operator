package rainbond

import (
	"context"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type daemonSetForRainbond func(p *rainbondv1alpha1.Rainbond) *appsv1.DaemonSet

var daemonSetForRainbondFuncs map[string]daemonSetForRainbond

func init() {
	daemonSetForRainbondFuncs := make(map[string]daemonSetForRainbond)
	daemonSetForRainbondFuncs["rbd-worker"] = daemonSetForRainbondWorker
	daemonSetForRainbondFuncs["rbd-api"] = daemonSetForRainbondAPI
	daemonSetForRainbondFuncs["rbd-chaos"] = daemonSetForRainbondChaos
	daemonSetForRainbondFuncs["rbd-eventlog"] = daemonSetForRainbondEventlog
	daemonSetForRainbondFuncs["rbd-gateway"] = daemonSetForRainbondGateway
	daemonSetForRainbondFuncs["rbd-monitor"] = daemonSetForRainbondMonitor
	daemonSetForRainbondFuncs["rbd-mq"] = daemonSetForRainbondMQ
	daemonSetForRainbondFuncs["rbd-dns"] = daemonSetForRainbondDNS
}

var log = logf.Log.WithName("controller_rainbond")

// Add creates a new Rainbond Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRainbond{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rainbond-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Rainbond
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.Rainbond{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner Rainbond
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.Rainbond{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRainbond implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRainbond{}

// ReconcileRainbond reconciles a Rainbond object
type ReconcileRainbond struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Rainbond object and makes changes based on the state read
// and what is in the Rainbond.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRainbond) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Rainbond")

	// Fetch the Rainbond instance
	instance := &rainbondv1alpha1.Rainbond{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	for name := range daemonSetForRainbondFuncs {
		// Define a new daemonset
		daemonSetForRainbond := daemonSetForRainbondFuncs[name]
		daemonSet := daemonSetForRainbond(instance)

		// Set PrivateRegistry instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, daemonSet, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the daemonset already exists, if not create a new one
		found := &appsv1.DaemonSet{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
			err = r.client.Create(context.TODO(), daemonSet)
			if err != nil {
				reqLogger.Error(err, "Failed to create new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
				return reconcile.Result{}, err
			}
			// daemonset created successfully - return and requeue
			return reconcile.Result{Requeue: true}, nil
		} else if err != nil {
			reqLogger.Error(err, "Failed to get DaemonSet")
			return reconcile.Result{}, err
		}

		// DaemonSet already exists - don't requeue
		reqLogger.Info("Skip reconcile: DaemonSet already exists", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
	}

	return reconcile.Result{}, nil
}

// labelsForRainbond returns the labels for selecting the resources
// belonging to the given Rainbond CR name.
func labelsForRainbond(name string) map[string]string {
	return map[string]string{"name": name} // TODO: only one rainbond?
}
