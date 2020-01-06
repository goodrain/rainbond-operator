package cluster

import (
	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/usecase"
)

// IClusterCase cluster case
type IClusterCase interface {
	GlobalConfigCaseGetter
	CompnseCaseGetter
	InstallCaseGetter
}

// GlobalConfigCaseGetter config case getter
type GlobalConfigCaseGetter interface {
	GlobalConfigs() usecase.GlobalConfigCase
}

// CompnseCaseGetter componse case getter
type CompnseCaseGetter interface {
	Componses() usecase.ComponseCase
}

// InstallCaseGetter install case getter
type InstallCaseGetter interface {
	Install() usecase.InstallCase
}

// CaseImpl case
type CaseImpl struct {
	composeCaseImpl      *usecase.ComponseCaseImpl
	globalConfigCaseImpl *usecase.GlobalConfigCaseImpl
	installCaseImpl      *usecase.InstallCaseImpl
}

// NewClusterCase new cluster case
func NewClusterCase(conf option.Config) IClusterCase {
	clusterCase := &CaseImpl{}
	clusterCase.composeCaseImpl = usecase.NewComponseCase(conf)
	clusterCase.globalConfigCaseImpl = usecase.NewGlobalConfigCase(conf)
	clusterCase.installCaseImpl = usecase.NewInstallCase(conf)
	return clusterCase
}

// Componses componse
func (c *CaseImpl) Componses() usecase.ComponseCase {
	return c.composeCaseImpl
}

// GlobalConfigs config
func (c *CaseImpl) GlobalConfigs() usecase.GlobalConfigCase {
	return c.globalConfigCaseImpl
}

// Install install
func (c *CaseImpl) Install() usecase.InstallCase {
	return c.installCaseImpl
}
