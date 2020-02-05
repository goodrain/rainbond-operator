package option

import (
	"fmt"

	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config config for openapi
type Config struct {
	KubeconfigPath     string
	Namespace          string
	ClusterName        string
	EtcdSecretName     string
	DownloadURL        string
	DownloadMD5        string
	ArchiveFilePath    string
	KubeClient         kubernetes.Interface // TODO
	RainbondKubeClient versioned.Interface
	RestConfig         *rest.Config
	SuffixHTTPHost     string // suffix http host configmap name
	KubeCfgSecretName  string
	Rainbondpackage    string
	LogLevel           string
}

// AddFlags add flag
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.LogLevel, "log-level", "info", "the api log level")
	fs.StringVar(&c.KubeconfigPath, "kube-config", "", "kubernets admin config path, default /root/.kube/config")
	fs.StringVar(&c.Namespace, "rbd-namespace", "rbd-system", "rbd component namespace")
	fs.StringVar(&c.ClusterName, "cluster-name", "rainbondcluster", "rbd cluster name")
	fs.StringVar(&c.EtcdSecretName, "rbd-etcd", "rbd-etcd-secret", "etcd cluster info saved in secret")
	fs.StringVar(&c.ArchiveFilePath, "rbd-archive", "/opt/rainbond/pkg/tgz/rainbond-pkg-V5.2-dev.tgz", "rbd base archive file path")
	fs.StringVar(&c.DownloadURL, "rbd-download-url", "", "download rainbond tar")
	fs.StringVar(&c.DownloadMD5, "rbd-download-md5", "c82f8782ee1b71443799ca6182d017ea", "check down rainbond tar md5")
	fs.StringVar(&c.SuffixHTTPHost, "suffix-configmap", "rbd-suffix-host", "rbd suffix http host configmap name")
	fs.StringVar(&c.KubeCfgSecretName, "kube-secret", "kube-cfg-secret", "kubernetes account info used for cadvisor through kubelet")
	fs.StringVar(&c.Rainbondpackage, "rainbond-package-name", "rainbondpackage", "kubernetes rainbondpackage resource name")
}

// SetLog set log
func (a *Config) SetLog() {
	level, err := logrus.ParseLevel(a.LogLevel)
	if err != nil {
		fmt.Println("set log level error." + err.Error())
		return
	}
	logrus.SetLevel(level)
}
