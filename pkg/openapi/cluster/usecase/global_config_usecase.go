package usecase

import (
	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	v1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GlobalConfigUseCaseImpl case
type GlobalConfigUseCaseImpl struct {
	cfg *option.Config
}

// NewGlobalConfigUseCase new global config case
func NewGlobalConfigUseCase(cfg *option.Config) *GlobalConfigUseCaseImpl {
	return &GlobalConfigUseCaseImpl{cfg: cfg}
}

// GlobalConfigs global configs
func (cc *GlobalConfigUseCaseImpl) GlobalConfigs() (*model.GlobalConfigs, error) {
	clusterInfo, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(cc.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return cc.parseRainbondClusterConfig(clusterInfo)
}

// UpdateGlobalConfig update gloobal config
func (cc *GlobalConfigUseCaseImpl) UpdateGlobalConfig(data *model.GlobalConfigs) error {
	clusterInfo, err := cc.formatRainbondClusterConfig(data)
	if err != nil {
		return err
	}
	_, err = cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Update(clusterInfo)
	return err
}

func (cc *GlobalConfigUseCaseImpl) parseRainbondClusterConfig(source *v1alpha1.RainbondCluster) (*model.GlobalConfigs, error) {
	clusterInfo := &model.GlobalConfigs{}
	if source == nil {
		return clusterInfo, nil
	}
	if source.Spec.ImageHub != nil {
		clusterInfo.ImageHub = &model.ImageHub{
			Domain:    source.Spec.ImageHub.Domain,
			Namespace: source.Spec.ImageHub.Namespace,
			Username:  source.Spec.ImageHub.Username,
			Password:  source.Spec.ImageHub.Password,
		}
	} else {
		clusterInfo.ImageHub = &model.ImageHub{Default: true}
	}
	if source.Spec.RegionDatabase != nil {
		clusterInfo.RegionDatabase = &model.Database{
			Host:     source.Spec.RegionDatabase.Host,
			Port:     source.Spec.RegionDatabase.Port,
			Username: source.Spec.RegionDatabase.Username,
			Password: source.Spec.RegionDatabase.Password,
		}
	} else {
		clusterInfo.RegionDatabase = &model.Database{Default: true}
	}
	if source.Spec.UIDatabase != nil {
		clusterInfo.UIDatabase = &model.Database{
			Host:     source.Spec.UIDatabase.Host,
			Port:     source.Spec.UIDatabase.Port,
			Username: source.Spec.UIDatabase.Username,
			Password: source.Spec.UIDatabase.Password,
		}
	} else {
		clusterInfo.UIDatabase = &model.Database{Default: true}
	}
	if source.Spec.EtcdConfig != nil {
		clusterInfo.EtcdConfig = &model.EtcdConfig{
			Endpoints: source.Spec.EtcdConfig.Endpoints,
			UseTLS:    source.Spec.EtcdConfig.UseTLS,
		}
		if source.Spec.EtcdConfig.UseTLS {
			etcdSecret, err := cc.cfg.KubeClient.CoreV1().Secrets(cc.cfg.Namespace).Get(cc.cfg.EtcdSecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			certInfo := &model.EtcdCertInfo{}
			clusterInfo.EtcdConfig.CertInfo = certInfo
			certInfo.CaFile = string(etcdSecret.Data["ca-file"]) // TODO fanyangyang etcd secert data key
			certInfo.CertFile = string(etcdSecret.Data["cert-file"])
			certInfo.KeyFile = string(etcdSecret.Data["key-file"])
		}
	} else {
		clusterInfo.EtcdConfig = &model.EtcdConfig{Default: true}
	}
	gatewayNodes := make([]*model.GatewayNode, 0)
	allNode := make(map[string]*model.GatewayNode)
	if source.Status != nil && source.Status.NodeAvailPorts != nil {
		for _, node := range source.Status.NodeAvailPorts {
			allNode[node.NodeIP] = &model.GatewayNode{NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports}
		}
	}
	if source.Spec.GatewayNodes != nil {
		for _, node := range source.Spec.GatewayNodes {
			selected := false
			if _, ok := allNode[node.NodeIP]; ok {
				selected = true
			} else {
			}
			gatewayNodes = append(gatewayNodes, &model.GatewayNode{Selected: selected, NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports})
		}
	}

	clusterInfo.GatewayNodes = gatewayNodes
	// model.HTTPDomain{Default: true} // TODO fanyangyang custom http domain
	if source.Spec.SuffixHTTPHost != "" {
		httpDomain := &model.HTTPDomain{Default: false}
		httpDomain.Domain = append(httpDomain.Domain, source.Spec.SuffixHTTPHost)
		clusterInfo.HTTPDomain = httpDomain
	} else {
		clusterInfo.HTTPDomain = &model.HTTPDomain{Default: true}
	}

	if source.Spec.GatewayIngressIPs != nil {
		clusterInfo.GatewayIngressIPs = append(clusterInfo.GatewayIngressIPs, source.Spec.GatewayIngressIPs...)
	}
	storage := model.Storage{}
	if source.Spec.StorageClassName == "" {
		storage.Default = true
	}
	if source.Status != nil && source.Status.StorageClasses != nil {
		for _, sc := range source.Status.StorageClasses {
			storage.Opts = append(storage.Opts, model.StorageOpts{Name: sc.Name, Provisioner: sc.Provisioner})
		}
	}
	clusterInfo.Storage = &storage
	return clusterInfo, nil
}

// get old config and then set into new
func (cc *GlobalConfigUseCaseImpl) formatRainbondClusterConfig(source *model.GlobalConfigs) (*v1alpha1.RainbondCluster, error) {
	clusterInfoSpec := v1alpha1.RainbondClusterSpec{}
	if source.ImageHub != nil {
		clusterInfoSpec.ImageHub = &v1alpha1.ImageHub{
			Domain:    source.ImageHub.Domain,
			Username:  source.ImageHub.Username,
			Password:  source.ImageHub.Password,
			Namespace: source.ImageHub.Namespace,
		}
	}
	clusterInfoSpec.StorageClassName = source.Storage.StorageClassName
	if source.RegionDatabase != nil {
		clusterInfoSpec.RegionDatabase = &v1alpha1.Database{
			Host:     source.RegionDatabase.Host,
			Port:     source.RegionDatabase.Port,
			Username: source.RegionDatabase.Username,
			Password: source.RegionDatabase.Password,
		}
	}
	if source.UIDatabase != nil {
		clusterInfoSpec.UIDatabase = &v1alpha1.Database{
			Host:     source.UIDatabase.Host,
			Port:     source.UIDatabase.Port,
			Username: source.UIDatabase.Username,
			Password: source.UIDatabase.Password,
		}
	}
	if source.EtcdConfig != nil {
		clusterInfoSpec.EtcdConfig = &v1alpha1.EtcdConfig{
			Endpoints: source.EtcdConfig.Endpoints,
			UseTLS:    source.EtcdConfig.UseTLS,
		}
		if source.EtcdConfig.UseTLS && source.EtcdConfig.CertInfo != nil {
			if err := cc.updateOrCreateEtcdCertInfo(source.EtcdConfig.CertInfo); err != nil {
				return nil, err
			}
		} else {
			// if update config set etcd that do not use tls, update config, remove etcd cert secret selector
			clusterInfoSpec.EtcdConfig.CertSecret = metav1.LabelSelector{}
		}
	}
	clusterInfo := &v1alpha1.RainbondCluster{Spec: clusterInfoSpec}
	clusterInfo.Name = cc.cfg.ClusterName
	clusterInfo.Namespace = cc.cfg.Namespace
	old, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(cc.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	clusterInfo.ResourceVersion = old.ResourceVersion
	return clusterInfo, nil
}

//TODO generate test case
func (cc *GlobalConfigUseCaseImpl) updateOrCreateEtcdCertInfo(certInfo *model.EtcdCertInfo) error {
	old, err := cc.cfg.KubeClient.CoreV1().Secrets(cc.cfg.Namespace).Get(cc.cfg.EtcdSecretName, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			new := &corev1.Secret{}
			new.SetName(cc.cfg.EtcdSecretName)
			new.SetNamespace(cc.cfg.Namespace)
			new.Data = make(map[string][]byte)
			new.Data["ca-file"] = []byte(certInfo.CaFile) // TODO fanyangyang etcd cert secret data key
			new.Data["cert-file"] = []byte(certInfo.CertFile)
			new.Data["key-file"] = []byte(certInfo.KeyFile)
			_, err = cc.cfg.KubeClient.CoreV1().Secrets(cc.cfg.Namespace).Create(new)
			return err
		}
		return err
	}
	old.Data["ca-file"] = []byte(certInfo.CaFile) // TODO fanyangyang etcd cert secret data key
	old.Data["cert-file"] = []byte(certInfo.CertFile)
	old.Data["key-file"] = []byte(certInfo.KeyFile)
	_, err = cc.cfg.KubeClient.CoreV1().Secrets(cc.cfg.Namespace).Update(old)
	return err
}
