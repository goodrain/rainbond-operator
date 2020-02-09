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

// ComponentUsecaseImpl cluster
type ClusterUsecaseImpl struct {
	cfg              *option.Config
	componentUsecase ComponentUseCase
}

// NewComponentUsecase new componse case impl
func NewClusterUsecaseImpl(cfg *option.Config, componentUsecase ComponentUseCase) *ClusterUsecaseImpl {
	return &ClusterUsecaseImpl{cfg: cfg, componentUsecase: componentUsecase}
}

// Status status
func (c *ClusterUsecaseImpl) Status() (*model.ClusterStatus, error) {
	clusterInfo, err := c.getCluster()
	if err != nil {
		// cluster is not found, means status is waiting
		log.Error(err, "get cluster error") //TODO fanyangyang 错误统一返回500，提示系统错误，请联系社区支持，待确认问题：找不到资源会不会报错
		if k8sErrors.IsNotFound(err) {
			return &model.ClusterStatus{
				FinalStatus: model.Waiting,
				ClusterInfo: model.ClusterInfo{},
			}, nil
		}
		return nil, err
	}

	// package
	rainbondPackage, _ := c.getRainbondPackage()

	components, _ := c.componentUsecase.List()

	status := c.handleStatus(clusterInfo, rainbondPackage, components)

	return &status, nil
}

//没有rainbondcluster说明状态为等待中
//rainbondcluster的status为空说明状态为初始化中
//有rainbondcluster说明状态应为配置中往上
//有rainbondpackage说明状态应为安装中往上
//有rbdcomponent说明状态应该为安装中往上
//component的状态都完成说明状态应为安装完成
//component的状态有terminal说明状态因为卸载中
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
	// 准备配置可用参数
	for _, sc := range rainbondCluster.Status.StorageClasses {
		status.ClusterInfo.Storage = append(status.ClusterInfo.Storage, model.Storage{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
		})
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
		if component.Status == "Running" { //TODO fanyangyang 定义
			readyCount += 1
		}
		if component.Status == "Terminating" { //terminal卸载中
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
	return err
}

func (c *ClusterUsecaseImpl) getOrCreate() (*rainbondv1alpha1.RainbondCluster, error) {
	clusterInfo, err := c.createCluster()
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			if clusterInfo.Status == nil {
				//正在初始化中
			}
			return c.getCluster()
		}
		return nil, err
	}
	return clusterInfo, err
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
	}
	return c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(c.cfg.Namespace).Create(cluster)
}

func (c *ClusterUsecaseImpl) getRainbondPackage() (*rainbondv1alpha1.RainbondPackage, error) {
	pkg, err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(c.cfg.Namespace).Get(c.cfg.Rainbondpackage, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pkg, err
}
