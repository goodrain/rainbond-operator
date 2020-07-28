package usecase

import (
	"github.com/goodrain/rainbond-operator/cmd/openapi/config"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/store"
	"github.com/goodrain/rainbond-operator/pkg/openapi/nodestore"
)

// CaseImpl case
type CaseImpl struct {
	componentUseCaseImpl    cluster.ComponentUsecase
	globalConfigUseCaseImpl cluster.GlobalConfigUseCase
	installCaseImpl         cluster.InstallUseCase
	clusterImpl             cluster.Usecase
}

// NewClusterCase new cluster case
func NewClusterCase(conf *config.Config, repo cluster.Repository, rainbondKubeClient versioned.Interface, nodestorer nodestore.Interface, storer store.Storer) cluster.IClusterUcase {
	clusterCase := &CaseImpl{}
	clusterCase.componentUseCaseImpl = NewComponentUsecase(conf, storer)
	clusterCase.globalConfigUseCaseImpl = NewGlobalConfigUseCase(conf)
	clusterCase.clusterImpl = NewClusterUsecase(conf, repo, clusterCase.componentUseCaseImpl, nodestorer)
	clusterCase.installCaseImpl = NewInstallUseCase(conf, rainbondKubeClient, clusterCase.componentUseCaseImpl, clusterCase.clusterImpl)
	return clusterCase
}

// Components components
func (c *CaseImpl) Components() cluster.ComponentUsecase {
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

// Cluster cluster impl
func (c *CaseImpl) Cluster() cluster.Usecase {
	return c.clusterImpl
}
