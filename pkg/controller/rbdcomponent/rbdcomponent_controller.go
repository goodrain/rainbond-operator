package rbdcomponent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	chandler "github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent/handler"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

func supportedComponents() string {
	var supported []string
	for name := range handlerFuncs {
		supported = append(supported, name)
	}
	sort.Strings(supported)
	return strings.Join(supported, ",")
}

// Add creates a new RbdComponent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	recorder := mgr.GetEventRecorderFor("rbdcomponent")
	return &ReconcileRbdComponent{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: recorder}
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
		&batchv1.Job{},
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
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
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
	if cpt.Status == nil {
		cpt.Status = &rainbondv1alpha1.RbdComponentStatus{}
	}

	mgr := newRbdcomponentMgr(ctx, r.client, reqLogger, cpt)

	fn, ok := handlerFuncs[cpt.Name]
	if !ok {
		reason := "UnsupportedType"
		msg := fmt.Sprintf("only supports the following types of rbdcomponent: %s", supportedComponents())

		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, reason, msg)
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, reason, msg)
			return reconcile.Result{}, mgr.updateStatus()
		}
		return reconcile.Result{}, nil
	}

	cluster := &rainbondv1alpha1.RainbondCluster{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondClusterName}, cluster); err != nil {
		condition := clusterCondition(err)
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.updateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	if !cluster.Spec.ConfigCompleted {
		reqLogger.V(6).Info("rainbondcluster configuration is not complete")
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted,
			corev1.ConditionFalse, "ConfigNotCompleted", "rainbondcluster configuration is not complete")
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.updateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, err
	}
	mgr.setConfigCompletedCondition()

	var pkg *rainbondv1alpha1.RainbondPackage
	if cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeFullOnline {
		pkg = &rainbondv1alpha1.RainbondPackage{}
		if err := r.client.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondPackageName}, pkg); err != nil {
			condition := packageCondition(err)
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.updateStatus()
			}

			return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
		}
	}
	mgr.setPackageReadyCondition(pkg)

	if !mgr.checkPrerequisites(cluster, pkg) {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
			"PrerequisitesFailed", "failed to check prerequisites")
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.updateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	hdl := fn(ctx, r.client, cpt, cluster)
	if err := hdl.Before(); err != nil {
		// TODO: merge with mgr.checkPrerequisites
		if chandler.IsIgnoreError(err) {
			reqLogger.V(7).Info("checking the prerequisites", "msg", err.Error())
		} else {
			reqLogger.V(6).Info("checking the prerequisites", "msg", err.Error())
		}

		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "PrerequisitesFailed", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.updateStatus()
		}
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
				reqLogger.Error(err, "set controller reference")
				condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
					corev1.ConditionFalse, "SetControllerReferenceFailed", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.updateStatus()
				}
				return reconcile.Result{}, err
			}
			if err := mgr.resourceCreateIfNotExists(res.(runtime.Object), res.(metav1.Object)); err != nil {
				reqLogger.Error(err, "create resouce if not exists")
				condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
					corev1.ConditionFalse, "ErrCreateResources", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.updateStatus()
				}
				return reconcile.Result{}, err
			}
		}
	}

	replicaser, ok := hdl.(chandler.Replicaser)
	if ok {
		mgr.setReplicaser(replicaser)
	}

	resources := hdl.Resources()
	for _, res := range resources {
		if res == nil {
			continue
		}
		// Set RbdComponent cpt as the owner and controller
		if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.scheme); err != nil {
			reqLogger.Error(err, "set controller reference")
			condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
				"SetControllerReferenceFailed", err.Error())
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{Requeue: true}, mgr.updateStatus()
			}
			return reconcile.Result{}, err
		}
		// Check if the resource already exists, if not create a new one
		reconcileResult, err := mgr.updateOrCreateResource(res.(runtime.Object), res.(metav1.Object))
		if err != nil {
			reqLogger.Error(err, "update or create resource")
			condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "ErrCreateResources", err.Error())
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{Requeue: true}, mgr.updateStatus()
			}
			return reconcileResult, err
		}
	}

	if err := hdl.After(); err != nil {
		reqLogger.Error(err, "failed to execute after process")
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
			"ErrAfterProcess", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{Requeue: true}, mgr.updateStatus()
		}
		return reconcile.Result{Requeue: true}, err
	}

	pods, err := hdl.ListPods()
	if err != nil {

		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
			"ErrListPods", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{Requeue: true}, mgr.updateStatus()
		}
		return reconcile.Result{Requeue: true}, err
	}

	mgr.generateStatus(pods)

	if !mgr.isRbdComponentReady() {
		return reconcile.Result{RequeueAfter: 1 * time.Second}, mgr.updateStatus()
	}

	return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.updateStatus()
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

func clusterCondition(err error) *rainbondv1alpha1.RbdComponentCondition {
	reason := "ClusterNotFound"
	msg := "rainbondcluster not found"
	if !k8sErrors.IsNotFound(err) {
		reason = "UnknownErr"
		msg = fmt.Sprintf("failed to get rainbondcluster: %v", err)
	}

	return rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionFalse, reason, msg)
}

func packageCondition(err error) *rainbondv1alpha1.RbdComponentCondition {
	reason := "PackageNotFound"
	msg := "rainbondpackage not found"
	if !k8sErrors.IsNotFound(err) {
		reason = "UnknownErr"
		msg = fmt.Sprintf("failed to get rainbondpackage: %v", err)
	}
	return rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, reason, msg)
}
