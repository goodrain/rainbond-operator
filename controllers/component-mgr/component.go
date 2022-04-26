package componentmgr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

//RbdcomponentMgr -
type RbdcomponentMgr struct {
	ctx      context.Context
	client   client.Client
	log      logr.Logger
	recorder record.EventRecorder

	cpt        *rainbondv1alpha1.RbdComponent
	replicaser handler.Replicaser
}

//NewRbdcomponentMgr -
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

//SetReplicaser -
func (r *RbdcomponentMgr) SetReplicaser(replicaser handler.Replicaser) {
	r.replicaser = replicaser
}

//UpdateStatus -
func (r *RbdcomponentMgr) UpdateStatus() error {
	status := r.cpt.Status.DeepCopy()
	// make sure status has ready conditoin
	_, condtion := status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condtion == nil {
		condtion = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "", "")
		status.SetCondition(*condtion)
	}
	r.cpt.Status = *status

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.client.Status().Update(r.ctx, r.cpt)
	})
}

//SetConfigCompletedCondition -
func (r *RbdcomponentMgr) SetConfigCompletedCondition() {
	condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	_ = r.cpt.Status.UpdateCondition(condition)
}

//SetPackageReadyCondition -
func (r *RbdcomponentMgr) SetPackageReadyCondition(pkg *rainbondv1alpha1.RainbondPackage) {
	if pkg == nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	_, pkgcondition := pkg.Status.GetCondition(rainbondv1alpha1.Ready)
	if pkgcondition == nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	if pkgcondition.Status != rainbondv1alpha1.Completed {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", pkgcondition.Message)
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
	_ = r.cpt.Status.UpdateCondition(condition)
}

//CheckPrerequisites -
func (r *RbdcomponentMgr) CheckPrerequisites(cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) bool {
	if r.cpt.Spec.PriorityComponent {
		// If ImageHub is empty, the priority component no need to wait until rainbondpackage is completed.
		return true
	}
	// Otherwise, we have to make sure rainbondpackage is completed before we create the resource.
	if cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeFullOnline {
		if err := checkPackageStatus(pkg); err != nil {
			r.log.V(6).Info(err.Error())
			return false
		}
	}
	return true
}

//GenerateStatus -
func (r *RbdcomponentMgr) GenerateStatus(pods []corev1.Pod) {
	status := r.cpt.Status.DeepCopy()
	var replicas int32 = 1
	if r.cpt.Spec.Replicas != nil {
		replicas = *r.cpt.Spec.Replicas
	}
	if r.replicaser != nil {
		if rc := r.replicaser.Replicas(); rc != nil {
			r.log.V(6).Info(fmt.Sprintf("replica from replicaser: %d", *rc))
			replicas = *rc
		}
	}
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
	r.log.V(5).Info(fmt.Sprintf("rainbond component: %s ready replicas count is %d", r.cpt.GetName(), status.ReadyReplicas))

	var newPods []corev1.LocalObjectReference
	for _, pod := range pods {
		newPod := corev1.LocalObjectReference{
			Name: pod.Name,
		}
		newPods = append(newPods, newPod)
	}
	status.Pods = newPods

	if status.ReadyReplicas >= replicas {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionTrue, "Ready", "")
		status.UpdateCondition(condition)
	}

	r.cpt.Status = *status
}

// CollectStatus -
func (r *RbdcomponentMgr) CollectStatus() {
	if os.Getenv("ENTERPRISE_ID") == "" || os.Getenv("DISABLE_LOG") == "true" {
		return
	}
	dockerInfo, err := logutil.GetDockerInfo()
	kernelVersion := "UnKnown"
	if dockerInfo != nil {
		kernelVersion = dockerInfo.Server.KernelVersion
	}
	nodes := corev1.NodeList{}
	err = r.client.List(context.Background(), &nodes)
	clusterStatus := "ClusterFailed"
	var clusterVersionSuffix string
	clusterVersion, _ := k8sutil.GetClientSet().Discovery().ServerVersion()
	if clusterVersion != nil {
		clusterVersionSuffix = clusterVersion.GitVersion
	}
	clusterStatus = clusterStatus + "-" + clusterVersionSuffix
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

	// handle region status
	regionStatus := "RegionFailed"
	regionPods, isReady := handleRegionInfo()
	if isReady {
		regionStatus = "RegionReady"
	}
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
	logutil.SendLog(log)
}

