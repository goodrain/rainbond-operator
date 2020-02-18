package usecase

import (
	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
)

// CaseImpl case
type CaseImpl struct {
	componentUseCaseImpl    cluster.ComponentUseCase
	globalConfigUseCaseImpl cluster.GlobalConfigUseCase
	installCaseImpl         cluster.InstallUseCase
	clusterImpl             cluster.ClusterUseCase
}

// NewClusterCase new cluster case
func NewClusterCase(conf *option.Config, repo cluster.Repository, rainbondKubeClient versioned.Interface) cluster.IClusterCase {
	clusterCase := &CaseImpl{}
	clusterCase.componentUseCaseImpl = NewComponentUsecase(conf)
	clusterCase.globalConfigUseCaseImpl = NewGlobalConfigUseCase(conf)
	clusterCase.installCaseImpl = NewInstallUseCase(conf, rainbondKubeClient, clusterCase.componentUseCaseImpl)
	clusterCase.clusterImpl = NewClusterUsecaseImpl(conf, repo, clusterCase.componentUseCaseImpl)
	return clusterCase
}

// Components components
func (c *CaseImpl) Components() cluster.ComponentUseCase {
	return c.componentUseCaseImpl
}

// GlobalConfigs config
func (c *CaseImpl) GlobalConfigs() cluster.GlobalConfigUseCase {
	return c.globalConfigUseCaseImpl
}

// Install install
func (c *CaseImpl) Install() cluster.InstallUseCase {
	return c.installCaseImpl
}

func (c *CaseImpl) Cluster() cluster.ClusterUseCase {
	return c.clusterImpl
}
