package rbdutil

import (
	"github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"path"
)

type Labels map[string]string

func (l Labels) WithRainbondLabels() map[string]string {
	rainbondLabels := map[string]string{
		"creator": "Rainbond",
	}
	for k, v := range l {
		rainbondLabels[k] = v
	}
	return rainbondLabels
}

func LabelsForRainbondResource() map[string]string {
	return map[string]string{
		"creator":  "Rainbond",
		"belongTo": "RainbondOperator",
	}
}

// GetStorageClass returns storage class name based on rainbondcluster.
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
