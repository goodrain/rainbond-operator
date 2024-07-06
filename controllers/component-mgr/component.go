package componentmgr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/logutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers/handler"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RbdcomponentMgr -
type RbdcomponentMgr struct {
	ctx      context.Context
	client   client.Client
	log      logr.Logger
	recorder record.EventRecorder
	// Rainbond 的CRD资源定义，根据获取到的 RainbondComponent 资源来针对性的处理。
	cpt        *rainbondv1alpha1.RbdComponent
	replicaser handler.Replicaser
}

// NewRbdcomponentMgr -
func NewRbdcomponentMgr(ctx context.Context, client client.Client, recorder record.EventRecorder, log logr.Logger, cpt *rainbondv1alpha1.RbdComponent) *RbdcomponentMgr {
	mgr := &RbdcomponentMgr{
		ctx:      ctx,
		client:   client,
		recorder: recorder,
		log:      log,
		cpt:      cpt,
	}
	return mgr
}

// SetReplicaser -
func (r *RbdcomponentMgr) SetReplicaser(replicaser handler.Replicaser) {
	r.replicaser = replicaser
}

// UpdateStatus 用于更新 rbdcomponent 资源的状态的函数
// 即 rbd-api、rbd-worker 等组件的状态。
func (r *RbdcomponentMgr) UpdateStatus() error {
	// 深拷贝当前状态，避免直接修改原始状态对象
	status := r.cpt.Status.DeepCopy()
	// 确保状态中包含 'RbdComponentReady' 条件
	_, condtion := status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condtion == nil {
		// 如果条件不存在，则创建一个新的 'RbdComponentReady' 条件，初始状态为 false
		condtion = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "", "")
		// 将新的条件添加到状态中
		status.SetCondition(*condtion)
	}
	// 将修改后的状态赋值回 r.cpt.Status
	r.cpt.Status = *status
	// 使用重试机制更新状态，处理可能的并发冲突
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 尝试更新 r.cpt 的状态，如果发生冲突则重试
		return r.client.Status().Update(r.ctx, r.cpt)
	})
}

// SetConfigCompletedCondition -
func (r *RbdcomponentMgr) SetConfigCompletedCondition() {
	condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	_ = r.cpt.Status.UpdateCondition(condition)
}

// SetPackageReadyCondition - 设置 Rainbond 组件的包装状态条件
func (r *RbdcomponentMgr) SetPackageReadyCondition(pkg *rainbondv1alpha1.RainbondPackage) {
	// 如果传入的包装对象为空，则设置为 PackageReady 条件为 True
	if pkg == nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	// 获取包装状态中的 Ready 条件
	_, pkgcondition := pkg.Status.GetCondition(rainbondv1alpha1.Ready)
	// 如果包装状态中 Ready 条件不存在，则设置 PackageReady 条件为 False，表示包装未准备好
	if pkgcondition == nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	// 如果包装状态中 Ready 条件不是 Completed，则设置 PackageReady 条件为 False，并包含错误消息
	if pkgcondition.Status != rainbondv1alpha1.Completed {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", pkgcondition.Message)
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	// 否则，设置 PackageReady 条件为 True，表示包装准备就绪
	condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
	_ = r.cpt.Status.UpdateCondition(condition)
}

// CheckPrerequisites 检查组件的先决条件。
// 如果组件的 Spec.PriorityComponent 设置为 true，则表示优先级高，无需等待 RainbondPackage 完成。
// 否则，需要确保在创建资源之前 RainbondPackage 已完成。
func (r *RbdcomponentMgr) CheckPrerequisites(cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) bool {
	// 如果组件的 Spec.PriorityComponent 设置为 true，则无需等待 RainbondPackage 完成
	if r.cpt.Spec.PriorityComponent {
		return true
	}
	// 否则，需要确保 RainbondPackage 已完成
	if cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeFullOnline {
		if err := checkPackageStatus(pkg); err != nil {
			r.log.V(6).Info(err.Error())
			return false
		}
	}
	return true
}

// GenerateStatus 生成组件的状态信息。
// 根据当前组件的配置和实际运行中的 Pod 列表，更新组件的副本数、就绪副本数和相关的 Pod 信息。
func (r *RbdcomponentMgr) GenerateStatus(pods []corev1.Pod) {
	status := r.cpt.Status.DeepCopy()

	// 设置默认副本数为1，如果在组件的 Spec 中指定了 Replicas，则使用指定的值
	var replicas int32 = 1
	if r.cpt.Spec.Replicas != nil {
		replicas = *r.cpt.Spec.Replicas
	}

	// 如果有 Replicaser 接口实现，可以从中获取副本数
	if r.replicaser != nil {
		if rc := r.replicaser.Replicas(); rc != nil {
			r.log.V(6).Info(fmt.Sprintf("replica from replicaser: %d", *rc))
			replicas = *rc
		}
	}

	// 更新状态中的副本数和就绪副本数
	status.Replicas = replicas
	readyReplicas := func() int32 {
		var result int32
		for _, pod := range pods {
			if k8sutil.IsPodReady(&pod) {
				result++
			}
		}
		return result
	}
	status.ReadyReplicas = readyReplicas()

	// 记录就绪 Pod 的信息到状态中
	r.log.V(5).Info(fmt.Sprintf("rainbond component: %s ready replicas count is %d", r.cpt.GetName(), status.ReadyReplicas))
	var newPods []corev1.LocalObjectReference
	for _, pod := range pods {
		newPod := corev1.LocalObjectReference{
			Name: pod.Name,
		}
		newPods = append(newPods, newPod)
	}
	status.Pods = newPods
	// 如果就绪副本数达到了设置的副本数，则更新组件状态为就绪
	if status.ReadyReplicas >= replicas {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionTrue, "Ready", "")
		status.UpdateCondition(condition)
	}
	// 更新组件的状态信息
	r.cpt.Status = *status
}

