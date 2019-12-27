package rainbond

import (
	"context"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type controllerForRainbond func(p *rainbondv1alpha1.Rainbond) interface{}

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

	// Watch for changes to secondary resource StatefulSet and requeue the owner Rainbond
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
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

	controllerForRainbondFuncs := map[string]controllerForRainbond{
		"rbd-app-ui":     deploymentForRainbondAppUI,
		"rbd-db":         statefulsetForRainbondDB,
		"metrics-server": deploymentForMetricsServer,
		"rbd-worker":     daemonSetForRainbondWorker,
		"rbd-api":        daemonSetForRainbondAPI,
		"rbd-chaos":      daemonSetForRainbondChaos,
		"rbd-eventlog":   daemonSetForRainbondEventlog,
		"rbd-gateway":    daemonSetForRainbondGateway,
		"rbd-monitor":    daemonSetForRainbondMonitor,
		"rbd-mq":         daemonSetForRainbondMQ,
		"rbd-dns":        daemonSetForRainbondDNS,
	}

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

	for name := range controllerForRainbondFuncs {
		generic := controllerForRainbondFuncs[name](instance)
		reqLogger.Info("Name", name, "Reconciling", generic.(runtime.Object).GetObjectKind().GroupVersionKind().Kind)
		// Set PrivateRegistry instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, generic.(metav1.Object), r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the statefulset already exists, if not create a new one
		reconcileResult, err := r.updateOrCreateResource(reqLogger, generic.(runtime.Object), generic.(metav1.Object))
		if err != nil {
			return reconcileResult, err
		}
	}

	return reconcile.Result{}, nil
}

// labelsForRainbond returns the labels for selecting the resources
// belonging to the given Rainbond CR name.
func labelsForRainbond(name string) map[string]string {
	return map[string]string{"name": name} // TODO: only one rainbond?
}

func (r *ReconcileRainbond) updateOrCreateResource(reqLogger logr.Logger, obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			reqLogger.Error(err, "Failed to create new", obj.GetObjectKind(), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
			return reconcile.Result{}, err
		}
		// daemonset created successfully - return and requeue TODO: why?
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	// obj exsits, update
	reqLogger.Info("Update ", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
	if err := r.client.Update(context.TODO(), obj); err != nil {
		reqLogger.Error(err, "Failed to update ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
