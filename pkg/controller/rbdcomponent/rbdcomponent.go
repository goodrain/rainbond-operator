package rbdcomponent

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent/handler"
	chandler "github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent/handler"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type rbdcomponentMgr struct {
	ctx      context.Context
	client   client.Client
	log      logr.Logger
	recorder record.EventRecorder

	cpt        *rainbondv1alpha1.RbdComponent
	replicaser handler.Replicaser
}

func newRbdcomponentMgr(ctx context.Context, client client.Client, recorder record.EventRecorder, log logr.Logger, cpt *rainbondv1alpha1.RbdComponent) *rbdcomponentMgr {
	mgr := &rbdcomponentMgr{
		ctx:      ctx,
		client:   client,
		recorder: recorder,
		log:      log,
		cpt:      cpt,
	}
	return mgr
}

func (r *rbdcomponentMgr) setReplicaser(replicaser handler.Replicaser) {
	r.replicaser = replicaser
}

func (r *rbdcomponentMgr) updateStatus() error {
	status := r.cpt.Status.DeepCopy()
	// make sure status has ready conditoin
	_, condtion := status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condtion == nil {
		condtion = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, "", "")
		status.SetCondition(*condtion)
	}
	r.cpt.Status = status

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.client.Status().Update(r.ctx, r.cpt)
	})
}

func (r *rbdcomponentMgr) setConfigCompletedCondition() {
	condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	_ = r.cpt.Status.UpdateCondition(condition)
	return
}

func (r *rbdcomponentMgr) setPackageReadyCondition(pkg *rainbondv1alpha1.RainbondPackage) {
	if pkg == nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	if pkg.Status == nil {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", "no status in rainbond package")
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	_, pkgcondition := pkg.Status.GetCondition(rainbondv1alpha1.Ready)
	if pkgcondition.Status != rainbondv1alpha1.Completed {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", pkgcondition.Message)
		_ = r.cpt.Status.UpdateCondition(condition)
		return
	}
	condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
	_ = r.cpt.Status.UpdateCondition(condition)
	return
}

func (r *rbdcomponentMgr) checkPrerequisites(cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) bool {
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

func (r *rbdcomponentMgr) generateStatus(pods []corev1.Pod) {
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

	var newPods []corev1.LocalObjectReference
	for _, pod := range pods {
		newPod := corev1.LocalObjectReference{
			Name: pod.Name,
		}
		newPods = append(newPods, newPod)
	}
	status.Pods = newPods

	if status.ReadyReplicas == replicas {
		condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionTrue, "Ready", "")
		status.UpdateCondition(condition)
	}

	r.cpt.Status = status
}

func (r *rbdcomponentMgr) isRbdComponentReady() bool {
	_, condition := r.cpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condition == nil {
		return false
	}

	return condition.Status == corev1.ConditionTrue && r.cpt.Status.ReadyReplicas == r.cpt.Status.Replicas
}

func (r *rbdcomponentMgr) resourceCreateIfNotExists(obj runtime.Object, meta metav1.Object) error {
	err := r.client.Get(r.ctx, types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
		r.log.V(4).Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		return r.client.Create(r.ctx, obj)
	}
	return nil
}

func (r *rbdcomponentMgr) updateOrCreateResource(obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
	oldOjb := obj.DeepCopyObject()
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, oldOjb)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			r.log.Error(err, fmt.Sprintf("Failed to get %s", obj.GetObjectKind()))
			return reconcile.Result{}, err
		}
		r.log.Info(fmt.Sprintf("Creating a new %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			r.log.Error(err, "Failed to create new", obj.GetObjectKind(), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
			return reconcile.Result{}, err
		}
		// daemonset created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	if !objectCanUpdate(obj) {
		return reconcile.Result{}, nil
	}

	obj = r.updateRuntimeObject(oldOjb, obj)

	r.log.V(5).Info("Object exists.", "Kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"Namespace", meta.GetNamespace(), "Name", meta.GetName())
	if err := r.client.Update(context.TODO(), obj); err != nil {
		r.log.Error(err, "Failed to update", "Kind", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *rbdcomponentMgr) updateRuntimeObject(old, new runtime.Object) runtime.Object {
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

func objectCanUpdate(obj runtime.Object) bool {
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
	return true
}

func (r *rbdcomponentMgr) deleteResources(deleter chandler.ResourcesDeleter) (*reconcile.Result, error) {
	resources := deleter.ResourcesNeedDelete()
	for _, res := range resources {
		if res == nil {
			continue
		}
		if err := r.deleteResourcesIfExists(res.(runtime.Object)); err != nil {
			condition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady,
				corev1.ConditionFalse, "ErrDeleteResource", err.Error())
			changed := r.cpt.Status.UpdateCondition(condition)
			if changed {
				r.recorder.Event(r.cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return &reconcile.Result{Requeue: true}, r.updateStatus()
			}
			return &reconcile.Result{}, err
		}
	}
	return nil, nil
}

func (r *rbdcomponentMgr) deleteResourcesIfExists(obj runtime.Object) error {
	err := r.client.Delete(r.ctx, obj, &client.DeleteOptions{GracePeriodSeconds: commonutil.Int64(0)})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	return nil
}
