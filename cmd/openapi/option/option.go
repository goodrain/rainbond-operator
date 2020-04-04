package option

import (
	"fmt"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config config for openapi
type Config struct {
	RainbondVersion         string
	KubeconfigPath          string
	Namespace               string
	ClusterName             string
	EtcdSecretName          string
	DownloadURL             string
	DownloadMD5             string
	ArchiveFilePath         string
	KubeClient              kubernetes.Interface // TODO
	RainbondKubeClient      versioned.Interface
	RestConfig              *rest.Config
	SuffixHTTPHost          string // suffix http host configmap name
	KubeCfgSecretName       string
	Rainbondpackage         string
	LogLevel                string
	InstallMode             string
	RainbondImageRepository string
	InitPath                string
	JWTSecretKey            string
	JWTExpTime              time.Duration
	DBPath                  string
	OnlyInstallRegion       bool
}

// AddFlags add flag
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.RainbondVersion, "rainbond-version", "V5.2.0-beta1", "The version of Rainbond.")
	fs.StringVar(&c.LogLevel, "log-level", "info", "the api log level")
	fs.StringVar(&c.KubeconfigPath, "kube-config", "", "kubernets admin config path, default /root/.kube/config")
	fs.StringVar(&c.Namespace, "rbd-namespace", "rbd-system", "rbd component namespace")
	fs.StringVar(&c.ClusterName, "cluster-name", "rainbondcluster", "rbd cluster name")
	fs.StringVar(&c.EtcdSecretName, "rbd-etcd", "rbd-etcd-secret", "etcd cluster info saved in secret")
	fs.StringVar(&c.ArchiveFilePath, "rbd-archive", "/opt/rainbond/pkg/tgz/rainbond-pkg-V5.2-dev.tgz", "rbd base archive file path")
	fs.StringVar(&c.DownloadURL, "rbd-download-url", "", "download rainbond tar")
	fs.StringVar(&c.DownloadMD5, "rbd-download-md5", "fcd61975ff0a55fc1a1dd997043488adc14fe7e4fea474f77865a0689b52e1de", "check down rainbond tar md5")
	fs.StringVar(&c.SuffixHTTPHost, "suffix-configmap", "rbd-suffix-host", "rbd suffix http host configmap name")
	fs.StringVar(&c.KubeCfgSecretName, "kube-secret", "kube-cfg-secret", "kubernetes account info used for cadvisor through kubelet")
	fs.StringVar(&c.Rainbondpackage, "rainbond-package-name", "rainbondpackage", "kubernetes rainbondpackage resource name")
	fs.StringVar(&c.InstallMode, "install-mode", "WithPackage", "Rainbond installation mode, install with package, or not.")
	fs.BoolVar(&c.OnlyInstallRegion, "only-region", false, "Only install region, if true, can not install ui.")
	fs.StringVar(&c.RainbondImageRepository, "image-repository", "registry.cn-hangzhou.aliyuncs.com/goodrain", "Image repository for Rainbond components.")
	fs.StringVar(&c.InitPath, "init-path", "/opt/rainbond/.init", "rainbond init file path")
	fs.StringVar(&c.JWTSecretKey, "jwt.secret.key", "123", "secret key for signing jwt token")
	fs.DurationVar(&c.JWTExpTime, "jwt.exp.time", time.Minute*30, "expired time for jwt token")
	fs.StringVar(&c.DBPath, "db.path", "/opt/rainbond/data/.init/operator.db", "sqlite path of operator")
}

// SetLog set log
func (c *Config) SetLog() {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		fmt.Println("set log level error." + err.Error())
		return
	}
	logrus.SetLevel(level)
}
