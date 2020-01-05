package cluster

import (
	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/usecase"
	"k8s.io/client-go/kubernetes"
)

// IClusterCase cluster case
type IClusterCase interface {
	usecase.GlobalConfigCaseGetter
	usecase.CompnseCaseGetter
	usecase.InstallCaseGetter
}

// CaseImpl case
type CaseImpl struct {
	normalClientset      *kubernetes.Clientset
	rbdClientset         *versioned.Clientset
	composeCaseImpl      *usecase.ComponseCaseImpl
	globalConfigCaseImpl *usecase.GlobalConfigCaseImpl
	installCaseImpl      *usecase.InstallCaseImpl
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
	clusterCase.composeCaseImpl = usecase.NewComponseCase(namespace, normalClientset, rbdClientset)
	clusterCase.globalConfigCaseImpl = usecase.NewGlobalConfigCase(namespace, configName, etcdSecretName, normalClientset, rbdClientset)
	clusterCase.installCaseImpl = usecase.NewInstallCase(namespace, archiveFilePath, configName, normalClientset, rbdClientset)
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
