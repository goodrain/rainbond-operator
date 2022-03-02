package componentmgr

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	"github.com/wutong/wutong-operator/controllers/handler"
	"github.com/wutong/wutong-operator/util/commonutil"
	"github.com/wutong/wutong-operator/util/k8sutil"
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

//WutongComponentMgr -
type WutongComponentMgr struct {
	ctx      context.Context
	client   client.Client
	log      logr.Logger
	recorder record.EventRecorder

	cpt        *wutongv1alpha1.WutongComponent
	replicaser handler.Replicaser
}

//NewWutongComponentMgr -
func NewWutongComponentMgr(ctx context.Context, client client.Client, recorder record.EventRecorder, log logr.Logger, cpt *wutongv1alpha1.WutongComponent) *WutongComponentMgr {
	mgr := &WutongComponentMgr{
		ctx:      ctx,
		client:   client,
		recorder: recorder,
		log:      log,
		cpt:      cpt,
	}
	return mgr
}

//SetReplicaser -
func (r *WutongComponentMgr) SetReplicaser(replicaser handler.Replicaser) {
	r.replicaser = replicaser
}

//UpdateStatus -
func (r *WutongComponentMgr) UpdateStatus() error {
	status := r.cpt.Status.DeepCopy()
	// make sure status has ready conditoin
	_, condtion := status.GetCondition(wutongv1alpha1.WutongComponentReady)
	if condtion == nil {
		condtion = wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse, "", "")
		status.SetCondition(*condtion)
	}
	r.cpt.Status = *status

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.client.Status().Update(r.ctx, r.cpt)
	})
}

//SetConfigCompletedCondition -
func (r *WutongComponentMgr) SetConfigCompletedCondition() {
	condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	_ = r.cpt.Status.UpdateCondition(condition)
}

//SetPackageReadyCondition -
func (r *WutongComponentMgr) SetPackageReadyCondition(pkg *wutongv1alpha1.WutongPackage) {
	if pkg == nil {
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongPackageReady, corev1.ConditionTrue, "PackageReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	_, pkgcondition := pkg.Status.GetCondition(wutongv1alpha1.Ready)
	if pkgcondition == nil {
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongPackageReady, corev1.ConditionFalse, "PackageNotReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	if pkgcondition.Status != wutongv1alpha1.Completed {
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongPackageReady, corev1.ConditionFalse, "PackageNotReady", pkgcondition.Message)
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongPackageReady, corev1.ConditionTrue, "PackageReady", "")
	_ = r.cpt.Status.UpdateCondition(condition)
}

//CheckPrerequisites -
func (r *WutongComponentMgr) CheckPrerequisites(cluster *wutongv1alpha1.WutongCluster, pkg *wutongv1alpha1.WutongPackage) bool {
	if r.cpt.Spec.PriorityComponent {
		// If ImageHub is empty, the priority component no need to wait until WutongPackage is completed.
		return true
	}
	// Otherwise, we have to make sure WutongPackage is completed before we create the resource.
	if cluster.Spec.InstallMode != wutongv1alpha1.InstallationModeFullOnline {
		if err := checkPackageStatus(pkg); err != nil {
			r.log.V(6).Info(err.Error())
			return false
		}
	}
	return true
}

//GenerateStatus -
func (r *WutongComponentMgr) GenerateStatus(pods []corev1.Pod) {
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
	r.log.V(5).Info(fmt.Sprintf("wutong component: %s ready replicas count is %d", r.cpt.GetName(), status.ReadyReplicas))

	var newPods []corev1.LocalObjectReference
	for _, pod := range pods {
		newPod := corev1.LocalObjectReference{
			Name: pod.Name,
		}
		newPods = append(newPods, newPod)
	}
	status.Pods = newPods

	if status.ReadyReplicas >= replicas {
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionTrue, "Ready", "")
		status.UpdateCondition(condition)
	}

	r.cpt.Status = *status
}

//IsWutongComponentReady -
func (r *WutongComponentMgr) IsWutongComponentReady() bool {
	_, condition := r.cpt.Status.GetCondition(wutongv1alpha1.WutongComponentReady)
	if condition == nil {
		return false
	}

	return condition.Status == corev1.ConditionTrue && r.cpt.Status.ReadyReplicas == r.cpt.Status.Replicas
}

//ResourceCreateIfNotExists -
func (r *WutongComponentMgr) ResourceCreateIfNotExists(obj client.Object) error {
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
func (r *WutongComponentMgr) UpdateOrCreateResource(obj client.Object) (reconcile.Result, error) {
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
			r.log.Error(err, "Failed to create new", obj.GetObjectKind(), "Namespace", obj.GetNamespace(), "Name", obj.GetName())
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

func (r *WutongComponentMgr) updateRuntimeObject(old, new client.Object) client.Object {
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
	if obj.GetName() == "wt-db" || obj.GetName() == "wt-etcd" {
		return false
	}
	return true
}

//DeleteResources -
func (r *WutongComponentMgr) DeleteResources(deleter handler.ResourcesDeleter) (*reconcile.Result, error) {
	resources := deleter.ResourcesNeedDelete()
	for _, res := range resources {
		if res == nil {
			continue
		}
		if err := r.deleteResourcesIfExists(res); err != nil {
			condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady,
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

func (r *WutongComponentMgr) deleteResourcesIfExists(obj client.Object) error {
	err := r.client.Delete(r.ctx, obj, &client.DeleteOptions{GracePeriodSeconds: commonutil.Int64(0)})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	return nil
}

func checkPackageStatus(pkg *wutongv1alpha1.WutongPackage) error {
	var packageCompleted bool
	for _, cond := range pkg.Status.Conditions {
		if cond.Type == wutongv1alpha1.Ready && cond.Status == wutongv1alpha1.Completed {
			packageCompleted = true
			break
		}
	}
	if !packageCompleted {
		return errors.New("wutong package is not completed in InstallationModeWithoutPackage mode")
	}
	return nil
}
