package usecase

import (
	"fmt"
	"strings"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	v1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/util/suffixdomain"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
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

func (cc *GlobalConfigUseCaseImpl) getSuffixHTTPHost(ip string) (domain string, err error) {
	id, auth, err := cc.getOrCreateUUIDAndAuth()
	if err != nil {
		return "", err
	}
	domain, err = suffixdomain.GenerateDomain(ip, id, auth)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(domain, "grapps.cn") {
		return "", fmt.Errorf("get suffix http host failure") // TODO 不能这样做
	}
	return domain, nil
}

func (cc *GlobalConfigUseCaseImpl) getOrCreateUUIDAndAuth() (id, auth string, err error) {
	cm, err := cc.cfg.KubeClient.CoreV1().ConfigMaps(cc.cfg.Namespace).Get(cc.cfg.SuffixHTTPHost, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return "", "", err
	}
	if k8sErrors.IsNotFound(err) {
		logrus.Info("not found configmap, create it")
		cm = generateSuffixConfigMap(cc.cfg.SuffixHTTPHost, cc.cfg.Namespace)
		if _, err = cc.cfg.KubeClient.CoreV1().ConfigMaps(cc.cfg.Namespace).Create(cm); err != nil {
			return "", "", err
		}

	}
	return cm.Data["uuid"], cm.Data["auth"], nil
}

func generateSuffixConfigMap(name, namespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"uuid": string(uuid.NewUUID()),
			"auth": string(uuid.NewUUID()),
		},
	}
	return cm
}

