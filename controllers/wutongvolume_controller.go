/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong-operator/controllers/plugin"
	"github.com/wutong-paas/wutong-operator/util/k8sutil"
	"github.com/wutong-paas/wutong-operator/util/wtutil"
)

// WutongVolumeReconciler reconciles a WutongVolume object
type WutongVolumeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//ErrCSIPluginNotReady -
var ErrCSIPluginNotReady = errors.New("csi plugin not ready")

// +kubebuilder:rbac:groups=wutong.io,resources=WutongVolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wutong.io,resources=WutongVolumes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wutong.io,resources=WutongVolumes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WutongVolume object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *WutongVolumeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("WutongVolume", request.NamespacedName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the WutongVolume instance
	volume := &wutongv1alpha1.WutongVolume{}
	err := r.Get(ctx, request.NamespacedName, volume)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	useStorageClassName := volume.Spec.StorageClassName != ""
	if useStorageClassName {
		if err := r.updateVolumeStatus(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("wutong volume storage class is ready", "storageclass", useStorageClassName)
		return reconcile.Result{}, nil
	}

	useStorageClassParameters := volume.Spec.StorageClassParameters != nil && volume.Spec.StorageClassParameters.Provisioner != ""
	if useStorageClassParameters {
		log.Info("wutong volume storage class is config, will sync storageclass", "provisioner", volume.Spec.StorageClassParameters.Provisioner)
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
		log.Info("wutong volume storage class is sync success", "provisioner", volume.Spec.StorageClassParameters.Provisioner)
		return reconcile.Result{}, nil
	}

	useCSIPlugin := volume.Spec.CSIPlugin != nil
	if useCSIPlugin {
		log.Info("wutong volume will sync csiplugin")
		csiplugin, err := NewCSIPlugin(ctx, r.Client, volume)
		if err != nil {
			if err := r.updateVolumeStatus(ctx, volume); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		if err := r.applyCSIPlugin(ctx, csiplugin, volume); err != nil {
			if err == ErrCSIPluginNotReady {
				log.Info(err.Error())
				return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
			}
			if err := r.updateVolumeStatus(ctx, volume); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		if err := r.updateVolumeRetryOnConflict(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("wutong volume sync csiplugin resource success")
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WutongVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wutongv1alpha1.WutongVolume{}).
		Complete(r)
}

func (r *WutongVolumeReconciler) applyCSIPlugin(ctx context.Context, plugin plugin.CSIPlugin, volume *wutongv1alpha1.WutongVolume) error {
	if plugin.IsPluginReady() {
		if volume.Spec.StorageClassParameters == nil {
			volume.Spec.StorageClassParameters = &wutongv1alpha1.StorageClassParameters{}
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
		if err := r.createIfNotExists(ctx, res); err != nil {
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
		if err := controllerutil.SetControllerReference(volume, res.(metav1.Object), r.Scheme); err != nil {
			return err
		}
		if err := r.createIfNotExists(ctx, res); err != nil {
			return err
		}
	}

	// requeue the volume with error
	return ErrCSIPluginNotReady
}

func (r *WutongVolumeReconciler) createIfNotExists(ctx context.Context, obj client.Object) error {
	log := r.Log.WithValues("namespace", obj.GetNamespace(), "name", obj.GetName())

	err := r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	} else {
		log.Info(fmt.Sprintf("%s %s is exist", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName()))
		return nil
	}

	log.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind))
	err = r.Create(ctx, obj)
	if err != nil {
		log.Error(err, "Failed to create new", obj.GetObjectKind())
		return err
	}
	return nil
}

func (r *WutongVolumeReconciler) updateVolumeStatus(ctx context.Context, volume *wutongv1alpha1.WutongVolume) error {
	status := volume.Status.DeepCopy()
	_, condtion := status.GetWutongVolumeCondition(wutongv1alpha1.WutongVolumeReady)
	if condtion == nil {
		condtion = &wutongv1alpha1.WutongVolumeCondition{Type: wutongv1alpha1.WutongVolumeReady}
	}
	if volume.Spec.StorageClassName == "" {
		condtion.Status = corev1.ConditionFalse
	} else {
		condtion.Status = corev1.ConditionTrue
	}

	volume.Status.UpdateWutongVolumeCondition(condtion)
	if updated := status.UpdateWutongVolumeCondition(condtion); updated {
		return r.updateVolumeStatusRetryOnConflict(ctx, volume)
	}
	return nil
}

func (r *WutongVolumeReconciler) updateVolumeStatusRetryOnConflict(ctx context.Context, volume *wutongv1alpha1.WutongVolume) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.Status().Update(ctx, volume)
	})
}

func (r *WutongVolumeReconciler) updateVolumeRetryOnConflict(ctx context.Context, volume *wutongv1alpha1.WutongVolume) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		old := &wutongv1alpha1.WutongVolume{}
		err := r.Get(ctx, types.NamespacedName{Namespace: volume.Namespace, Name: volume.Name}, old)
		if err != nil {
			return err
		}
		old.Labels = volume.Labels
		old.Annotations = volume.Annotations
		old.Spec = volume.Spec
		return r.Update(ctx, old)
	})
}

func (r *WutongVolumeReconciler) createIfNotExistStorageClass(ctx context.Context, volume *wutongv1alpha1.WutongVolume) (string, error) {
	old := &storagev1.StorageClass{}
	// check if the storageclass based on the given sc exists.
	err := r.Get(ctx, types.NamespacedName{Name: volume.Name}, old)
	if err == nil {
		return old.Name, nil
	}
	if !k8sErrors.IsNotFound(err) {
		return "", err
	}
	// create a new one
	sc := storageClassForWutongVolume(volume)
	if err := r.Create(ctx, sc); err != nil {
		return "", err
	}
	return sc.Name, nil
}

func storageClassForWutongVolume(volume *wutongv1alpha1.WutongVolume) *storagev1.StorageClass {
	class := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   volume.Name,
			Labels: wtutil.LabelsForWutong(nil),
		},
		MountOptions:  volume.Spec.StorageClassParameters.MountOptions,
		Provisioner:   volume.Spec.StorageClassParameters.Provisioner,
		Parameters:    volume.Spec.StorageClassParameters.Parameters,
		ReclaimPolicy: k8sutil.PersistentVolumeReclaimPolicy(corev1.PersistentVolumeReclaimRetain),
	}

	if volume.Spec.CSIPlugin != nil && volume.Spec.CSIPlugin.AliyunNas != nil && len(class.MountOptions) == 0 {
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
