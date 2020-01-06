package cluster

import (
	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/usecase"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
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
}

// ComponentUseCase cluster componse case
type ComponentUseCase interface { // TODO: loop call
	Get(name string) (*model.ComponseInfo, error)
	List() ([]*model.ComponseInfo, error)
}

// InstallUseCase cluster install case
type InstallUseCase interface {
	Install() error
	InstallStatus() (string, error)
}

// CaseImpl case
type CaseImpl struct {
	componentUseCaseImpl    *usecase.ComponentUseCaseImpl
	globalConfigUseCaseImpl *usecase.GlobalConfigUseCaseImpl
	installCaseImpl         *usecase.InstallUseCaseImpl
}

// NewClusterCase new cluster case
func NewClusterCase(conf *option.Config) IClusterCase {
	clusterCase := &CaseImpl{
		componentUseCaseImpl:    usecase.NewComponentUseCase(conf),
		globalConfigUseCaseImpl: usecase.NewGlobalConfigUseCase(conf),
		installCaseImpl:         usecase.NewInstallUseCase(conf),
	}
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