// CollectStatus -
// CollectStatus 收集并发送组件的状态信息到日志服务。
// 如果未设置环境变量 "ENTERPRISE_ID" 或者设置了 "DISABLE_LOG" 为 "true"，则直接返回。
// 否则，收集 Docker 信息、集群状态信息和区域状态信息，并将其组织成日志数据发送到日志服务。
func (r *RbdcomponentMgr) CollectStatus() {
	// 如果未设置企业 ID 或者禁用日志记录，则直接返回
	if os.Getenv("ENTERPRISE_ID") == "" || os.Getenv("DISABLE_LOG") == "true" {
		return
	}

	// 获取 Docker 信息
	dockerInfo, err := logutil.GetDockerInfo()
	kernelVersion := "UnKnown"
	if dockerInfo != nil {
		kernelVersion = dockerInfo.Server.KernelVersion
	}

	// 获取节点列表
	nodes := corev1.NodeList{}
	err = r.client.List(context.Background(), &nodes)
	clusterStatus := "ClusterFailed"
	var clusterVersionSuffix string

	// 获取集群版本信息
	clusterVersion, _ := k8sutil.GetClientSet().Discovery().ServerVersion()
	if clusterVersion != nil {
		clusterVersionSuffix = clusterVersion.GitVersion
	}
	clusterStatus = clusterStatus + "-" + clusterVersionSuffix

	// 如果能够获取节点列表，则检查节点的状态，更新集群状态
	if err == nil {
		for _, node := range nodes.Items {
			for _, con := range node.Status.Conditions {
				if con.Type == corev1.NodeReady && con.Status == corev1.ConditionTrue {
					clusterStatus = "ClusterReady" + "-" + clusterVersionSuffix
					break
				}
			}
		}
	}

	// 处理区域状态信息
	regionStatus := "RegionFailed"
	regionPods, isReady := handleRegionInfo()
	if isReady {
		regionStatus = "RegionReady"
	}

	// 如果设置了 CONSOLE_DOMAIN 环境变量，则附加 "-docking" 到区域状态
	if os.Getenv("CONSOLE_DOMAIN") != "" {
		regionStatus += "-docking"
	}

	// 组织日志数据
	log := &logutil.LogCollectRequest{
		EID:           os.Getenv("ENTERPRISE_ID"),
		Version:       os.Getenv("INSTALL_VERSION"),
		OS:            runtime.GOOS,
		OSArch:        runtime.GOARCH,
		DockerInfo:    dockerInfo,
		KernelVersion: kernelVersion,
		ClusterInfo:   &logutil.ClusterInfo{Status: clusterStatus},
		RegionInfo:    &logutil.RegionInfo{Status: regionStatus, Pods: regionPods},
	}

	// 发送日志数据到日志服务
	logutil.SendLog(log)
}

// getPodLogs 获取指定容器的日志。
// pod: 要获取日志的 Pod 对象。
// containerName: 要获取日志的容器名称。
// 返回值为字符串形式的容器日志。
func getPodLogs(pod corev1.Pod, containerName string) string {
	// 定义获取容器日志的选项，限制返回行数为30行
	var lines int64 = 30
	podLogOpts := corev1.PodLogOptions{TailLines: &lines, Container: containerName}

	// 获取 Kubernetes 客户端
	clientset := k8sutil.GetClientSet()

	// 创建获取日志的请求
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)

	// 通过流式传输获取日志
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "Get pod log error!"
	}
	defer podLogs.Close()

	// 创建缓冲区来存储日志内容
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "Store pod log to buffer error!"
	}

	// 将日志内容转换为字符串并返回
	str := buf.String()
	return str
}

