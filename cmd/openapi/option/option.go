package option

import (
	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
)

// Config config for openapi
type Config struct {
	KubeconfigPath     string
	Namespace          string //= "rbd-system"
	ConfigName         string //= "rbd-globalconfig"
	EtcdSecretName     string //= "rbd-etcd-secret"
	ArchiveFilePath    string //= "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz"
	KubeClient         kubernetes.Interface
	RainbondKubeClient versioned.Interface
}

// APIServer -
type APIServer struct {
	Config
	LogLevel string
}

// AddFlags add flag
func (s *APIServer) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.LogLevel, "log-level", "info", "the api log level")
	fs.StringVar(&s.Config.KubeconfigPath, "kube-config", "/root/.kube/config", "kubernets admin config path, default /root/.kube/config")
	fs.StringVar(&s.Config.Namespace, "rbd-namespace", "rbd-system", "rbd component namespace")
	fs.StringVar(&s.Config.ConfigName, "rbd-config", "rbd-globalconfig", "rbd cluster global config info")
	fs.StringVar(&s.Config.EtcdSecretName, "rbd-etcd", "rbd-etcd-secret", "etcd cluster info saved in secret")
	fs.StringVar(&s.Config.ArchiveFilePath, "rbd-archive", "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz", "rbd base archive file path")
}
