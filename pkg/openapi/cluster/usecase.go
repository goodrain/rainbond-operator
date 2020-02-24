package cluster

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
)

// IClusterUcase cluster case
type IClusterUcase interface {
	GlobalConfigs() GlobalConfigUseCase
	Components() ComponentUseCase
	Install() InstallUseCase
	Cluster() Usecase
}

// Usecase cluster case
type Usecase interface {
	Status() (*model.ClusterStatus, error)
	Init() error
	UnInstall() error
	StatusInfo() (*v1.ClusterStatusInfo, error)
}

// GlobalConfigUseCase global config case
type GlobalConfigUseCase interface {
	GlobalConfigs() (*model.GlobalConfigs, error)
	UpdateGlobalConfig(config *v1.GlobalConfigs) error
	Address() (string, error)
}

// ComponentUseCase cluster componse case
type ComponentUseCase interface { // TODO: loop call
	Get(name string) (*v1.RbdComponentStatus, error)
	List(isInit bool) ([]*v1.RbdComponentStatus, error)
}

// InstallUseCase cluster install case
type InstallUseCase interface {
	Install(req *v1.ClusterInstallReq) error
	InstallStatus() (model.StatusRes, error)
}
