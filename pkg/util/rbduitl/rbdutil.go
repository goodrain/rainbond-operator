package rbdutil

import (
	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"path"
)

func LabelsForRainbond(labels map[string]string) map[string]string {
	rbdLabels := map[string]string{
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

// GetProvisioner returns storage class name based on rainbondcluster.
func GetStorageClass(cluster *v1alpha1.RainbondCluster) string {
	if cluster.Spec.StorageClassName == "" {
		return constants.DefStorageClass
	}
	return cluster.Spec.StorageClassName
}

// GetImageRepository returns image repository name based on rainbondcluster.
func GetImageRepository(cluster *v1alpha1.RainbondCluster) string {
	if cluster.Spec.ImageHub == nil {
		return constants.DefImageRepository
	}
	return path.Join(cluster.Spec.ImageHub.Domain, cluster.Spec.ImageHub.Namespace)
}

func LabelsForAccessModeRWX() map[string]string {
	return map[string]string{
		"accessModes": "rwx",
	}
}

func LabelsForAccessModeRWO() map[string]string {
	return map[string]string{
		"accessModes": "rwo",
	}
}
