package rbdcomponent

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
)

var log = logf.Log.WithName("controller_rbdcomponent")

type resourcesFunc func(r *rainbondv1alpha1.RbdComponent) []interface{}

var resourcesFuncs map[string]resourcesFunc

func init() {
	resourcesFuncs = map[string]resourcesFunc{
		"rbd-db":       resourcesForDB,
		"rbd-etcd":     resourcesForEtcd,
		"rbd-hub":      resourcesForHub,
		"rbd-gateway":  resourcesForGateway,
		"rbd-node":     resourcesForNode,
		"rbd-api":      resourcesForAPI,
		"rbd-app-ui":   resourcesForAppUI,
		"rbd-worker":   resourcesForWorker,
		"rbd-chaos":    resourcesForChaos,
		"rbd-eventlog": resourcesForEventLog,
		"rbd-monitor":  resourcesForMonitor,
		"rbd-mq":       resourcesForMQ,
		"rbd-dns":      resourcesForDNS,
	}
}

// Add creates a new RbdComponent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRbdComponent{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rbdcomponent-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RbdComponent
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RbdComponent{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner RbdComponent
	err = c.Watch(&source.Kind{Type: &appv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.RbdComponent{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployment and requeue the owner RbdComponent
	err = c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.RbdComponent{},
	})
	if err != nil {
		return err
	}

	// TODO: duplicated code
	// Watch for changes to secondary resource StatefulSet and requeue the owner RbdComponent
	err = c.Watch(&source.Kind{Type: &appv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.RbdComponent{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRbdComponent implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRbdComponent{}

// ReconcileRbdComponent reconciles a RbdComponent object
type ReconcileRbdComponent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RbdComponent object and makes changes based on the state read
// and what is in the RbdComponent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRbdComponent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Namespace", request.Namespace, "Name", request.Name)
	reqLogger.Info("Reconciling RbdComponent")

	// Fetch the RbdComponent instance
	instance := &rainbondv1alpha1.RbdComponent{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{Requeue: true}, err
	}

	if instance.Name == "rbd-package" {
		rainbondcluster := &rainbondv1alpha1.RainbondCluster{}
		if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "rainbondcluster"}, rainbondcluster); err != nil {
			reqLogger.Error(err, "Error getting rainbondcluster")
			return reconcile.Result{Requeue: true}, err
		}

		if err := handleRainbondPackage(r.client, rainbondcluster, "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz", "/opt/rainbond/pkg"); err != nil {
			reqLogger.Error(err, "handle rainbond package")
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	fn, ok := resourcesFuncs[instance.Name]
	if !ok {
		reqLogger.Info("Unsupported RbdComponent.")
		return reconcile.Result{}, nil
	}

	// TODO: check if the component is ready to be created

	controllerType := rainbondv1alpha1.ControllerTypeUnknown
	for _, res := range fn(instance) {
		if ct := detectControllerType(res); ct != rainbondv1alpha1.ControllerTypeUnknown {
			controllerType = ct
		}

		// Set RbdComponent instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, res.(metav1.Object), r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// TODO: do not update secret for hub every times.
		// Check if the resource already exists, if not create a new one
		reconcileResult, err := r.updateOrCreateResource(reqLogger, res.(runtime.Object), res.(metav1.Object))
		if err != nil {
			return reconcileResult, err
		}
	}

	if instance.Name == "rbd-etcd" { // TODO:
		return reconcile.Result{}, nil
	}
	instance.Status = &rainbondv1alpha1.RbdComponentStatus{
		ControllerType: controllerType,
		ControllerName: instance.Name,
	}

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Update RbdComponent status", "Name", instance.Name)
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRbdComponent) updateOrCreateResource(reqLogger logr.Logger, obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			reqLogger.Error(err, "Failed to create new", obj.GetObjectKind(), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
			return reconcile.Result{}, err
		}
		// daemonset created successfully - return and requeue TODO: why?
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, fmt.Sprintf("Failed to get %s", obj.GetObjectKind()))
		return reconcile.Result{}, err
	}

	// obj exsits, update
	reqLogger.Info("Object exists.", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
	if err := r.client.Update(context.TODO(), obj); err != nil {
		reqLogger.Error(err, "Failed to update ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRbdComponent) createIfNotExistResource(reqLogger logr.Logger, obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
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

	return reconcile.Result{}, nil
}

// labelsForRbdComponent returns the labels for selecting the resources
// belonging to the given RbdComponent CR name.
func labelsForRbdComponent(labels map[string]string) map[string]string {
	rbdlabels := map[string]string{
		"creator": "Rainbond",
	}

	for k, v := range labels {
		rbdlabels[k] = v
	}

	return rbdlabels
}

func (r *ReconcileRbdComponent) isRbdHubReady() bool {
	reqLogger := log.WithName("Check if rbd-hub is ready")

	eps := &corev1.EndpointsList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"name": "rbd-hub",
		}),
	}
	err := r.client.List(context.TODO(), eps, listOpts...)
	if err != nil {
		reqLogger.Error(err, "list rbd-hub endpints")
		return false
	}

	for _, ep := range eps.Items {
		for _, subset := range ep.Subsets {
			if len(subset.Addresses) > 0 {
				reqLogger.Info("Found a healthy endpoint address", "address", subset.Addresses[0])
				return true
			}
		}
	}

	return false
}

func detectControllerType(ctrl interface{}) rainbondv1alpha1.ControllerType {
	if _, ok := ctrl.(*appv1.Deployment); ok {
		return rainbondv1alpha1.ControllerTypeDeployment
	}
	if _, ok := ctrl.(*appv1.StatefulSet); ok {
		return rainbondv1alpha1.ControllerTypeStatefulSet
	}
	if _, ok := ctrl.(*appv1.DaemonSet); ok {
		return rainbondv1alpha1.ControllerTypeDaemonSet
	}
	return rainbondv1alpha1.ControllerTypeUnknown
}
