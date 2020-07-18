package precheck

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type storage struct {
	ctx    context.Context
	client client.Client
	ns     string
	rwx    *rainbondv1alpha1.RainbondVolumeSpec
}

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

	if s.rwx.StorageClassName != "" {
		// check if pvc exists
		pvc, err := s.getGrdataPVC()
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return s.failConditoin(condition, err)
			}
			// create pvc
			accessModes := []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			}
			pvc = s.pvcForGrdata(accessModes, s.rwx.StorageClassName)
			if err := s.client.Create(s.ctx, pvc); err != nil {
				return s.failConditoin(condition, err)
			}
			// return false immediately. check if pvc is ready next time.
			condition.Status = corev1.ConditionFalse
			return condition
		}

		if !s.isPVCBound(pvc) {
			condition.Status = corev1.ConditionFalse
			return condition
		}
	}
	return condition
}

func (s *storage) isPVCBound(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Status.Phase == corev1.ClaimBound {
		return true
	}
	return false
}

func (s *storage) getGrdataPVC() (*corev1.PersistentVolumeClaim, error) {
	pvc := corev1.PersistentVolumeClaim{}
	err := s.client.Get(s.ctx, types.NamespacedName{Namespace: s.ns, Name: constants.GrDataPVC}, &pvc)
	if err != nil {
		return nil, err
	}
	return &pvc, nil
}

func (s *storage) pvcForGrdata(accessModes []corev1.PersistentVolumeAccessMode, storageClassName string) *corev1.PersistentVolumeClaim {
	labels := rbdutil.LabelsForRainbond(nil)
	return k8sutil.CreatePersistentVolumeClaim(s.ns, constants.GrDataPVC, accessModes, labels, storageClassName, 1)
}

func (s *storage) failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, err error) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "StorageFailed", err)
}