// handleRegionInfo 处理区域信息，包括获取 Pod 状态、日志和事件。
// 返回值为区域内的 Pod 列表和区域是否就绪的布尔值。
func handleRegionInfo() (regionPods []*logutil.Pod, isReady bool) {
	// 获取 Kubernetes 客户端
	clientSet := k8sutil.GetClientSet()

	// 获取 Pod 列表
	podList, err := clientSet.CoreV1().Pods(rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace)).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, false
	}

	// 存储 Pod 信息的映射
	podInfos := make(map[string]*logutil.Pod)

	// 遍历 Pod 列表，处理每个 Pod 的状态和事件信息
	for index, po := range podList.Items {
		podName := po.Name
		var events []*logutil.PodEvent
		var readyContainers int
		var failureContainerLog string

		// 创建 Pod 对象
		pod := &logutil.Pod{
			Name:   podName,
			Status: string(po.Status.Phase),
		}

		// 遍历 Pod 的容器状态，检查容器是否就绪并获取失败的容器日志
		for i, container := range po.Status.ContainerStatuses {
			if container.Ready {
				readyContainers++
			} else {
				failureContainerLog = getPodLogs(podList.Items[index], po.Status.ContainerStatuses[i].Name)
			}
		}

		// 设置 Pod 的就绪状态和失败日志
		pod.Ready = fmt.Sprintf("%d/%d", readyContainers, len(po.Status.ContainerStatuses))
		pod.Log = failureContainerLog

		// 获取 Pod 的事件列表
		eventLists, err := clientSet.CoreV1().Events(rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace)).List(context.Background(), metav1.ListOptions{FieldSelector: fields.Set{"involvedObject.name": podName}.String()})
		if err != nil {
			return nil, false
		}

		// 处理每个事件并添加到 Pod 的事件列表中
		for _, eve := range eventLists.Items {
			event := &logutil.PodEvent{
				Type:    eve.Type,
				Reason:  eve.Reason,
				Message: eve.Message,
				From:    eve.Source.Component,
				Age:     eve.LastTimestamp.Time.Sub(eve.FirstTimestamp.Time).String(),
			}
			events = append(events, event)
			podInfos[podName].Events = events
		}

		// 将 Pod 信息存储到映射中
		podInfos[podName] = pod
	}

	// 处理完毕后，将 Pod 信息转换为列表形式返回，并判断区域是否就绪
	var pods []*logutil.Pod
	for _, value := range podInfos {
		pods = append(pods, value)
		if strings.Contains(value.Name, "rbd-api") {
			if value.Status == string(corev1.PodRunning) && value.Ready == "1/1" {
				return pods, true
			}
		}
	}

	return pods, false
}

// IsRbdComponentReady 检查 Rainbond 组件是否准备就绪。
// 返回值为布尔值，表示组件是否准备就绪。
func (r *RbdcomponentMgr) IsRbdComponentReady() bool {
	_, condition := r.cpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condition == nil {
		return false
	}

	// 检查组件的状态和就绪副本数是否匹配
	return condition.Status == corev1.ConditionTrue && r.cpt.Status.ReadyReplicas == r.cpt.Status.Replicas
}

