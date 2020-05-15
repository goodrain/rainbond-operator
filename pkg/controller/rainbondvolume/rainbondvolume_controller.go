package rainbondvolume

import (
	"context"
	"fmt"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/controller/rainbondvolume/plugin"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_rainbondvolume")

// Add creates a new RainbondVolume Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, ns string) error {
	return add(mgr, newReconciler(mgr), ns)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRainbondVolume{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, ns string) error {
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

	secondaryResourceTypes := []runtime.Object{
		&appv1.DaemonSet{},
		&appv1.StatefulSet{},
		&appv1.Deployment{},
		&corev1.Service{},
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

	// TODO: validate volume

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

	useStorageClassName := volume.Spec.StorageClassName != ""
	if useStorageClassName {
		if err := r.updateVolumeStatus(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	useStorageClassParameters := volume.Spec.StorageClassParameters != nil && volume.Spec.StorageClassParameters.Provisioner != ""
	if useStorageClassParameters {
		className, err := r.createIfNotExistStorageClass(ctx, volume)
		if err != nil {
			return reconcile.Result{}, err
		}
		volume.Spec.StorageClassName = className
		if err := r.updateVolumeRetryOnConflict(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.updateVolumeStatus(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	useCSIPlugin := volume.Spec.CSIPlugin != nil
	if useCSIPlugin {
		csiplugin, err := NewCSIPlugin(ctx, r.client, volume)
		if err != nil {
			if err := r.updateVolumeStatus(ctx, volume); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		if err := r.applyCSIPlugin(ctx, csiplugin, volume); err != nil {
			if err := r.updateVolumeStatus(ctx, volume); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		if err := r.updateVolumeRetryOnConflict(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRainbondVolume) applyCSIPlugin(ctx context.Context, plugin plugin.CSIPlugin, volume *rainbondv1alpha1.RainbondVolume) error {
	if plugin.IsPluginReady() {
		if volume.Spec.StorageClassParameters == nil {
			volume.Spec.StorageClassParameters = &rainbondv1alpha1.StorageClassParameters{}
		}
		volume.Spec.StorageClassParameters.Provisioner = plugin.GetProvisioner()
		return nil
	}

	clusterScopedResources := plugin.GetClusterScopedResources()
	for idx := range clusterScopedResources {
		res := clusterScopedResources[idx]
		if res == nil {
			continue
		}
		if err := r.createIfNotExists(ctx, res.(runtime.Object), res.(metav1.Object)); err != nil {
			return err
		}
	}

	subResources := plugin.GetSubResources()
	for idx := range subResources {
		res := subResources[idx]
		if res == nil {
			continue
		}
		// set volume as the owner and controller
		if err := controllerutil.SetControllerReference(volume, res.(metav1.Object), r.scheme); err != nil {
			return err
		}
		if err := r.updateOrCreateResource(ctx, res.(runtime.Object), res.(metav1.Object)); err != nil {
			return err
		}
	}

	// requeue the volume with error
	return fmt.Errorf("csi plugin not ready")
}

func (r *ReconcileRainbondVolume) updateOrCreateResource(ctx context.Context, obj runtime.Object, meta metav1.Object) error {
	reqLogger := log.WithValues("Namespace", meta.GetNamespace(), "Name", meta.GetName())

	err := r.client.Get(ctx, types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}

		reqLogger.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind))
		err = r.client.Create(ctx, obj)
		if err != nil {
			reqLogger.Error(err, "Failed to create new", obj.GetObjectKind())
			return err
		}
		return nil
	}

	reqLogger.V(5).Info("Object exists.", "Kind", obj.GetObjectKind().GroupVersionKind().Kind)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.client.Status().Update(ctx, obj)
	})
}

func (r *ReconcileRainbondVolume) createIfNotExists(ctx context.Context, obj runtime.Object, meta metav1.Object) error {
	reqLogger := log.WithValues("Namespace", meta.GetNamespace(), "Name", meta.GetName())

	err := r.client.Get(ctx, types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	} else {
		reqLogger.Info(fmt.Sprintf("%s %s is exist", obj.GetObjectKind().GroupVersionKind().Kind, meta.GetName()))
		return nil
	}

	reqLogger.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind))
	err = r.client.Create(ctx, obj)
	if err != nil {
		reqLogger.Error(err, "Failed to create new", obj.GetObjectKind())
		return err
	}

	return nil
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

func (r *ReconcileRainbondVolume) updateVolumeRetryOnConflict(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.client.Update(ctx, volume)
	})
}

func (r *ReconcileRainbondVolume) createIfNotExistStorageClass(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) (string, error) {
	old := &storagev1.StorageClass{}
	// check if the storageclass based on the given sc exists.
	err := r.client.Get(ctx, types.NamespacedName{Name: volume.Name}, old)
	if err == nil {
		return old.Name, nil
	}
	if !k8sErrors.IsNotFound(err) {
		return "", err
	}
	// create a new one
	sc := storageClassForRainbondVolume(volume)
	if err := r.client.Create(ctx, sc); err != nil {
		return "", err
	}
	return sc.Name, nil
}

func storageClassForRainbondVolume(volume *rainbondv1alpha1.RainbondVolume) *storagev1.StorageClass {
	class := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   volume.Name,
			Labels: rbdutil.LabelsForRainbond(nil),
		},
		MountOptions:  volume.Spec.StorageClassParameters.MountOptions,
		Provisioner:   volume.Spec.StorageClassParameters.Provisioner,
		Parameters:    volume.Spec.StorageClassParameters.Parameters,
		ReclaimPolicy: k8sutil.PersistentVolumeReclaimPolicy(corev1.PersistentVolumeReclaimRetain),
	}

	if volume.Spec.CSIPlugin != nil && volume.Spec.CSIPlugin.AliyunNas != nil && class.MountOptions == nil {
		class.MountOptions = []string{
			"nolock,tcp,noresvport",
			"vers=4",
			"minorversion=0",
			"rsize=1048576",
			"wsize=1048576",
			"timeo=600",
			"retrans=2",
			"hard",
		}
	}

	return class
}
