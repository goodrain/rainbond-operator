package usecase

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	corev1 "k8s.io/api/core/v1"
)

func convertClusterConditions(cluster *rainbondv1alpha1.RainbondCluster) []*v1.ClusterPreCheckCondition {
	var resp []*v1.ClusterPreCheckCondition

	// cluster initialized
	clusterInitialized := &v1.ClusterPreCheckCondition{
		Type:   "ClusterInitialized",
		Status: string(corev1.ConditionTrue),
	}
	if cluster == nil {
		clusterInitialized.Reason = "ClusterNotFound"
		clusterInitialized.Message = "rainbond cluster not found"
		resp = append(resp, clusterInitialized)
		return resp
	}
	if cluster.Status == nil {
		clusterInitialized.Reason = "EmptyClusterStatus"
		clusterInitialized.Message = "rainbond cluster status is empty"
		resp = append(resp, clusterInitialized)
		return resp
	}
	resp = append(resp, clusterInitialized)

	for _, cdt := range cluster.Status.Conditions {
		if cdt.Type == rainbondv1alpha1.RainbondClusterConditionTypeImageRepository &&
			cluster.Spec.ImageHub != nil && cluster.Spec.ImageHub.Domain == constants.DefImageRepository {
			continue
		}

		condition := &v1.ClusterPreCheckCondition{
			Type:    string(cdt.Type),
			Status:  string(cdt.Status),
			Reason:  cdt.Reason,
			Message: cdt.Message,
		}
		resp = append(resp, condition)
	}

	return resp
}
