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

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers/plugin"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
)

// RainbondVolumeReconciler 用于调节 RainbondVolume 对象的协调器。
// 它确保 RainbondVolume 对象与实际集群状态同步，
// 根据 RainbondVolume 的配置更新或创建必要的资源，如 StorageClass 和 CSI 插件。

// ErrCSIPluginNotReady 表示与 RainbondVolume 关联的 CSI 插件未准备就绪。

// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondvolumes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondvolumes/finalizers,verbs=update

// Reconcile 是主 Kubernetes 协调循环的一部分，旨在比较 RainbondVolume 对象指定的状态与实际集群状态，
// 并执行操作以使集群状态反映用户指定的状态。
// 它处理存储类创建、CSI 插件同步，并相应地更新 RainbondVolume 对象的状态。

// RainbondVolumeReconciler reconciles a RainbondVolume object
type RainbondVolumeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// ErrCSIPluginNotReady -
var ErrCSIPluginNotReady = errors.New("csi plugin not ready")

// SetupWithManager 使用 Manager 设置控制器。
func (r *RainbondVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rainbondv1alpha1.RainbondVolume{}).
		Complete(r)
}

// Reconcile 是主要的协调逻辑。它检查指定的 RainbondVolume 对象并尝试将集群状态与所需状态接近。
func (r *RainbondVolumeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("rainbondvolume", request.NamespacedName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 获取 RainbondVolume 实例
	volume := &rainbondv1alpha1.RainbondVolume{}
	err := r.Get(ctx, request.NamespacedName, volume)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// 如果找不到对象，可能是在协调请求后被删除。
			// 拥有的对象会自动垃圾回收。对于额外的清理逻辑，请使用 finalizers。
			// 返回并不重新排队
			return ctrl.Result{}, nil
		}
		// 读取对象时出错，重新排队请求。
		return ctrl.Result{}, err
	}

	// 检查是否使用 StorageClassName
	useStorageClassName := volume.Spec.StorageClassName != ""
	if useStorageClassName {
		// 如果使用 StorageClassName，则更新卷的状态
		if err := r.updateVolumeStatus(ctx, volume); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("rainbond volume storage class is ready", "storageclass", useStorageClassName)
		return reconcile.Result{}, nil
	}

	// 检查是否使用 StorageClassParameters
	useStorageClassParameters := volume.Spec.StorageClassParameters != nil && volume.Spec.StorageClassParameters.Provisioner != ""
	if useStorageClassParameters {
		log.Info("rainbond volume storage class is config, will sync storageclass", "provisioner", volume.Spec.StorageClassParameters.Provisioner)
		// 如果使用 StorageClassParameters，则创建或更新 StorageClass
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
		log.Info("rainbond volume storage class is sync success", "provisioner", volume.Spec.StorageClassParameters.Provisioner)
		return reconcile.Result{}, nil
	}

	// 检查是否使用 CSIPlugin
	useCSIPlugin := volume.Spec.CSIPlugin != nil
	if useCSIPlugin {
		log.Info("rainbond volume will sync csiplugin")
		// 如果使用 CSIPlugin，则尝试应用 CSI 插件
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
		log.Info("rainbond volume sync csiplugin resource success")
		return reconcile.Result{Requeue: true}, nil
	}

	// 如果没有任何匹配的情况，返回空结果
	return reconcile.Result{}, nil
}

// applyCSIPlugin 尝试应用 CSI 插件到 RainbondVolume 对象。
// 如果 CSI 插件未准备就绪，会返回 ErrCSIPluginNotReady 错误。
func (r *RainbondVolumeReconciler) applyCSIPlugin(ctx context.Context, plugin plugin.CSIPlugin, volume *rainbondv1alpha1.RainbondVolume) error {
	if plugin.IsPluginReady() {
		if volume.Spec.StorageClassParameters == nil {
			volume.Spec.StorageClassParameters = &rainbondv1alpha1.StorageClassParameters{}
		}
		volume.Spec.StorageClassParameters.Provisioner = plugin.GetProvisioner()
		return nil
	}

	// 同步集群范围资源和子资源
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
		// 设置 RainbondVolume 为资源的所有者和控制器
		if err := controllerutil.SetControllerReference(volume, res.(metav1.Object), r.Scheme); err != nil {
			return err
		}
		if err := r.createIfNotExists(ctx, res); err != nil {
			return err
		}
	}

	// 返回插件未准备就绪的错误，重新排队 RainbondVolume 处理
	return ErrCSIPluginNotReady
}

// createIfNotExists 检查并创建指定的 Kubernetes 资源，如 StorageClass。
// 如果资源已存在，则仅记录日志。
func (r *RainbondVolumeReconciler) createIfNotExists(ctx context.Context, obj client.Object) error {
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

// updateVolumeStatus 更新 RainbondVolume 对象的状态。
// 根据实际情况更新 Ready 条件。
func (r *RainbondVolumeReconciler) updateVolumeStatus(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) error {
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

// updateVolumeStatusRetryOnConflict 通过重试机制更新 RainbondVolume 对象的状态。
func (r *RainbondVolumeReconciler) updateVolumeStatusRetryOnConflict(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.Status().Update(ctx, volume)
	})
}

// updateVolumeRetryOnConflict 通过重试机制更新 RainbondVolume 对象。
func (r *RainbondVolumeReconciler) updateVolumeRetryOnConflict(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		old := &rainbondv1alpha1.RainbondVolume{}
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

// createIfNotExistStorageClass 检查是否存在指定的 StorageClass，如果不存在则创建。
func (r *RainbondVolumeReconciler) createIfNotExistStorageClass(ctx context.Context, volume *rainbondv1alpha1.RainbondVolume) (string, error) {
	old := &storagev1.StorageClass{}
	// 检查给定 StorageClass 名称是否存在
	err := r.Get(ctx, types.NamespacedName{Name: volume.Name}, old)
	if err == nil {
		return old.Name, nil
	}
	if !k8sErrors.IsNotFound(err) {
		return "", err
	}
	// 创建一个新的 StorageClass
	sc := storageClassForRainbondVolume(volume)
	if err := r.Create(ctx, sc); err != nil {
		return "", err
	}
	return sc.Name, nil
}

// storageClassForRainbondVolume 根据 RainbondVolume 对象创建适当的 StorageClass。
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
