package rbdcomponent

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent/handler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type rbdcomponentMgr struct {
	ctx    context.Context
	client client.Client
	log    logr.Logger

	cpt        *rainbondv1alpha1.RbdComponent
	replicaser handler.Replicaser
}

func newRbdcomponentMgr(ctx context.Context, client client.Client, log logr.Logger, cpt *rainbondv1alpha1.RbdComponent) *rbdcomponentMgr {
	mgr := &rbdcomponentMgr{
		ctx:    ctx,
		client: client,
		log:    log,
		cpt:    cpt,
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
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					result++
				}
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

func (r *rbdcomponentMgr) isRbdCmponentReady() bool {
	_, condition := r.cpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if condition == nil {
		return false
	}
	return condition.Status == corev1.ConditionTrue
}
