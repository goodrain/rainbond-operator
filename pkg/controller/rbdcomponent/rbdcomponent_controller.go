package rbdcomponent

import (
	"context"
	"fmt"
	"time"

	chandler "github.com/GLYASAI/rainbond-operator/pkg/controller/rbdcomponent/handler"
	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	storagev1 "k8s.io/api/storage/v1"
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
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
)

var log = logf.Log.WithName("controller_rbdcomponent")

type handlerFunc func(component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) chandler.ComponentHandler

var handlerFuncs map[string]handlerFunc

func AddHandlerFunc(name string, fn handlerFunc) {
	if handlerFuncs == nil {
		handlerFuncs = map[string]handlerFunc{}
	}
	handlerFuncs[name] = fn
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

	// Fetch the RbdComponent cpt
	cpt := &rainbondv1alpha1.RbdComponent{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cpt)
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

	fn, ok := handlerFuncs[cpt.Name]
	if !ok {
		// TODO: report status, events
		reqLogger.Info("Unsupported RbdComponent.")
		return reconcile.Result{}, nil
	}

	// check prerequisites
	cluster := &rainbondv1alpha1.RainbondCluster{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondClusterName}, cluster); err != nil {
		reqLogger.Error(err, "failed to get rainbondcluster.")
		// TODO: report status, events
		return reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}

	hdl := fn(cpt, cluster)

	if err := hdl.Before(); err != nil {
		// TODO: report events
		reqLogger.Info("error checking the prerequisites", "err", err)
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	resourceses := hdl.Resources()

	controllerType := rainbondv1alpha1.ControllerTypeUnknown
	for _, res := range resourceses {
		if ct := detectControllerType(res); ct != rainbondv1alpha1.ControllerTypeUnknown {
			controllerType = ct
		}

		// Set RbdComponent cpt as the owner and controller
		if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the resource already exists, if not create a new one
		reconcileResult, err := r.updateOrCreateResource(reqLogger, res.(runtime.Object), res.(metav1.Object))
		if err != nil {
			return reconcileResult, err
		}
	}

	if cpt.Name == "rbd-nfs-provisioner" { // TODO: move to prepare manager
		class := storageClassForNFSProvisioner()
		oldClass := &storagev1.StorageClass{}
		if err := r.client.Get(context.TODO(), types.NamespacedName{Name: class.Name}, oldClass); err != nil {
			if errors.IsNotFound(err) {
				reqLogger.Info("StorageClass not found, will create a new one")
				if err := r.client.Create(context.TODO(), class); err != nil {
					reqLogger.Error(err, "Error creating storageclass")
					return reconcile.Result{Requeue: true}, err
				}
			} else {
				reqLogger.Error(err, "Error checking if storageclass exists")
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	if cpt.Name == "rbd-etcd" { // TODO:
		return reconcile.Result{}, nil
	}

	cpt.Status = &rainbondv1alpha1.RbdComponentStatus{
		ControllerType: controllerType,
		ControllerName: cpt.Name,
	}
	if err := r.client.Status().Update(context.TODO(), cpt); err != nil {
		reqLogger.Error(err, "Update RbdComponent status", "Name", cpt.Name)
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
		reqLogger.Error(err, "Failed to update", "Kind", obj.GetObjectKind())
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

func storageClassForNFSProvisioner() *storagev1.StorageClass {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rbd-nfs",
		},
		Provisioner: "rainbond.io/nfs",
		MountOptions: []string{
			"vers=4.1",
		},
	}

	return sc
}