func (cc *GlobalConfigUseCaseImpl) parseRainbondClusterConfig(source *v1alpha1.RainbondCluster) (*model.GlobalConfigs, error) {
	clusterInfo := &model.GlobalConfigs{}
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
			CertInfo:  model.EtcdCertInfo{},
		}
		if source.Spec.EtcdConfig.SecretName != "" {
			clusterInfo.EtcdConfig.UseTLS = true
			etcdSecret, err := cc.cfg.KubeClient.CoreV1().Secrets(cc.cfg.Namespace).Get(source.Spec.EtcdConfig.SecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			clusterInfo.EtcdConfig.CertInfo = model.EtcdCertInfo{
				CaFile:   string(etcdSecret.Data["ca-file"]),
				CertFile: string(etcdSecret.Data["cert-file"]),
				KeyFile:  string(etcdSecret.Data["key-file"]),
			}
		}
	} else {
		clusterInfo.EtcdConfig = model.EtcdConfig{Default: true, CertInfo: model.EtcdCertInfo{}}
	}
	gatewayNodes := make(map[string]model.GatewayNode)
	allNode := make([]model.GatewayNode, 0)
	if source.Status != nil {
		for _, node := range source.Status.NodeAvailPorts {
			allNode = append(allNode, model.GatewayNode{NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports})
		}
	}
	for _, node := range source.Spec.GatewayNodes {
		gatewayNodes[node.NodeIP] = model.GatewayNode{NodeName: node.NodeName, NodeIP: node.NodeIP, Ports: node.Ports}
	}
	for i := range allNode {
		if _, ok := gatewayNodes[allNode[i].NodeIP]; ok {
			allNode[i].Selected = true
		}
	}

	clusterInfo.GatewayNodes = allNode
	if source.Spec.SuffixHTTPHost != "" && !strings.HasSuffix(source.Spec.SuffixHTTPHost, "grapps.cn") {
		httpDomain := model.HTTPDomain{}
		httpDomain.Custom = source.Spec.SuffixHTTPHost
		clusterInfo.HTTPDomain = httpDomain
	} else {
		clusterInfo.HTTPDomain = model.HTTPDomain{Default: true}
	}

	clusterInfo.GatewayIngressIPs = append(clusterInfo.GatewayIngressIPs, source.Spec.GatewayIngressIPs...)

	storage := model.Storage{}
	if source.Spec.StorageClassName == "" {
		storage.Default = true
	} else {
		storage.Default = false
		storage.StorageClassName = source.Spec.StorageClassName
	}
	if source.Status != nil {
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

	clusterInfo := &v1alpha1.RainbondCluster{}
	clusterInfo.ObjectMeta = old.ObjectMeta

	if !source.ImageHub.Default {
		clusterInfo.Spec.ImageHub = &v1alpha1.ImageHub{
			Domain:    source.ImageHub.Domain,
			Username:  source.ImageHub.Username,
			Password:  source.ImageHub.Password,
			Namespace: source.ImageHub.Namespace,
		}
	}

	if !source.RegionDatabase.Default {
		clusterInfo.Spec.RegionDatabase = &v1alpha1.Database{
			Host:     source.RegionDatabase.Host,
			Port:     source.RegionDatabase.Port,
			Username: source.RegionDatabase.Username,
			Password: source.RegionDatabase.Password,
		}
	}
	if !source.UIDatabase.Default {
		clusterInfo.Spec.UIDatabase = &v1alpha1.Database{
			Host:     source.UIDatabase.Host,
			Port:     source.UIDatabase.Port,
			Username: source.UIDatabase.Username,
			Password: source.UIDatabase.Password,
		}
	}
	if !source.EtcdConfig.Default {
		clusterInfo.Spec.EtcdConfig = &v1alpha1.EtcdConfig{
			Endpoints: source.EtcdConfig.Endpoints,
		}
		if source.EtcdConfig.UseTLS {
			clusterInfo.Spec.EtcdConfig.SecretName = cc.cfg.EtcdSecretName
			if err := cc.updateOrCreateEtcdCertInfo(source.EtcdConfig.CertInfo); err != nil {
				return nil, err
			}
		} else {
			// if update config set etcd that do not use tls, update config, remove etcd cert secret selector
			clusterInfo.Spec.EtcdConfig.SecretName = ""
		}
	}

	for _, node := range source.GatewayNodes {
		clusterInfo.Spec.GatewayNodes = append(clusterInfo.Spec.GatewayNodes, v1alpha1.NodeAvailPorts{NodeIP: node.NodeIP})
	}

	if !source.HTTPDomain.Default || source.HTTPDomain.Custom != "" {
		clusterInfo.Spec.SuffixHTTPHost = source.HTTPDomain.Custom
	} else {
		domain, err := cc.getSuffixHTTPHost(clusterInfo.Spec.GatewayNodes[0].NodeIP)
		if err != nil {
			logrus.Warn("get suffix http host error: ", err.Error())
			clusterInfo.Spec.SuffixHTTPHost = "pass.grapps.cn"
		} else {
			clusterInfo.Spec.SuffixHTTPHost = domain
		}
	}

	// must provide all, can't patch
	clusterInfo.Spec.GatewayIngressIPs = append(clusterInfo.Spec.GatewayIngressIPs, source.GatewayIngressIPs...)

	if !source.Storage.Default {
		clusterInfo.Spec.StorageClassName = source.Storage.StorageClassName
	}
	return clusterInfo, nil
}

//TODO generate test case
func (cc *GlobalConfigUseCaseImpl) updateOrCreateEtcdCertInfo(certInfo model.EtcdCertInfo) error {
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

// Address address
func (cc *GlobalConfigUseCaseImpl) Address() (string, error) {
	cluster, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(cc.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	addr := cluster.GatewayIngressIP()
	if addr == "" {
		return "", fmt.Errorf("can't get gatewayIngressIP")
	}

	return fmt.Sprintf("http://%s:7070", addr), nil
}

// Uninstall reset cluster
func (cc *GlobalConfigUseCaseImpl) Uninstall() error {
	components, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var nfscomponent *v1alpha1.RbdComponent
	for i := range components.Items {
		if components.Items[i].Name == "rbd-nfs" {
			nfscomponent = &components.Items[i]
			continue
		}
		err = cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).Delete(components.Items[i].Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	if nfscomponent != nil {
		if err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).Delete(nfscomponent.Name, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(cc.cfg.Namespace).Delete("rainbondpackage", &metav1.DeleteOptions{})
}
