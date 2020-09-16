package cluster

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	corev1 "k8s.io/api/core/v1"
)

// IClusterUcase cluster case
type IClusterUcase interface {
	GlobalConfigs() GlobalConfigUseCase
	Components() ComponentUsecase
	Install() InstallUseCase
	Cluster() Usecase
}

// Usecase cluster case
type Usecase interface {
	PreCheck() (*v1.ClusterPreCheckResp, error)
	Status() (*model.ClusterStatus, error)
	Init() error
	UnInstall() error
	StatusInfo() (*v1.ClusterStatusInfo, error)
	ClusterNodes(query string, runGateway bool) []*v1.K8sNode
	CompleteNodes(nodes []*v1.K8sNode, runGateway bool) ([]*v1.K8sNode, []*v1.K8sNode)
}

// GlobalConfigUseCase global config case
type GlobalConfigUseCase interface {
	GlobalConfigs() (*model.GlobalConfigs, error)
	UpdateGlobalConfig(config *v1.GlobalConfigs) error
	Address() (string, error)
	UpdateGatewayIP(gatewayIP string) error
}

// ComponentUsecase cluster componse case
type ComponentUsecase interface { // TODO: loop call
	Get(name string) (*v1.RbdComponentStatus, error)
	List(isInit bool) ([]*v1.RbdComponentStatus, error)
	ListComponents() []*rainbondv1alpha1.RbdComponent
	ListPodsByComponent(cpn *rainbondv1alpha1.RbdComponent) ([]*corev1.Pod, error)
}

// InstallUseCase cluster install case
type InstallUseCase interface {
	Install() error
	InstallStatus() (model.StatusRes, error)
	RestartPackage() error
}
