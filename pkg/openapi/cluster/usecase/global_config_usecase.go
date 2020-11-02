package usecase

import (
	"fmt"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"strconv"
	"time"

	"github.com/goodrain/rainbond-operator/cmd/openapi/config"
	v1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/retryutil"
	"github.com/goodrain/rainbond-operator/pkg/util/suffixdomain"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/retry"
)

// GlobalConfigUseCaseImpl case
type GlobalConfigUseCaseImpl struct {
	cfg *config.Config
}

// NewGlobalConfigUseCase new global config case
func NewGlobalConfigUseCase(cfg *config.Config) cluster.GlobalConfigUseCase {
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
func (cc *GlobalConfigUseCaseImpl) UpdateGlobalConfig(data *v1.GlobalConfigs) error {
	newCluster, err := cc.formatRainbondClusterConfig(data)
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		oldCluster, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(newCluster.Name, metav1.GetOptions{})
		if err != nil {
			log.Info("get new cluster before update cluster", "warning", err)
		} else {
			newCluster.ResourceVersion = oldCluster.ResourceVersion
		}
		_, err = cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Update(newCluster)
		if err != nil {
			return err
		}
		return nil
	})
}

func (cc *GlobalConfigUseCaseImpl) UpdateGatewayIP(gatewayIP string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cls, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(cc.cfg.ClusterName, metav1.GetOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				// ignore not found
				return nil
			}
			return err
		}

		cls.Spec.GatewayIngressIPs = []string{gatewayIP}

		_, err = cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Update(cls)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil
	}

	// TODO: update * domain
	return nil
}

