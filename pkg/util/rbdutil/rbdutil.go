package rbdutil

import (
	"fmt"
	"net"
	"path"

	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
)

// LabelsForRainbond returns labels for resources created by rainbond operator.
func LabelsForRainbond(labels map[string]string) map[string]string {
	rbdLabels := map[string]string{
		"creator":  "Rainbond",
		"belongTo": "rainbond-operator",
	}
	for key, val := range labels {
		// rbdLabels has priority over labels
		if rbdLabels[key] != "" {
			continue
		}
		rbdLabels[key] = val
	}
	return rbdLabels
}

// GetImageRepository returns image repository name based on rainbondcluster.
func GetImageRepository(cluster *v1alpha1.RainbondCluster) string {
	if cluster.Spec.ImageHub == nil {
		return constants.DefImageRepository
	}
	return path.Join(cluster.Spec.ImageHub.Domain, cluster.Spec.ImageHub.Namespace)
}

// LabelsForAccessModeRWX returns rainbond labels with access mode rwx.
func LabelsForAccessModeRWX() map[string]string {
	return map[string]string{
		"accessModes": "rwx",
	}
}

// LabelsForAccessModeRWO returns rainbond labels with access mode rwo.
func LabelsForAccessModeRWO() map[string]string {
	return map[string]string{
		"accessModes": "rwo",
	}
}

// FilterNodesWithPortConflicts -
func FilterNodesWithPortConflicts(nodes []*rainbondv1alpha1.K8sNode) []*rainbondv1alpha1.K8sNode {
	var result []*rainbondv1alpha1.K8sNode
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
