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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	componentmgr "github.com/goodrain/rainbond-operator/controllers/component-mgr"
	chandler "github.com/goodrain/rainbond-operator/controllers/handler"
	"github.com/goodrain/rainbond-operator/util/constants"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RbdComponentReconciler reconciles a RbdComponent object
type RbdComponentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=rainbond.io,resources=rbdcomponents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rainbond.io,resources=rbdcomponents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rainbond.io,resources=rbdcomponents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RbdComponent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
// Reconcile 是 RbdComponentReconciler 的方法，用于协调处理请求并更新 RbdComponent 资源状态。
func (r *RbdComponentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// 设置日志记录器，用于记录与 RbdComponent 相关的日志信息
	log := r.Log.WithValues("rbdcomponent", request.NamespacedName)
	// 创建上下文并设置取消函数，确保在函数退出时及时取消上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 获取 RbdComponent 对象 cpt
	cpt := &rainbondv1alpha1.RbdComponent{}
	err := r.Get(ctx, request.NamespacedName, cpt)
	if err != nil {
		// 如果对象未找到，可能是在调解请求后已删除
		if k8sErrors.IsNotFound(err) {
			// 对象未找到，自动垃圾回收已拥有的对象。如果需要额外的清理逻辑，请使用 finalizers。
			// 返回并且不重新排队
			return reconcile.Result{}, nil
		}
		// 读取对象时出现错误，重新排队请求
		return reconcile.Result{Requeue: true}, err
	}
	// 创建 RbdcomponentMgr 实例，用于管理 RbdComponent 资源的状态更新
	mgr := componentmgr.NewRbdcomponentMgr(ctx, r.Client, r.Recorder, log, cpt)
	// 根据 RbdComponent 资源名称获取对应的处理函数
	fn, ok := handlerFuncs[cpt.Name]
	if !ok {
		// 如果找不到处理函数，则返回不支持的类型错误
		reason := "UnsupportedType"
		msg := fmt.Sprintf("only supports the following types of rbdcomponent: %s", supportedComponents())

		// 创建新的 RbdComponent 条件，并更新状态
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, reason, msg)
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			// 记录事件并更新状态
			r.Recorder.Event(cpt, corev1.EventTypeWarning, reason, msg)
			return reconcile.Result{}, mgr.UpdateStatus()
		}
		return reconcile.Result{}, nil
	}
	// 获取 RainbondCluster 对象
	cluster := &rainbondv1alpha1.RainbondCluster{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondClusterName}, cluster); err != nil {
		// 更新 Cluster 相关条件并返回，需要重新排队
		condition := clusterCondition(err)
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}
	// 检查 RainbondCluster 的配置是否完成
	if !cluster.Spec.ConfigCompleted {
		log.V(6).Info("rainbondcluster configuration is not complete")
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted,
			corev1.ConditionFalse, "ConfigNotCompleted", "rainbondcluster configuration is not complete")
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, err
	}
	// 设置配置完成的条件到状态中
	mgr.SetConfigCompletedCondition()
	var pkg *rainbondv1alpha1.RainbondPackage
	// 如果安装模式不是完全在线，则获取 RainbondPackage 对象
	if cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeFullOnline {
		pkg = &rainbondv1alpha1.RainbondPackage{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.RainbondPackageName}, pkg); err != nil {
			// 更新 Package 相关条件并返回，需要重新排队
			condition := packageCondition(err)
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
			}
			return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
		}
	}
	// 设置包装就绪的条件到状态中
	mgr.SetPackageReadyCondition(pkg)
	// 检查组件的先决条件
	if !mgr.CheckPrerequisites(cluster, pkg) {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
			"PrerequisitesFailed", "failed to check prerequisites")
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}
	// 执行处理函数，并获取处理器对象
	hdl := fn(ctx, r.Client, cpt, cluster)
	// 处理处理器的 Before 方法，处理先决条件相关逻辑
	if err := hdl.Before(); err != nil {
		// 处理 Before 方法返回的错误
		if chandler.IsIgnoreError(err) {
			log.V(7).Info("checking the prerequisites", "msg", err.Error())
		} else {
			log.V(6).Info("checking the prerequisites", "msg", err.Error())
		}
		// 更新组件状态为不可用，并返回需要重新排队
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "PrerequisitesFailed", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}
	// 处理处理器对象中的 ResourcesDeleter 接口
	resourcesDeleter, ok := hdl.(chandler.ResourcesDeleter)
	if ok {
		// 删除资源，并处理返回结果
		result, err := mgr.DeleteResources(resourcesDeleter)
		if err != nil {
			return reconcile.Result{}, err
		}
		if result != nil {
			return *result, nil
		}
	}
	// 处理处理器对象中的 ResourcesCreator 接口
	resourceCreator, ok := hdl.(chandler.ResourcesCreator)
	if ok {
		log.V(6).Info("ResourcesCreator create resources create if not exists")
		// 创建或检查资源是否存在
		resourcesCreateIfNotExists := resourceCreator.ResourcesCreateIfNotExists()
		for _, res := range resourcesCreateIfNotExists {
			if res == nil {
				continue
			}
			// 将 RbdComponent 设置为资源的所有者和控制器
			if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.Scheme); err != nil {
				log.Error(err, "set controller reference")
				condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
					corev1.ConditionFalse, "SetControllerReferenceFailed", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
				}
				return reconcile.Result{}, err
			}
			// 如果资源不存在，则创建新的资源
			if err := mgr.ResourceCreateIfNotExists(res); err != nil {
				log.Error(err, "create resouce if not exists")
				condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
					corev1.ConditionFalse, "ErrCreateResources", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
				}
				return reconcile.Result{}, err
			}
		}
	}
	// 处理处理器对象中的 ClusterScopedResourcesCreator 接口
	resourceCreatorClusterScope, ok := hdl.(chandler.ClusterScopedResourcesCreator)
	if ok {
		log.V(6).Info("ClusterScopedResourcesCreator create resources create if not exists")
		// 创建或检查集群范围内的资源是否存在
		resourcesCreateIfNotExists := resourceCreatorClusterScope.CreateClusterScoped()
		for _, res := range resourcesCreateIfNotExists {
			if res == nil {
				continue
			}
			// 如果资源不存在，则创建新的资源
			if err := mgr.ResourceCreateIfNotExists(res); err != nil {
				log.Error(err, "create resouce if not exists")
				condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
					corev1.ConditionFalse, "ErrCreateResources", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
				}
				return reconcile.Result{}, err
			}
		}
	}
	// 处理处理器对象中的 Replicaser 接口
	replicaser, ok := hdl.(chandler.Replicaser)
	if ok {
		// 设置 Replicaser 并处理
		mgr.SetReplicaser(replicaser)
	}
	// 获取处理器对象中的所有资源并更新到状态中
	resources := hdl.Resources()
	for _, res := range resources {
		if res == nil {
			continue
		}
		// 设置 RbdComponent 为资源的所有者和控制器
		if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.Scheme); err != nil {
			log.Error(err, "set controller reference")
			condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
				"SetControllerReferenceFailed", err.Error())
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
			}
			return reconcile.Result{}, err
		}
		// 更新或创建资源
		reconcileResult, err := mgr.UpdateOrCreateResource(res)
		if err != nil {
			log.Error(err, "update or create resource")
			condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "ErrCreateResources", err.Error())
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
			}
			return reconcileResult, err
		}
	}
	// 处理处理器对象中的 After 方法，处理后续逻辑
	if err := hdl.After(); err != nil {
		log.Error(err, "failed to execute after process")
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
			"ErrAfterProcess", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
		}
		return reconcile.Result{Requeue: true}, err
	}
	// 获取处理器中的 Pod 列表并更新状态
	pods, err := hdl.ListPods()
	if err != nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse,
			"ErrListPods", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
		}
		return reconcile.Result{Requeue: true}, err
	}
	// 生成并收集状态信息
	mgr.GenerateStatus(pods)
	mgr.CollectStatus()
	// 更新状态信息到 RbdComponent 对象
	if err := mgr.UpdateStatus(); err != nil {
		log.Error(err, "update rainbond component status failure %s")
	}
	// 如果 RbdComponent 不可用，则返回需要重新排队
	if !mgr.IsRbdComponentReady() {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	// 处理完毕，返回空结果
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RbdComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rainbondv1alpha1.RbdComponent{}).
		Complete(r)
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
