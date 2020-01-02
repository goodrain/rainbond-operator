package clustercase

import (
	"io"
	"net/http"
	"os"

	v1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	version             = "v5.2-dev"
	rbdAPPUI            = initComponse(version, "rbd-app-ui")
	rbdAPI              = initComponse(version, "rbd-api")
	rbdWorker           = initComponse(version, "rbd-worker")
	rbdWebCli           = initComponse(version, "rbd-webcli")
	rbdGateway          = initComponse(version, "rbd-gateway")
	rbdMonitor          = initComponse(version, "rbd-monitor")
	rbdRepo             = initComponse(version, "rbd-repo") // TODO fanyangyang 是否需要
	rbdDNS              = initComponse(version, "rbd-dns")  // TODO fanyangyang 是否需要
	rbdDB               = initComponse(version, "rbd-db")   // TODO fanyangyang 是否需要
	rbdHUB              = initComponse(version, "rbd-hub")  // TODO fanyangyang 是否需要
	rbdMQ               = initComponse(version, "rbd-mq")
	rbdChaos            = initComponse(version, "rbd-chaos")
	componses           = make([]*v1alpha1.RbdComponent, 0)
	rainbondDownloadURL = "192.168.2.222" // TODO fanyangyang download url
)

func init() {
	componses = append(componses, rbdAPPUI)
	componses = append(componses, rbdAPI)
	componses = append(componses, rbdWorker)
	componses = append(componses, rbdWebCli)
	componses = append(componses, rbdGateway)
	componses = append(componses, rbdMonitor)
	componses = append(componses, rbdRepo)
	componses = append(componses, rbdDNS)
	componses = append(componses, rbdDB)
	componses = append(componses, rbdHUB)
	componses = append(componses, rbdMQ)
	componses = append(componses, rbdChaos)
}

func initComponse(version, typeName string) *v1alpha1.RbdComponent {
	componse := &v1alpha1.RbdComponent{}
	componse.Name = typeName
	componse.Spec.Version = version
	componse.Spec.LogLevel = "debug"
	componse.Spec.Type = typeName
	return componse
}

// InstallCaseGatter install case gatter
type InstallCaseGatter interface {
	Install() InstallCase
}

// InstallCase cluster install case
type InstallCase interface {
	Install() error
	InstallStatus() (string, error)
}

// InstallCaseImpl install case
type InstallCaseImpl struct {
	normalClientset *kubernetes.Clientset
	rbdClientset    *versioned.Clientset
	namespace       string
	archiveFilePath string
	configName      string
}

// NewInstallCase new install case
func NewInstallCase(namespace, archiveFilePath, configName string, normalClientset *kubernetes.Clientset, rbdClientset *versioned.Clientset) *InstallCaseImpl {
	return &InstallCaseImpl{
		normalClientset: normalClientset,
		rbdClientset:    rbdClientset,
		namespace:       namespace,
		archiveFilePath: archiveFilePath,
		configName:      configName,
	}
}

// Install install
func (ic *InstallCaseImpl) Install() error {
	// step 1 check if archive is exists or not
	if _, err := os.Stat(ic.archiveFilePath); os.IsNotExist(err) {
		logrus.Warnf("rainbond archive file does not exists, downloading background ...")

		// step 2 download archive
		if err := downloadFile(ic.archiveFilePath, rainbondDownloadURL); err != nil { // TODO fanyangyang file path
			logrus.Errorf("download rainbond file error: %s", err.Error())
			return err // TODO fanyangyang bad smell code, fix it
		}

	}

	// step 3 create custom resource
	return ic.createComponse()
}

func (ic *InstallCaseImpl) createComponse() error {
	for _, componse := range componses {
		componse.Namespace = ic.namespace
		old, err := ic.rbdClientset.RainbondV1alpha1().RbdComponents(ic.namespace).Get(componse.Name, metav1.GetOptions{})
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
			_, err = ic.rbdClientset.RainbondV1alpha1().RbdComponents(ic.namespace).Create(componse)
			if err != nil {
				return err
			}
		} else {
			componse.ResourceVersion = old.ResourceVersion
			_, err = ic.rbdClientset.RainbondV1alpha1().RbdComponents(ic.namespace).Update(componse)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// InstallStatus install status
func (ic *InstallCaseImpl) InstallStatus() (string, error) {
	configs, err := ic.rbdClientset.RainbondV1alpha1().GlobalConfigs(ic.namespace).Get(ic.configName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	status := "" // TODO fanyangyang if install process is downloading rainbond, what status it is?
	if configs != nil {
		status = string(configs.Status.Phase)
	} else {
		logrus.Warn("cluster config has not be created yet, something occured ? ")
	}
	return status, nil
}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil { // TODO fanyangyang if can't create connection, download manual and upload it
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath) // TODO fanyangyang file path and generate test case
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
