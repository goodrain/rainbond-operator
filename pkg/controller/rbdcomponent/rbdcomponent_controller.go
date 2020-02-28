package rbdcomponent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	chandler "github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent/handler"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
)

var log = logf.Log.WithName("controller_rbdcomponent")

type handlerFunc func(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) chandler.ComponentHandler

var handlerFuncs map[string]handlerFunc

// AddHandlerFunc adds handlerFunc to handlerFuncs.
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

	secondaryResourceTypes := []runtime.Object{
		&appv1.DaemonSet{},
		&appv1.StatefulSet{},
		&appv1.Deployment{},
		&corev1.Service{},
		&extensions.Ingress{},
		&corev1.Secret{},
		&corev1.ConfigMap{},
		&corev1.PersistentVolumeClaim{},
	}

	for _, t := range secondaryResourceTypes {
		err = c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &rainbondv1alpha1.RbdComponent{},
		})
		if err != nil {
			return err
		}
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
	reqLogger.V(6).Info("Reconciling RbdComponent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the RbdComponent cpt
	cpt := &rainbondv1alpha1.RbdComponent{}
	err := r.client.Get(ctx, request.NamespacedName, cpt)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
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

	cluster := &rainbondv1alpha1.RainbondCluster{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondClusterName}, cluster); err != nil {
		reqLogger.Error(err, "failed to get rainbondcluster.")
		cpt.Status = &rainbondv1alpha1.RbdComponentStatus{
			Message: fmt.Sprintf("failed to get rainbondcluster: %v", err),
			Reason:  "ErrGetRainbondCluster",
		}
		if err := k8sutil.UpdateCRStatus(r.client, cpt); err != nil {
			reqLogger.Error(err, "update rbdcomponent status")
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, err
	}

	pkg := &rainbondv1alpha1.RainbondPackage{}
	if cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeFullOnline {
		if err := r.client.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondPackageName}, pkg); err != nil {
			reqLogger.Error(err, "failed to get rainbondpackage.")
			cpt.Status = &rainbondv1alpha1.RbdComponentStatus{
				Message: fmt.Sprintf("failed to get rainbondpackage: %v", err),
				Reason:  "ErrGetRainbondPackage",
			}
			if err := k8sutil.UpdateCRStatus(r.client, cpt); err != nil {
				reqLogger.Error(err, "update rbdcomponent status")
			}
			return reconcile.Result{RequeueAfter: 3 * time.Second}, err
		}
	}

	checkPrerequisites := func() bool {
		if cpt.Spec.PriorityComponent {
			// If ImageHub is empty, the priority component no need to wait until rainbondpackage is completed.
			return true
		}
		// Otherwise, we have to make sure rainbondpackage is completed before we create the resource.
		if cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeFullOnline {
			if err := checkPackageStatus(pkg); err != nil {
				return false
			}
		}
		return true
	}
	if !checkPrerequisites() {
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	hdl := fn(ctx, r.client, cpt, cluster)

	if err := hdl.Before(); err != nil {
		// TODO: report events
		if chandler.IsIgnoreError(err) {
			reqLogger.V(6).Info("checking the prerequisites", "msg", err.Error())
			return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
		}
		isSetV1beta1MetricsFlag := cpt.Annotations != nil && cpt.Annotations["v1beta1.metrics.k8s.io.exists"] == "true"
		if err == chandler.ErrV1beta1MetricsExists && !isSetV1beta1MetricsFlag {
			if err := r.setV1beta1MetricsFlag(ctx, cpt); err == nil {
				return reconcile.Result{}, nil
			}
		}
		reqLogger.Info("error checking the prerequisites", "err", err)
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	k8sResourcesMgr, ok := hdl.(chandler.K8sResourcesInterface)
	if ok {
		reqLogger.V(6).Info("K8sResourcesInterface create resources create if not exists")
		resourcesCreateIfNotExists := k8sResourcesMgr.ResourcesCreateIfNotExists()
		for _, res := range resourcesCreateIfNotExists {
			if res == nil {
				continue
			}
			// Set RbdComponent cpt as the owner and controller
			if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.scheme); err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			if err := r.resourceCreateIfNotExists(ctx, res.(runtime.Object), res.(metav1.Object)); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	resources := hdl.Resources()
	for _, res := range resources {
		if res == nil {
			continue
		}
		// Set RbdComponent cpt as the owner and controller
		if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.scheme); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		// Check if the resource already exists, if not create a new one
		reconcileResult, err := r.updateOrCreateResource(reqLogger, res.(runtime.Object), res.(metav1.Object))
		if err != nil {
			return reconcileResult, err
		}
	}

	if err := hdl.After(); err != nil {
		reqLogger.Error(err, "failed to execute after process")
		return reconcile.Result{Requeue: true}, err
	}

	cpt.Status = generateRainbondComponentStatus(cpt, cluster, resources)
	if err := r.client.Status().Update(ctx, cpt); err != nil {
		reqLogger.Error(err, "Update RbdComponent status", "Name", cpt.Name)
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRbdComponent) resourceCreateIfNotExists(ctx context.Context, obj runtime.Object, meta metav1.Object) error {
	reqLogger := log.WithValues("Namespace", meta.GetNamespace(), "Name", meta.GetName())

	err := r.client.Get(ctx, types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
		reqLogger.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		return r.client.Create(ctx, obj)
	}
	return nil
}

func (r *ReconcileRbdComponent) updateOrCreateResource(reqLogger logr.Logger, obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil && k8sErrors.IsNotFound(err) {
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

func generateRainbondComponentStatus(cpt *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, resources []interface{}) *rainbondv1alpha1.RbdComponentStatus {
	controllerType := rainbondv1alpha1.ControllerTypeUnknown
	for _, res := range resources {
		if res == nil {
			continue
		}
		if ct := detectControllerType(res); ct != rainbondv1alpha1.ControllerTypeUnknown {
			controllerType = ct
			break
		}
	}

	status := &rainbondv1alpha1.RbdComponentStatus{
		ControllerType: controllerType,
		ControllerName: cpt.Name,
	}

	return status
}

func checkPackageStatus(pkg *rainbondv1alpha1.RainbondPackage) error {
	var packageCompleted bool
	if pkg.Status != nil {
		for _, cond := range pkg.Status.Conditions {
			if cond.Type == rainbondv1alpha1.Ready && cond.Status == rainbondv1alpha1.Completed {
				packageCompleted = true
				break
			}
		}
	}
	if !packageCompleted {
		return errors.New("rainbond package is not completed in InstallationModeWithoutPackage mode")
	}
	return nil
}

func (r *ReconcileRbdComponent) setV1beta1MetricsFlag(ctx context.Context, cpt *rainbondv1alpha1.RbdComponent) error {
	if cpt.Annotations == nil {
		cpt.Annotations = make(map[string]string)
	}
	cpt.Annotations["v1beta1.metrics.k8s.io.exists"] = "true"
	if err := r.client.Update(ctx, cpt); err != nil {
		return fmt.Errorf("update rbdcomponent: %v", err)
	}
	return nil
}