func (cc *GlobalConfigUseCaseImpl) genSuffixHTTPHost(ip string) (domain string, err error) {
	id, auth, err := cc.getOrCreateUUIDAndAuth()
	if err != nil {
		return "", err
	}
	domain, err = suffixdomain.GenerateDomain(ip, id, auth)
	if err != nil {
		return "", err
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
	clusterInfo := &model.GlobalConfigs{
		OnlyInstallRegion: cc.cfg.OnlyInstallRegion,
	}
	if source.Spec.ImageHub != nil {
		clusterInfo.ImageHub = model.ImageHub{
			Domain:    source.Spec.ImageHub.Domain,
			Namespace: source.Spec.ImageHub.Namespace,
			Username:  source.Spec.ImageHub.Username,
			Password:  source.Spec.ImageHub.Password,
		}
	}
	if source.Spec.RegionDatabase != nil {
		clusterInfo.RegionDatabase = model.Database{
			Host:     source.Spec.RegionDatabase.Host,
			Port:     source.Spec.RegionDatabase.Port,
			Username: source.Spec.RegionDatabase.Username,
			Password: source.Spec.RegionDatabase.Password,
		}
	}
	if source.Spec.UIDatabase != nil {
		clusterInfo.UIDatabase = model.Database{
			Host:     source.Spec.UIDatabase.Host,
			Port:     source.Spec.UIDatabase.Port,
			Username: source.Spec.UIDatabase.Username,
			Password: source.Spec.UIDatabase.Password,
		}
	}
	if source.Spec.EtcdConfig != nil {
		clusterInfo.EtcdConfig = model.EtcdConfig{
			Endpoints: source.Spec.EtcdConfig.Endpoints,
			CertInfo:  model.EtcdCertInfo{},
		}
		if source.Spec.EtcdConfig.SecretName != "" {
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
	}

	clusterInfo.HTTPDomain = source.Spec.SuffixHTTPHost

	clusterInfo.GatewayIngressIPs = source.Spec.GatewayIngressIPs

	return clusterInfo, nil
}

// get old config and then set into new
func (cc *GlobalConfigUseCaseImpl) formatRainbondClusterConfig(source *v1.GlobalConfigs) (*v1alpha1.RainbondCluster, error) {
	old, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(cc.cfg.Namespace).Get(cc.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, bcode.NotFound
		}
		return nil, err
	}

	clusterInfo := old.DeepCopy()
	clusterInfo.Spec.ConfigCompleted = true
	clusterInfo.Spec.EnableHA = source.EnableHA

	clusterInfo.Spec.ImageHub = nil
	if source.ImageHub.Domain != "" {
		clusterInfo.Spec.ImageHub = &v1alpha1.ImageHub{
			Domain:    source.ImageHub.Domain,
			Username:  source.ImageHub.Username,
			Password:  source.ImageHub.Password,
			Namespace: source.ImageHub.Namespace,
		}
	}

	clusterInfo.Spec.RegionDatabase = nil
	if source.RegionDatabase.Host != "" {
		clusterInfo.Spec.RegionDatabase = &v1alpha1.Database{
			Host:     source.RegionDatabase.Host,
			Port:     source.RegionDatabase.Port,
			Username: source.RegionDatabase.Username,
			Password: source.RegionDatabase.Password,
		}
	}

	clusterInfo.Spec.UIDatabase = nil
	if source.UIDatabase.Host != "" {
		clusterInfo.Spec.UIDatabase = &v1alpha1.Database{
			Host:     source.UIDatabase.Host,
			Port:     source.UIDatabase.Port,
			Username: source.UIDatabase.Username,
			Password: source.UIDatabase.Password,
		}
	}

	clusterInfo.Spec.EtcdConfig = nil
	if len(source.EtcdConfig.Endpoints) > 0 {
		clusterInfo.Spec.EtcdConfig = &v1alpha1.EtcdConfig{
			Endpoints: source.EtcdConfig.Endpoints,
		}
		if source.EtcdConfig.CertInfo.CertFile != "" && source.EtcdConfig.CertInfo.KeyFile != "" {
			clusterInfo.Spec.EtcdConfig.SecretName = cc.cfg.EtcdSecretName
			if err := cc.updateOrCreateEtcdCertInfo(source.EtcdConfig.CertInfo); err != nil {
				return nil, err
			}
		} else {
			// if update config set etcd that do not use tls, update config, remove etcd cert secret selector
			clusterInfo.Spec.EtcdConfig.SecretName = ""
		}
	}

	if source.HTTPDomain != "" {
		clusterInfo.Spec.SuffixHTTPHost = source.HTTPDomain
	} else {
		// NodesForGateways can not be nil
		ip := source.NodesForGateways[0].InternalIP
		if len(source.GatewayIngressIPs) > 0 && source.GatewayIngressIPs[0] != "" {
			ip = source.GatewayIngressIPs[0]
		}

		err := retryutil.Retry(1*time.Second, 3, func() (bool, error) {
			domain, err := cc.genSuffixHTTPHost(ip)
			if err != nil {
				return false, err
			}
			clusterInfo.Spec.SuffixHTTPHost = domain
			return true, nil
		})
		if err != nil {
			logrus.Warningf("generate suffix http host: %v", err)
			clusterInfo.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
		}
	}

	// must provide all, can't patch
	clusterInfo.Spec.GatewayIngressIPs = source.GatewayIngressIPs

	setNodes := func(nodes []*v1.K8sNode) []*v1alpha1.K8sNode {
		var result []*v1alpha1.K8sNode
		for _, node := range nodes {
			result = append(result, &v1alpha1.K8sNode{
				Name:       node.Name,
				InternalIP: node.InternalIP,
				ExternalIP: node.ExternalIP,
			})
		}
		return result
	}
	clusterInfo.Spec.NodesForGateway = setNodes(source.NodesForGateways)
	clusterInfo.Spec.NodesForChaos = setNodes(source.NodesForChaos)

	clusterInfo.Spec.RainbondVolumeSpecRWX = convertRainbondVolume(source.RainbondVolumes.RWX)
	if source.RainbondVolumes.RWO != nil {
		clusterInfo.Spec.RainbondVolumeSpecRWO = convertRainbondVolume(source.RainbondVolumes.RWO)
	}

	return clusterInfo, nil
}

func convertRainbondVolume(rv *v1.RainbondVolume) *v1alpha1.RainbondVolumeSpec {
	var storageRequest int32 = 1
	spec := v1alpha1.RainbondVolumeSpec{
		StorageClassName: rv.StorageClassName,
	}
	if rv.StorageClassParameters != nil {
		spec.StorageClassParameters = &v1alpha1.StorageClassParameters{
			Provisioner: rv.StorageClassParameters.Provisioner,
			Parameters:  rv.StorageClassParameters.Parameters,
		}
	}

	if rv.CSIPlugin != nil {
		csiplugin := &v1alpha1.CSIPluginSource{}
		switch {
		case rv.CSIPlugin.AliyunCloudDisk != nil:
			csiplugin.AliyunCloudDisk = &v1alpha1.AliyunCloudDiskCSIPluginSource{
				AccessKeyID:      rv.CSIPlugin.AliyunCloudDisk.AccessKeyID,
				AccessKeySecret:  rv.CSIPlugin.AliyunCloudDisk.AccessKeySecret,
				MaxVolumePerNode: strconv.Itoa(rv.CSIPlugin.AliyunCloudDisk.MaxVolumePerNode),
			}
			storageRequest = 21
		case rv.CSIPlugin.AliyunNas != nil:
			csiplugin.AliyunNas = &v1alpha1.AliyunNasCSIPluginSource{
				AccessKeyID:     rv.CSIPlugin.AliyunNas.AccessKeyID,
				AccessKeySecret: rv.CSIPlugin.AliyunNas.AccessKeySecret,
			}
		case rv.CSIPlugin.NFS != nil:
			csiplugin.NFS = &v1alpha1.NFSCSIPluginSource{}
		}
		spec.CSIPlugin = csiplugin
	}

	spec.StorageRequest = commonutil.Int32(storageRequest)

	return &spec
}

//TODO generate test case
func (cc *GlobalConfigUseCaseImpl) updateOrCreateEtcdCertInfo(certInfo *v1.EtcdCertInfo) error {
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
