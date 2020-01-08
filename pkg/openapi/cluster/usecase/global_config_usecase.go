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
		clusterInfo.ImageHub = model.ImageHub{
			Domain:    source.Spec.ImageHub.Domain,
			Namespace: source.Spec.ImageHub.Namespace,
			Username:  source.Spec.ImageHub.Username,
			Password:  source.Spec.ImageHub.Password,
		}
	} else {
		clusterInfo.ImageHub = model.ImageHub{Default: true}
	}
	if source.Spec.RegionDatabase != nil {
		clusterInfo.RegionDatabase = model.Database{
			Host:     source.Spec.RegionDatabase.Host,
			Port:     source.Spec.RegionDatabase.Port,
			Username: source.Spec.RegionDatabase.Username,
			Password: source.Spec.RegionDatabase.Password,
		}
	} else {
		clusterInfo.RegionDatabase = model.Database{Default: true}
	}
	if source.Spec.UIDatabase != nil {
		clusterInfo.UIDatabase = model.Database{
			Host:     source.Spec.UIDatabase.Host,
			Port:     source.Spec.UIDatabase.Port,
			Username: source.Spec.UIDatabase.Username,
			Password: source.Spec.UIDatabase.Password,
		}
	} else {
		clusterInfo.UIDatabase = model.Database{Default: true}
	}
	if source.Spec.EtcdConfig != nil {
		clusterInfo.EtcdConfig = model.EtcdConfig{
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
		clusterInfo.EtcdConfig = model.EtcdConfig{Default: true}
	}
	gatewayNodes := make([]model.GatewayNode, 0)
	allNode := make(map[string]model.GatewayNode)
	if source.Status != nil && source.Status.NodeAvailPorts != nil {
		for _, node := range source.Status.NodeAvailPorts {
			allNode[node.NodeIP] = model.GatewayNode{NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports}
		}
	}
	if len(allNode) > 0 && source.Spec.GatewayNodes != nil {
		for _, node := range source.Spec.GatewayNodes {
			selected := false
			if _, ok := allNode[node.NodeIP]; ok {
				selected = true
			} else {
			}
			gatewayNodes = append(gatewayNodes, model.GatewayNode{Selected: selected, NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports})
		}
	}

	clusterInfo.GatewayNodes = gatewayNodes
	// model.HTTPDomain{Default: true} // TODO fanyangyang custom http domain
	if source.Spec.SuffixHTTPHost != "" {
		httpDomain := model.HTTPDomain{Default: false}
		httpDomain.Domain = append(httpDomain.Domain, source.Spec.SuffixHTTPHost)
		clusterInfo.HTTPDomain = httpDomain
	} else {
		clusterInfo.HTTPDomain = model.HTTPDomain{Default: true}
	}

	if source.Spec.GatewayIngressIPs != nil {
		clusterInfo.GatewayIngressIPs = append(clusterInfo.GatewayIngressIPs, source.Spec.GatewayIngressIPs...)
	}
	storage := model.Storage{}
	if source.Spec.StorageClassName == "" {
		storage.Default = true
	} else {
		storage.Default = false
		storage.StorageClassName = source.Spec.StorageClassName
	}
	if source.Status != nil && source.Status.StorageClasses != nil {
		for _, sc := range source.Status.StorageClasses {
			storage.Opts = append(storage.Opts, model.StorageOpts{Name: sc.Name, Provisioner: sc.Provisioner})
		}
	}
	clusterInfo.Storage = storage
	return clusterInfo, nil
}

// get old config and then set into new
func (cc *GlobalConfigUseCaseImpl) formatRainbondClusterConfig(source *model.GlobalConfigs) (*v1alpha1.RainbondCluster, error) {
	old, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(cc.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if !source.ImageHub.Default {
		old.Spec.ImageHub = &v1alpha1.ImageHub{
			Domain:    source.ImageHub.Domain,
			Username:  source.ImageHub.Username,
			Password:  source.ImageHub.Password,
			Namespace: source.ImageHub.Namespace,
		}
	} else {
		old.Spec.ImageHub = nil // if use change image hub setting from custom to default, operator will create deefault image hub, api set it vlaue as nil
	}

	if !source.RegionDatabase.Default {
		old.Spec.RegionDatabase = &v1alpha1.Database{
			Host:     source.RegionDatabase.Host,
			Port:     source.RegionDatabase.Port,
			Username: source.RegionDatabase.Username,
			Password: source.RegionDatabase.Password,
		}
	} else {
		old.Spec.RegionDatabase = nil
	}
	if !source.UIDatabase.Default {
		old.Spec.UIDatabase = &v1alpha1.Database{
			Host:     source.UIDatabase.Host,
			Port:     source.UIDatabase.Port,
			Username: source.UIDatabase.Username,
			Password: source.UIDatabase.Password,
		}
	} else {
		old.Spec.UIDatabase = nil
	}
	if !source.EtcdConfig.Default {
		old.Spec.EtcdConfig = &v1alpha1.EtcdConfig{
			Endpoints: source.EtcdConfig.Endpoints,
			UseTLS:    source.EtcdConfig.UseTLS,
		}
		if source.EtcdConfig.UseTLS && source.EtcdConfig.CertInfo != nil {
			if err := cc.updateOrCreateEtcdCertInfo(source.EtcdConfig.CertInfo); err != nil {
				return nil, err
			}
		} else {
			// if update config set etcd that do not use tls, update config, remove etcd cert secret selector
			old.Spec.EtcdConfig.CertSecret = metav1.LabelSelector{}
		}
	} else {
		old.Spec.EtcdConfig = nil
	}
	allNode := make(map[string]*model.GatewayNode)
	if old.Status != nil && old.Status.NodeAvailPorts != nil {
		for _, node := range old.Status.NodeAvailPorts {
			allNode[node.NodeIP] = &model.GatewayNode{NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports}
		}
	}
	if len(allNode) > 0 && source.GatewayNodes != nil {
		nowNodes := make(map[string]struct{})
		for _, node := range old.Spec.GatewayNodes {
			nowNodes[node.NodeIP] = struct{}{}
		}
		for _, node := range source.GatewayNodes {
			if _, ok := allNode[node.NodeIP]; ok {
				if node.Selected {
					if _, ok := nowNodes[node.NodeIP]; !ok {
						old.Spec.GatewayNodes = append(old.Spec.GatewayNodes, v1alpha1.NodeAvailPorts{NodeIP: node.NodeIP})
					}
				}
			}
		}
	}

	if !source.HTTPDomain.Default {
		if len(source.HTTPDomain.Domain) > 0 {
			old.Spec.SuffixHTTPHost = source.HTTPDomain.Domain[0]
		} else {
			old.Spec.SuffixHTTPHost = ""
		}
	} else {
		old.Spec.SuffixHTTPHost = ""
	}

	if source.GatewayIngressIPs != nil {
		nodeIPs := make(map[string]struct{})
		for _, ip := range old.Spec.GatewayIngressIPs {
			nodeIPs[ip] = struct{}{}
		}
		for _, ip := range source.GatewayIngressIPs {
			if _, ok := nodeIPs[ip]; !ok {
				old.Spec.GatewayIngressIPs = append(old.Spec.GatewayIngressIPs, ip)
			}
		}
	} else {
		old.Spec.GatewayIngressIPs = nil
	}

	if !source.Storage.Default {
		old.Spec.StorageClassName = source.Storage.StorageClassName
	} else {
		old.Spec.StorageClassName = ""
	}
	return old, nil
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
