package status

import (
	"context"
	"crypto/tls"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	appv1 "k8s.io/api/apps/v1"
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

	RainbondPackageNotFound = "RainbondPackageNotFound"
	ErrGetPackage           = "ErrGetPackage"
)

var rainbondPackagePhase2Int = map[string]int{
	string(rainbondv1alpha1.RainbondPackageFailed):     -1,
	string(rainbondv1alpha1.RainbondPackageWaiting):    0,
	string(rainbondv1alpha1.RainbondPackageExtracting): 1,
	string(rainbondv1alpha1.RainbondPackageLoading):    2,
	string(rainbondv1alpha1.RainbondPackagePushing):    3,
	string(rainbondv1alpha1.RainbondPackageCompleted):  4,
}

type Status struct {
	client  client.Client
	cluster *rainbondv1alpha1.RainbondCluster
}

func NewStatus(client client.Client, cluster *rainbondv1alpha1.RainbondCluster) *Status {
	return &Status{
		client:  client,
		cluster: cluster,
	}
}

// GenerateRainbondClusterStorageReadyCondition returns storageready condition if the storage is ready, else it
// returns an unstorageready condition.
func (s *Status) GenerateRainbondClusterStorageReadyCondition() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:   rainbondv1alpha1.StorageReady,
		Status: rainbondv1alpha1.ConditionFalse,
	}

	sts := &appv1.StatefulSet{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Namespace: s.cluster.Namespace, Name: s.cluster.StorageClass()}, sts); err != nil {
		condition.Reason = "ErrGetProvisioner"
		condition.Message = fmt.Sprintf("failed to get provisioner: %v", err)
		return condition
	}

	condition.Status = rainbondv1alpha1.ConditionTrue

	return condition
}

// GenerateRainbondClusterImageRepositoryReadyCondition returns imagerepositoryready condition if the image repository is ready,
// else it returns an unimagerepositoryready condition.
func (s *Status) GenerateRainbondClusterImageRepositoryReadyCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster) rainbondv1alpha1.RainbondClusterCondition {
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
// TODO: merge GenerateRainbondClusterPackageExtractedCondition, GenerateRainbondClusterPackageLoadedCondition and GenerateRainbondClusterImagesPushedCondition
func (s *Status) GenerateRainbondClusterPackageExtractedCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster) rainbondv1alpha1.RainbondClusterCondition {
	if condition := conditionAlreadyTrue(rainbondCluster.Status, rainbondv1alpha1.PackageExtracted); condition != nil {
		return *condition
	}

	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:   rainbondv1alpha1.PackageExtracted,
		Status: rainbondv1alpha1.ConditionFalse,
	}

	if rainbondCluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithoutPackage {
		condition.Status = rainbondv1alpha1.ConditionTrue
		condition.Reason = string(rainbondv1alpha1.InstallationModeWithoutPackage)
		return condition
	}

	pkg := &rainbondv1alpha1.RainbondPackage{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Namespace: rainbondCluster.Namespace, Name: "rainbondpackage"}, pkg); err != nil {
		if errors.IsNotFound(err) {
			condition.Reason = RainbondPackageNotFound
			return condition
		}
		condition.Reason = ErrGetPackage
		condition.Message = fmt.Sprintf("failed to get rainbondpackage: %v", err)
		return condition
	}

	if rainbondPackagePhase2Int[string(pkg.Status.Phase)] <= rainbondPackagePhase2Int[string(rainbondv1alpha1.RainbondPackageExtracting)] {
		condition.Reason = fmt.Sprintf("RainbondPackage%s", string(pkg.Status.Phase))
		return condition
	}

	condition.Status = rainbondv1alpha1.ConditionTrue

	return condition
}

// GenerateRainbondClusterPackageLoadedCondition returns imagesloaded condition if the image repository is ready,
// else it returns an unimagesloaded condition.
func (s *Status) GenerateRainbondClusterImagesLoadedCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster) rainbondv1alpha1.RainbondClusterCondition {
	if condition := conditionAlreadyTrue(rainbondCluster.Status, rainbondv1alpha1.ImagesLoaded); condition != nil {
		return *condition
	}

	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:   rainbondv1alpha1.ImagesLoaded,
		Status: rainbondv1alpha1.ConditionFalse,
	}

	if rainbondCluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithoutPackage {
		condition.Status = rainbondv1alpha1.ConditionTrue
		condition.Reason = string(rainbondv1alpha1.InstallationModeWithoutPackage)
		return condition
	}

	pkg := &rainbondv1alpha1.RainbondPackage{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Namespace: rainbondCluster.Namespace, Name: "rainbondpackage"}, pkg); err != nil {
		condition.Status = rainbondv1alpha1.ConditionFalse
		if errors.IsNotFound(err) {
			condition.Reason = RainbondPackageNotFound
			return condition
		}
		condition.Reason = ErrGetPackage
		condition.Message = fmt.Sprintf("failed to get rainbondpackage: %v", err)
		return condition
	}

	if rainbondPackagePhase2Int[string(pkg.Status.Phase)] <= rainbondPackagePhase2Int[string(rainbondv1alpha1.RainbondPackageLoading)] {
		condition.Reason = fmt.Sprintf("RainbondPackage%s", string(pkg.Status.Phase))
		return condition
	}

	condition.Status = rainbondv1alpha1.ConditionTrue

	return condition
}

// GenerateRainbondClusterImagesPushedCondition returns imagespushed condition if all the images have been pushed,
// else it returns an unimagespushed condition.
func (s *Status) GenerateRainbondClusterImagesPushedCondition(rainbondCluster *rainbondv1alpha1.RainbondCluster) rainbondv1alpha1.RainbondClusterCondition {
	if condition := conditionAlreadyTrue(rainbondCluster.Status, rainbondv1alpha1.ImagesPushed); condition != nil {
		return *condition
	}

	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:   rainbondv1alpha1.ImagesPushed,
		Status: rainbondv1alpha1.ConditionFalse,
	}

	if rainbondCluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithoutPackage {
		condition.Status = rainbondv1alpha1.ConditionTrue
		condition.Reason = string(rainbondv1alpha1.InstallationModeWithoutPackage)
		return condition
	}

	pkg := &rainbondv1alpha1.RainbondPackage{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Namespace: rainbondCluster.Namespace, Name: "rainbondpackage"}, pkg); err != nil {
		condition.Status = rainbondv1alpha1.ConditionFalse
		if errors.IsNotFound(err) {
			condition.Reason = RainbondPackageNotFound
			return condition
		}
		condition.Reason = ErrGetPackage
		condition.Message = fmt.Sprintf("failed to get rainbondpackage: %v", err)
		return condition
	}

	if rainbondPackagePhase2Int[string(pkg.Status.Phase)] <= rainbondPackagePhase2Int[string(rainbondv1alpha1.RainbondPackagePushing)] {
		condition.Reason = fmt.Sprintf("RainbondPackage%s", string(pkg.Status.Phase))
		return condition
	}

	condition.Status = rainbondv1alpha1.ConditionTrue

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
