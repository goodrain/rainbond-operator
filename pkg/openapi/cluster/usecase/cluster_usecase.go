package usecase

import (
	"fmt"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterUsecaseImpl cluster usecase impl
type ClusterUsecaseImpl struct {
	cfg              *option.Config
	componentUsecase ComponentUseCase
}

// NewClusterUsecaseImpl new cluster case impl
func NewClusterUsecaseImpl(cfg *option.Config, componentUsecase ComponentUseCase) *ClusterUsecaseImpl {
	return &ClusterUsecaseImpl{cfg: cfg, componentUsecase: componentUsecase}
}

// Uninstall uninstall cluster reset cluster
func (c *ClusterUsecaseImpl) UnInstall() error {
	if err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(c.cfg.Namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "name!=rbd-nfs"}); err != nil {
		log.Error(err, "delete component error")
		return err
	}
	if err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(c.cfg.Namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "name=rbd-nfs"}); err != nil {
		log.Error(err, "delete storage component error")
		return err
	}

	if err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(c.cfg.Namespace).Delete(c.cfg.Rainbondpackage, &metav1.DeleteOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	}

	if err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(c.cfg.Namespace).Delete(c.cfg.ClusterName, &metav1.DeleteOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// Status status
func (c *ClusterUsecaseImpl) Status() (*model.ClusterStatus, error) {
	clusterInfo, err := c.getCluster()
	if err != nil {
		// cluster is not found, means status is waiting
		log.Error(err, "get cluster error")
		if k8sErrors.IsNotFound(err) {
			return &model.ClusterStatus{
				FinalStatus: model.Waiting,
				ClusterInfo: model.ClusterInfo{},
			}, nil
		}
		return nil, err
	}

	// package
	rainbondPackage, err := c.getRainbondPackage()
	if err != nil {
		log.Error(err, "get package error")
		rainbondPackage = nil // if can't find package cr, client will return 404 and empty package info not nil
	}

	components, err := c.componentUsecase.List(false)
	if err != nil {
		log.Error(err, "get component status list error")
	}

	status := c.handleStatus(clusterInfo, rainbondPackage, components)

	return &status, nil
}

// no rainbondcluster cr means cluster status is waiting
// rainbondcluster cr without status parameter means cluster status is initing
// rainbondcluster cr with status parameter means cluster status is setting
// rainbondpackage cr means cluster status is installing or running
// rbdcomponent cr means cluster stauts is installing or running
// all rbdcomponent cr are running means cluster status is running
// rbdcomponent cr has pod with status terminal means cluster status is uninstalling
func (c *ClusterUsecaseImpl) handleStatus(rainbondCluster *rainbondv1alpha1.RainbondCluster, rainbondPackage *rainbondv1alpha1.RainbondPackage, componentStatusList []*v1.RbdComponentStatus) model.ClusterStatus {
	reqLogger := log.WithValues("Namespace", c.cfg.Namespace)

	rainbondClusterStatus := c.handleRainbondClusterStatus(rainbondCluster)
	rainbondPackageStatus := c.handlePackageStatus(rainbondPackage)
	componentStatus := c.handleComponentStatus(componentStatusList)
	reqLogger.Info(fmt.Sprintf("cluster: %s; package: %s; component: %s \n", rainbondClusterStatus.FinalStatus, rainbondPackageStatus.FinalStatus, componentStatus.FinalStatus))
	if componentStatus.FinalStatus == model.UnInstalling {
		rainbondClusterStatus.FinalStatus = model.UnInstalling
		return rainbondClusterStatus
	}
	if rainbondClusterStatus.FinalStatus == model.Waiting {
		return rainbondClusterStatus
	}

	if rainbondClusterStatus.FinalStatus == model.Initing {
		return rainbondClusterStatus
	}

	if rainbondPackageStatus.FinalStatus == model.Setting && componentStatus.FinalStatus == model.Setting {
		reqLogger.Info("setting status")
		rainbondClusterStatus.FinalStatus = model.Setting
		return rainbondClusterStatus
	}

	if componentStatus.FinalStatus == model.Running {
		reqLogger.Info("running status")
		rainbondClusterStatus.FinalStatus = model.Running
		return rainbondClusterStatus
	}

	if rainbondPackageStatus.FinalStatus == model.Installing || componentStatus.FinalStatus == model.Installing {
		reqLogger.Info("installing status")
		rainbondClusterStatus.FinalStatus = model.Installing
		return rainbondClusterStatus
	}

	return rainbondClusterStatus
}

func (c *ClusterUsecaseImpl) handleRainbondClusterStatus(rainbondCluster *rainbondv1alpha1.RainbondCluster) model.ClusterStatus {
	status := model.ClusterStatus{
		FinalStatus: model.Waiting,
		ClusterInfo: model.ClusterInfo{},
	}

	if rainbondCluster == nil {
		return status
	}
	if rainbondCluster.Status == nil {
		status.FinalStatus = model.Initing
		return status
	}
	status.FinalStatus = model.Setting
	//prepare cluster info
	for _, sc := range rainbondCluster.Status.StorageClasses {
		if sc.Name != "rainbondslsc" && sc.Name != "rainbondsssc" {
			status.ClusterInfo.Storage = append(status.ClusterInfo.Storage, model.Storage{
				Name:        sc.Name,
				Provisioner: sc.Provisioner,
			})
		}
	}
	for _, node := range rainbondCluster.Status.NodeAvailPorts {
		status.ClusterInfo.NodeAvailPorts = append(status.ClusterInfo.NodeAvailPorts, model.NodeAvailPorts{
			Ports:    node.Ports,
			NodeIP:   node.NodeIP,
			NodeName: node.NodeName,
		})
	}
	return status
}

func (c *ClusterUsecaseImpl) handlePackageStatus(rainbondPackage *rainbondv1alpha1.RainbondPackage) model.ClusterStatus {
	status := model.ClusterStatus{
		FinalStatus: model.Setting,
	}
	if rainbondPackage == nil {
		return status
	}
	status.FinalStatus = model.Installing
	return status
}

func (c *ClusterUsecaseImpl) handleComponentStatus(componentList []*v1.RbdComponentStatus) model.ClusterStatus {
	status := model.ClusterStatus{
		FinalStatus: model.Setting,
	}
	if len(componentList) == 0 {
		return status
	}
	status.FinalStatus = model.Installing

	readyCount := 0
	terminal := false
	for _, component := range componentList {
		if component.Status == v1.ComponentStatusRunning {
			readyCount += 1
		}
		if component.Status == v1.ComponentStatusTerminating { //TODO terminal uninstalling
			terminal = true
		}
	}
	if terminal {
		status.FinalStatus = model.UnInstalling
		return status
	}
	if readyCount != len(componentList) {
		return status
	}
	status.FinalStatus = model.Running
	return status
}

// Init init
func (c *ClusterUsecaseImpl) Init() error {
	_, err := c.createCluster()
	log.Error(err, "create cluster error")
	return err
}

func (c *ClusterUsecaseImpl) getCluster() (*rainbondv1alpha1.RainbondCluster, error) {
	return c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(c.cfg.Namespace).Get(c.cfg.ClusterName, metav1.GetOptions{})
}

func (c *ClusterUsecaseImpl) createCluster() (*rainbondv1alpha1.RainbondCluster, error) {
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.cfg.Namespace,
			Name:      c.cfg.ClusterName,
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			RainbondImageRepository: "registry.cn-hangzhou.aliyuncs.com/goodrain",
			RainbondShareStorage: rainbondv1alpha1.RainbondShareStorage{
				FstabLine: &rainbondv1alpha1.FstabLine{},
			},
			InstallPackageConfig: rainbondv1alpha1.InstallPackageConfig{
				URL: c.cfg.DownloadURL,
				MD5: c.cfg.DownloadMD5,
			},
			InstallMode: rainbondv1alpha1.InstallationModeWithoutPackage,
		},
	}
	return c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(c.cfg.Namespace).Create(cluster)
}

func (c *ClusterUsecaseImpl) getRainbondPackage() (*rainbondv1alpha1.RainbondPackage, error) {
	return c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(c.cfg.Namespace).Get(c.cfg.Rainbondpackage, metav1.GetOptions{})
}
