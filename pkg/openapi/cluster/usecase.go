package cluster

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
)

// IClusterCase cluster case
type IClusterCase interface {
	GlobalConfigs() GlobalConfigUseCase
	Components() ComponentUseCase
	Install() InstallUseCase
	Cluster() ClusterUseCase
}

// ClusterUseCase cluster case
type ClusterUseCase interface {
	Status() (*model.ClusterStatus, error)
	Init() error
	UnInstall() error
}

// GlobalConfigUseCase global config case
type GlobalConfigUseCase interface {
	GlobalConfigs() (*model.GlobalConfigs, error)
	UpdateGlobalConfig(config *model.GlobalConfigs) error
	Address() (string, error)
}

// ComponentUseCase cluster componse case
type ComponentUseCase interface { // TODO: loop call
	Get(name string) (*v1.RbdComponentStatus, error)
	List(isInit bool) ([]*v1.RbdComponentStatus, error)
}

// InstallUseCase cluster install case
type InstallUseCase interface {
	Install() error
	InstallStatus() (model.StatusRes, error)
}
