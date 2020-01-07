package cluster

import (
	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/usecase"
)

// IClusterCase cluster case
type IClusterCase interface {
	GlobalConfigUseCaseGetter
	CompnentUseCaseGetter
	InstallUseCaseGetter
}

// GlobalConfigUseCaseGetter config case getter
type GlobalConfigUseCaseGetter interface {
	GlobalConfigs() usecase.GlobalConfigUseCase
}

// CompnentUseCaseGetter componse case getter
type CompnentUseCaseGetter interface {
	Components() usecase.ComponentUseCase
}

// InstallUseCaseGetter install case getter
type InstallUseCaseGetter interface {
	Install() usecase.InstallUseCase
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

// Components componse
func (c *CaseImpl) Components() usecase.ComponentUseCase {
	return c.componentUseCaseImpl
}

// GlobalConfigs config
func (c *CaseImpl) GlobalConfigs() usecase.GlobalConfigUseCase {
	return c.globalConfigUseCaseImpl
}

// Install install
func (c *CaseImpl) Install() usecase.InstallUseCase {
	return c.installCaseImpl
}