// ResourceCreateIfNotExists 如果资源不存在，则创建资源。
// obj: 要创建的客户端对象。
// 返回错误（如果有），表示创建过程中的任何错误。
func (r *RbdcomponentMgr) ResourceCreateIfNotExists(obj client.Object) error {
	// 尝试获取已存在的资源
	err := r.client.Get(r.ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	if err != nil {
		// 如果资源不存在，则创建新资源
		if !k8sErrors.IsNotFound(err) {
			return err
		}
		r.log.V(4).Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", obj.GetNamespace(), "Name", obj.GetName())
		return r.client.Create(r.ctx, obj)
	}
	return nil
}

// UpdateOrCreateResource 更新或创建资源。
// obj: 要更新或创建的客户端对象。
// 返回值为 reconcile.Result 和错误（如果有）。
func (r *RbdcomponentMgr) UpdateOrCreateResource(obj client.Object) (reconcile.Result, error) {
	// 创建旧对象的副本用于比较
	var oldOjb = reflect.New(reflect.ValueOf(obj).Elem().Type()).Interface().(client.Object)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// 尝试获取已存在的资源
	err := r.client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, oldOjb)
	if err != nil {
		// 如果资源不存在，则创建新资源
		if !k8sErrors.IsNotFound(err) {
			r.log.Error(err, fmt.Sprintf("Failed to get %s", obj.GetObjectKind()))
			return reconcile.Result{}, err
		}
		r.log.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", obj.GetNamespace(), "Name", obj.GetName())
		err = r.client.Create(ctx, obj)
		if err != nil {
			r.log.Error(err, fmt.Sprintf("Failed to create new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", obj.GetNamespace(), "Name", obj.GetName())
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// 如果资源存在且可以更新，则执行更新操作
	if !objectCanUpdate(obj) {
		return reconcile.Result{}, nil
	}

	obj = r.updateRuntimeObject(oldOjb, obj)

	r.log.V(5).Info("Object exists.", "Kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"Namespace", obj.GetNamespace(), "Name", obj.GetName())
	if err := r.client.Update(ctx, obj); err != nil {
		r.log.Error(err, "Failed to update", "Kind", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// updateRuntimeObject 更新运行时对象，以确保不重复创建资源。
// old: 老的客户端对象。
// new: 新的客户端对象。
// 返回更新后的客户端对象。
func (r *RbdcomponentMgr) updateRuntimeObject(old, new client.Object) client.Object {
	// 根据对象类型复制必要字段以进行更新
	if n, ok := new.(*corev1.Service); ok {
		r.log.V(6).Info("copy necessary fields from old service before updating")
		o := old.(*corev1.Service)
		n.ResourceVersion = o.ResourceVersion
		n.Spec.ClusterIP = o.Spec.ClusterIP
		return n
	}

	if n, ok := new.(*v2.ApisixRoute); ok {
		r.log.V(6).Info("copy necessary fields from old ApisixRoute before updating")
		o := old.(*v2.ApisixRoute)
		n.ResourceVersion = o.ResourceVersion
		return n
	}

	if n, ok := new.(*v2.ApisixTls); ok {
		r.log.V(6).Info("copy necessary fields from old ApisixTls before updating")
		o := old.(*v2.ApisixTls)
		n.ResourceVersion = o.ResourceVersion
		return n
	}

	if n, ok := new.(*v2.ApisixGlobalRule); ok {
		r.log.V(6).Info("copy necessary fields from old ApisixRoute before updating")
		o := old.(*v2.ApisixGlobalRule)
		n.ResourceVersion = o.ResourceVersion
		return n
	}

	return new
}

// objectCanUpdate 检查对象是否可以更新。
// obj: 要检查的客户端对象。
// 返回布尔值，表示对象是否可以更新。
func objectCanUpdate(obj client.Object) bool {
	// 检查是否是不可更新的对象类型
	if _, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		return false
	}
	if _, ok := obj.(*corev1.PersistentVolume); ok {
		return false
	}
	if _, ok := obj.(*storagev1.StorageClass); ok {
		return false
	}
	if _, ok := obj.(*batchv1.Job); ok {
		return false
	}

	// 检查特定命名的资源是否可以更新
	if obj.GetName() == "rbd-db" || obj.GetName() == "rbd-etcd" {
		return false
	}

	return true
}

// DeleteResources 删除指定资源。
// deleter: 资源删除器接口。
// 返回 reconcile.Result 和错误（如果有）。
func (r *RbdcomponentMgr) DeleteResources(deleter handler.ResourcesDeleter) (*reconcile.Result, error) {
	// 获取需要删除的资源列表
	resources := deleter.ResourcesNeedDelete()

	// 遍历删除每个资源
	for _, res := range resources {
		if res == nil {
			continue
		}
		if err := r.deleteResourcesIfExists(res); err != nil {
			// 如果删除资源时出错，则更新组件的状态并返回错误
			condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
				corev1.ConditionFalse, "ErrDeleteResource", err.Error())
			changed := r.cpt.Status.UpdateCondition(condition)
			if changed {
				r.recorder.Event(r.cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return &reconcile.Result{Requeue: true}, r.UpdateStatus()
			}
			return &reconcile.Result{}, err
		}
	}

	return nil, nil
}

// deleteResourcesIfExists 删除存在的资源（如果存在）。
// obj: 要删除的客户端对象。
// 返回错误（如果有），表示删除过程中的任何错误。
func (r *RbdcomponentMgr) deleteResourcesIfExists(obj client.Object) error {
	// 删除资源并处理删除操作的结果
	err := r.client.Delete(r.ctx, obj, &client.DeleteOptions{GracePeriodSeconds: commonutil.Int64(0)})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	return nil
}

// checkPackageStatus 检查 Rainbond 包的状态是否完成。
// pkg: 要检查的 Rainbond 包对象。
// 返回错误（如果包未完成）。
func checkPackageStatus(pkg *rainbondv1alpha1.RainbondPackage) error {
	var packageCompleted bool

	// 检查包的条件，判断是否已完成
	for _, cond := range pkg.Status.Conditions {
		if cond.Type == rainbondv1alpha1.Ready && cond.Status == rainbondv1alpha1.Completed {
			packageCompleted = true
			break
		}
	}

	// 如果包未完成，则返回错误
	if !packageCompleted {
		return errors.New("rainbond package is not completed in InstallationModeWithoutPackage mode")
	}

	return nil
}
