package cluster

import (
	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/clustercase"
	"k8s.io/client-go/kubernetes"
)

// IClusterCase cluster case
type IClusterCase interface {
	clustercase.GlobalConfigCaseGatter
	clustercase.CompnseCaseGatter
	clustercase.InstallCaseGatter
}

// CaseImpl case
type CaseImpl struct {
	normalClientset      *kubernetes.Clientset
	rbdClientset         *versioned.Clientset
	composeCaseImpl      *clustercase.ComponseCaseImpl
	globalConfigCaseImpl *clustercase.GlobalConfigCaseImpl
	installCaseImpl      *clustercase.InstallCaseImpl
	namespace            string
	configName           string
	etcdSecretName       string
	archiveFilePath      string
}

// NewClusterCase new cluster case
func NewClusterCase(namespace, configName, etcdSecretName, archiveFilePath string, normalClientset *kubernetes.Clientset, rbdClientset *versioned.Clientset) IClusterCase {
	clusterCase := &CaseImpl{
		normalClientset: normalClientset,
		rbdClientset:    rbdClientset,
	}
	clusterCase.composeCaseImpl = clustercase.NewComponseCase(namespace, normalClientset, rbdClientset)
	clusterCase.globalConfigCaseImpl = clustercase.NewGlobalConfigCase(namespace, configName, etcdSecretName, normalClientset, rbdClientset)
	clusterCase.installCaseImpl = clustercase.NewInstallCase(namespace, archiveFilePath, configName, normalClientset, rbdClientset)
	return clusterCase
}

// Componses componse
func (c *CaseImpl) Componses() clustercase.ComponseCase {
	return c.composeCaseImpl
}

// GlobalConfigs config
func (c *CaseImpl) GlobalConfigs() clustercase.GlobalConfigCase {
	return c.globalConfigCaseImpl
}

// Install install
func (c *CaseImpl) Install() clustercase.InstallCase {
	return c.installCaseImpl
}
