package precheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type storage struct {
	ctx    context.Context
	client client.Client
	ns     string
	rwx    *rainbondv1alpha1.RainbondVolumeSpec
}

// NewStorage -
func NewStorage(ctx context.Context, client client.Client, ns string, rwx *rainbondv1alpha1.RainbondVolumeSpec) PreChecker {
	return &storage{
		ctx:    ctx,
		client: client,
		ns:     ns,
		rwx:    rwx,
	}
}

func (s *storage) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeStorage,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	if s.rwx != nil && s.rwx.StorageClassName != "" {
		if s.rwx.StorageClassName != "" {
			// check if pvc exists
			pvc, err := k8sutil.GetFoobarPVC(s.ctx, s.client, s.ns)
			if err != nil {
				return s.failConditoin(condition, err.Error())
			}

			if !s.isPVCBound(pvc) {
				// list Events
				eventList, err := k8sutil.EventsForPersistentVolumeClaim(pvc)
				if err != nil {
					return s.failConditoin(condition, err.Error())
				}
				return s.failConditoin(condition, eventListToString(eventList))
			}
		}
		return condition
	}

	if s.rwx == nil {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "InProgress"
		condition.Message =
			fmt.Sprintf("precheck for %s is in progress", rainbondv1alpha1.RainbondClusterConditionTypeStorage)
		return condition
	}

	return condition
}

func (s *storage) isPVCBound(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Status.Phase == corev1.ClaimBound {
		return true
	}
	return false
}

func (s *storage) failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, msg string) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "StorageFailed", msg)
}

func eventListToString(eventList *corev1.EventList) string {
	var res []string
	for _, event := range eventList.Items {
		res = append(res, fmt.Sprintf("%s: %s", event.Reason, event.Message))
	}
	return strings.Join(res, ",")
}
