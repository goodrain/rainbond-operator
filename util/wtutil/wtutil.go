package wtutil

import (
	"fmt"
	"net"
	"path"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong-operator/util/constants"
	corev1 "k8s.io/api/core/v1"
)

// LabelsForWutong returns labels for resources created by wutong operator.
func LabelsForWutong(labels map[string]string) map[string]string {
	wtLabels := map[string]string{
		"creator":  "Wutong",
		"belongTo": "wutong-operator",
	}
	for key, val := range labels {
		// wtLabels has priority over labels
		if wtLabels[key] != "" {
			continue
		}
		wtLabels[key] = val
	}
	return wtLabels
}

// GetImageRepository returns image repository name based on WutongCluster.
func GetImageRepository(cluster *wutongv1alpha1.WutongCluster) string {
	if cluster.Spec.ImageHub == nil {
		return constants.DefImageRepository
	}
	return path.Join(cluster.Spec.ImageHub.Domain, cluster.Spec.ImageHub.Namespace)
}

// LabelsForAccessModeRWX returns wutong labels with access mode rwx.
func LabelsForAccessModeRWX() map[string]string {
	return map[string]string{
		"accessModes": "rwx",
	}
}

// LabelsForAccessModeRWO returns wutong labels with access mode rwo.
func LabelsForAccessModeRWO() map[string]string {
	return map[string]string{
		"accessModes": "rwo",
	}
}

// FilterNodesWithPortConflicts -
func FilterNodesWithPortConflicts(nodes []*wutongv1alpha1.K8sNode) []*wutongv1alpha1.K8sNode {
	var result []*wutongv1alpha1.K8sNode
	gatewayPorts := []int{80, 443, 10254, 18080, 18081, 8443, 6060, 7070}
	for idx := range nodes {
		node := nodes[idx]
		ok := true
		for _, port := range gatewayPorts {
			if isPortOccupied(fmt.Sprintf("%s:%d", node.InternalIP, port)) {
				ok = false
				break
			}
		}
		if ok {
			result = append(result, node)
		}
	}
	return result
}

func isPortOccupied(address string) bool {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()
	return true
}

// FailCondition -
func FailCondition(condition wutongv1alpha1.WutongClusterCondition, reason, msg string) wutongv1alpha1.WutongClusterCondition {
	condition.Status = corev1.ConditionFalse
	condition.Reason = reason
	condition.Message = msg
	return condition
}

// GetImageRepositoryDomain returns image repository domain based on wutongcluster.
func GetImageRepositoryDomain(cluster *wutongv1alpha1.WutongCluster) string {
	if cluster.Spec.ImageHub == nil {
		return constants.DefImageRepository
	}
	return cluster.Spec.ImageHub.Domain
}
