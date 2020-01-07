package status

import (
	"fmt"
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"k8s.io/klog"
	"net/http"
)

const (
	ImageRepositoryUnavailable = "ImageRepositoryUnavailable"
)

// GenerateRainbondClusterStorageReadyCondition returns storageready condition if the storage is ready, else it
// returns an unstorageready condition.
func GenerateRainbondClusterStorageReadyCondition() rainbondv1alpha1.RainbondClusterCondition {
	// TODO(huangrh): implementation
	return rainbondv1alpha1.RainbondClusterCondition{
		Type:   rainbondv1alpha1.StorageReady,
		Status: rainbondv1alpha1.ConditionTrue,
	}
}

// GenerateRainbondClusterImageRepositoryReadyCondition returns imagerepositoryready condition if the image repository is ready,
// else it returns an unimagerepositoryready condition.
func GenerateRainbondClusterImageRepositoryReadyCondition(config *rainbondv1alpha1.GlobalConfig) rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type: rainbondv1alpha1.StorageReady,
	}

	if config.Spec.ImageHub != nil {
		condition.Status = rainbondv1alpha1.ConditionTrue
		return condition
	}

	domain := config.Spec.RainbondImageRepositoryDomain
	if domain == "" {
		domain = constants.DefImageRepositoryDomain
	}

	res, err := http.Get(domain)
	if err != nil {
		klog.Errorf("Error issuing a GET to %s: %v", domain, err)
		condition.Status = rainbondv1alpha1.ConditionFalse
		condition.Reason = ImageRepositoryUnavailable
		condition.Message = fmt.Sprintf("image repository unavailable: %v", err)
		return condition
	}

	if res.StatusCode != http.StatusOK {
		condition.Status = rainbondv1alpha1.ConditionFalse
		condition.Reason = ImageRepositoryUnavailable
		condition.Message = fmt.Sprintf("image repository unavailable. http status code: %d", res.StatusCode)
		return condition
	}

	condition.Status = rainbondv1alpha1.ConditionTrue
	return condition
}
