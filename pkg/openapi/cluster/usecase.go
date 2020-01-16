package cluster

import (
	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/usecase"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	v1 "github.com/GLYASAI/rainbond-operator/pkg/openapi/types/v1"
)

// IClusterCase cluster case
type IClusterCase interface {
	GlobalConfigs() GlobalConfigUseCase
	Components() ComponentUseCase
	Install() InstallUseCase
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
	List() ([]*v1.RbdComponentStatus, error)
}

// InstallUseCase cluster install case
type InstallUseCase interface {
	InstallPreCheck() (model.StatusRes, error)
	Install() error
	InstallStatus() (model.StatusRes, error)
}

// CaseImpl case
type CaseImpl struct {
	componentUseCaseImpl    *usecase.ComponentUsecaseImpl
	globalConfigUseCaseImpl *usecase.GlobalConfigUseCaseImpl
	installCaseImpl         *usecase.InstallUseCaseImpl
}

// NewClusterCase new cluster case
func NewClusterCase(conf *option.Config) IClusterCase {
	clusterCase := &CaseImpl{}
	clusterCase.componentUseCaseImpl = usecase.NewComponentUsecase(conf)
	clusterCase.globalConfigUseCaseImpl = usecase.NewGlobalConfigUseCase(conf)
	clusterCase.installCaseImpl = usecase.NewInstallUseCase(conf)
	return clusterCase
}

// Components components
func (c *CaseImpl) Components() ComponentUseCase {
	return c.componentUseCaseImpl
}

// GlobalConfigs config
func (c *CaseImpl) GlobalConfigs() GlobalConfigUseCase {
	return c.globalConfigUseCaseImpl
}

// Install install
func (c *CaseImpl) Install() InstallUseCase {
	return c.installCaseImpl
}
