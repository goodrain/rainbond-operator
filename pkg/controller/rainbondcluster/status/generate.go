package status

import (
	"crypto/tls"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rainbondcluster/pkg"
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rainbondcluster/types"
	"net/http"
	"net/url"
	"time"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"k8s.io/klog"
)

const (
	// ImageRepositoryUnavailable means the image repository is nnavailable.
	ImageRepositoryUnavailable = "ImageRepositoryUnavailable"
	// NoGatewayIP means gateway ip not found
	NoGatewayIP = "NoGatewayIP"
	// ErrHistoryFetch means failed to fetching installation package processing history.
	ErrHistoryFetch = "ErrHistoryFetch"
	// ErrGetMetadata means failed to getting installation package metadata.
	ErrGetMetadata = "ErrGetMetadata"
	// NotAllImagesLoaded means not all images has been loaded successfully.
	NotAllImagesLoaded = "NotAllImagesLoaded"
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
func GenerateRainbondClusterImageRepositoryReadyCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster) rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type: rainbondv1alpha1.ImageRepositoryInstalled,
	}

	if rainbondCluster.Spec.ImageHub != nil {
		// TODO(huangrh): custom image repository also needs to be verify.
		condition.Status = rainbondv1alpha1.ConditionTrue
		return condition
	}

	domain := rainbondCluster.Spec.RainbondImageRepositoryDomain
	if domain == "" {
		domain = constants.DefImageRepositoryDomain
	}

	client := &http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	var gatewayIP string
	if len(rainbondCluster.Spec.GatewayNodes) > 0 {
		gatewayIP = rainbondCluster.Spec.GatewayNodes[0].NodeIP
	} else if len(rainbondCluster.Status.NodeAvailPorts) > 0 {
		gatewayIP = rainbondCluster.Status.NodeAvailPorts[0].NodeIP
	} else {
		condition.Status = rainbondv1alpha1.ConditionFalse
		condition.Reason = NoGatewayIP
		condition.Message = fmt.Sprint("gateway ip not found.")
		return condition
	}

	// TODO: check all gateway ips

	u, _ := url.Parse(fmt.Sprintf("https://%s/v2/", gatewayIP))
	request := &http.Request{
		URL:  u,
		Host: domain,
	}
	res, err := client.Do(request)
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

// GenerateRainbondClusterPackageExtractedCondition returns pakcageextracted condition if the image repository is ready,
// else it returns an unpakcageextracted condition.
func GenerateRainbondClusterPackageExtractedCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster, history pkg.HistoryInterface) rainbondv1alpha1.RainbondClusterCondition {
	if condition := conditionAlreadyTrue(rainbondCluster.Status, rainbondv1alpha1.PackageExtracted); condition != nil {
		return *condition
	}

	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type: rainbondv1alpha1.PackageExtracted,
	}

	if rainbondCluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithoutPackage {
		condition.Status = rainbondv1alpha1.ConditionTrue
		condition.Reason = string(rainbondv1alpha1.InstallationModeWithoutPackage)
		return condition
	}

	// get extraction history
	h, err := history.ExtractionHistory()
	if err != nil {
		condition.Status = rainbondv1alpha1.ConditionFalse
		condition.Reason = ErrHistoryFetch
		condition.Message = fmt.Sprintf("Error fetching extraction history: %v", err)
		return condition
	}

	if h.Status == types.HistoryStatusFalse {
		condition.Status = rainbondv1alpha1.ConditionFalse
		condition.Reason = h.Reason
		return condition
	}

	condition.Status = rainbondv1alpha1.ConditionTrue
	condition.Reason = h.Reason

	return condition
}

// GenerateRainbondClusterPackageLoadedCondition returns imagesloaded condition if the image repository is ready,
// else it returns an unimagesloaded condition.
func GenerateRainbondClusterImagesLoadedCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster, packager pkg.PackageInterface) rainbondv1alpha1.RainbondClusterCondition {
	if condition := conditionAlreadyTrue(rainbondCluster.Status, rainbondv1alpha1.ImagesLoaded); condition != nil {
		return *condition
	}

	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type: rainbondv1alpha1.ImagesLoaded,
	}

	if rainbondCluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithoutPackage {
		condition.Status = rainbondv1alpha1.ConditionTrue
		condition.Reason = string(rainbondv1alpha1.InstallationModeWithoutPackage)
		return condition
	}

	images, err := packager.GetMetadata()
	if err != nil {
		condition.Status = rainbondv1alpha1.ConditionFalse
		condition.Reason = ErrGetMetadata
		condition.Message = fmt.Sprintf("Error fetching metadata: %v", err)
		return condition
	}

	// TODO: check if the image exits
	_ = images

	return condition
}

func conditionAlreadyTrue(status *rainbondv1alpha1.RainbondClusterStatus, conditionType rainbondv1alpha1.RainbondClusterConditionType) *rainbondv1alpha1.RainbondClusterCondition {
	// Before calling this function, you must ensure that status is not nil.
	// The extra judgment here is to facilitate testing.
	if status != nil {
		for _, c := range status.Conditions {
			if c.Type == conditionType && c.Status == rainbondv1alpha1.ConditionTrue {
				return c.DeepCopy()
			}
		}
	}

	return nil
}