func getPodLogs(pod corev1.Pod, containerName string) string {
	var lines int64 = 30
	// Define the options of getting container logs
	podLogOpts := corev1.PodLogOptions{TailLines: &lines, Container: containerName}
	// Create the clientset
	clientset := k8sutil.GetClientSet()
	// Generate log request
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	// Get logs from stream
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "Get pod log error!"
	}
	defer podLogs.Close()
	// Make new buffer to store logs
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "Store pod log to buffer error!"
	}
	// Turn logs to string
	str := buf.String()

	return str
}

func handleRegionInfo() (regionPods []*logutil.Pod, isReady bool) {
	clientSet := k8sutil.GetClientSet()
	podList, err := clientSet.CoreV1().Pods(constants.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, false
	}
	podInfos := make(map[string]*logutil.Pod)
	for index, po := range podList.Items {
		podName := po.Name
		var events []*logutil.PodEvent
		var readyContainers int
		var failureContainerLog string
		pod := &logutil.Pod{
			Name:   podName,
			Status: string(po.Status.Phase),
		}
		for i, container := range po.Status.ContainerStatuses {
			if container.Ready {
				readyContainers++
			} else {
				failureContainerLog = getPodLogs(podList.Items[index], po.Status.ContainerStatuses[i].Name)
			}
		}
		pod.Ready = fmt.Sprintf("%d/%d", readyContainers, len(po.Status.ContainerStatuses))
		pod.Log = failureContainerLog
		podInfos[podName] = pod

		eventLists, err := clientSet.CoreV1().Events(constants.Namespace).List(context.Background(), metav1.ListOptions{FieldSelector: fields.Set{"involvedObject.name": podName}.String()})
		if err != nil {
			return nil, false
		}
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
	}

	// handle Pod
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

//IsRbdComponentReady -
func (r *RbdcomponentMgr) IsRbdComponentReady() bool {
	_, condition := r.cpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condition == nil {
		return false
	}

	return condition.Status == corev1.ConditionTrue && r.cpt.Status.ReadyReplicas == r.cpt.Status.Replicas
}

//ResourceCreateIfNotExists -
func (r *RbdcomponentMgr) ResourceCreateIfNotExists(obj client.Object) error {
	err := r.client.Get(r.ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
		r.log.V(4).Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", obj.GetNamespace(), "Name", obj.GetName())
		return r.client.Create(r.ctx, obj)
	}
	return nil
}

//UpdateOrCreateResource -
func (r *RbdcomponentMgr) UpdateOrCreateResource(obj client.Object) (reconcile.Result, error) {
	var oldOjb = reflect.New(reflect.ValueOf(obj).Elem().Type()).Interface().(client.Object)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err := r.client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, oldOjb)
	if err != nil {
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

func (r *RbdcomponentMgr) updateRuntimeObject(old, new client.Object) client.Object {
	// TODO: maybe use patch is better
	// spec.clusterIP: Invalid value: \"\": field is immutable
	if n, ok := new.(*corev1.Service); ok {
		r.log.V(6).Info("copy necessary fields from old service before updating")
		o := old.(*corev1.Service)
		n.ResourceVersion = o.ResourceVersion
		n.Spec.ClusterIP = o.Spec.ClusterIP
		return n
	}
	return new
}

func objectCanUpdate(obj client.Object) bool {
	// do not use 'obj.GetObjectKind().GroupVersionKind().Kind', because it may be empty
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
	if obj.GetName() == "rbd-db" || obj.GetName() == "rbd-etcd" {
		return false
	}
	return true
}

//DeleteResources -
func (r *RbdcomponentMgr) DeleteResources(deleter handler.ResourcesDeleter) (*reconcile.Result, error) {
	resources := deleter.ResourcesNeedDelete()
	for _, res := range resources {
		if res == nil {
			continue
		}
		if err := r.deleteResourcesIfExists(res); err != nil {
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

func (r *RbdcomponentMgr) deleteResourcesIfExists(obj client.Object) error {
	err := r.client.Delete(r.ctx, obj, &client.DeleteOptions{GracePeriodSeconds: commonutil.Int64(0)})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	return nil
}

func checkPackageStatus(pkg *rainbondv1alpha1.RainbondPackage) error {
	var packageCompleted bool
	for _, cond := range pkg.Status.Conditions {
		if cond.Type == rainbondv1alpha1.Ready && cond.Status == rainbondv1alpha1.Completed {
			packageCompleted = true
			break
		}
	}
	if !packageCompleted {
		return errors.New("rainbond package is not completed in InstallationModeWithoutPackage mode")
	}
	return nil
}
