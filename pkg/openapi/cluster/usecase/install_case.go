package usecase

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
	version                    = "v5.2-dev"
	defaultRainbondDownloadURL = "192.168.2.222" // TODO fanyangyang download url
	defaultRainbondFilePath    = "/opt/rainbond/rainbond.tar"
	componentClaims            = make([]string, 0)
)

type componentClaim struct {
	namespace string
	name      string
	version   string
}

func init() {
	componentClaims = append(componentClaims, "rbd-app-ui")
	componentClaims = append(componentClaims, "rbd-api")
	componentClaims = append(componentClaims, "rbd-worker")
	componentClaims = append(componentClaims, "rbd-webcli")
	componentClaims = append(componentClaims, "rbd-gateway")
	componentClaims = append(componentClaims, "rbd-monitor")
	componentClaims = append(componentClaims, "rbd-repo")
	componentClaims = append(componentClaims, "rbd-dns")
	componentClaims = append(componentClaims, "rbd-db")
	componentClaims = append(componentClaims, "rbd-mq")
	componentClaims = append(componentClaims, "rbd-chaos")
	componentClaims = append(componentClaims, "rbd-storage")
	componentClaims = append(componentClaims, "rbd-hub")
	componentClaims = append(componentClaims, "rbd-package")
	componentClaims = append(componentClaims, "rbd-node")
}

func parseComponentClaim(claim *componentClaim) *v1alpha1.RbdComponent {
	component := &v1alpha1.RbdComponent{}
	component.Namespace = claim.namespace
	component.Name = claim.name
	component.Spec.Version = claim.version
	component.Spec.LogLevel = "debug"
	component.Spec.Type = claim.name
	return component
}

// InstallCaseGetter install case getter
type InstallCaseGetter interface {
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
		if err := downloadFile(ic.archiveFilePath, ""); err != nil {
			logrus.Errorf("download rainbond file error: %s", err.Error())
			return err // TODO fanyangyang bad smell code, fix it
		}

	} else {
		logrus.Debug("rainbond archive file already exits, do not download again")
	}

	// step 3 create custom resource
	return ic.createComponse(componentClaims...)
}

func (ic *InstallCaseImpl) createComponse(components ...string) error {
	for _, rbdComponent := range components {
		component := &componentClaim{name: rbdComponent, version: version, namespace: ic.namespace}
		data := parseComponentClaim(component)
		// init component
		data.Namespace = ic.namespace
		old, err := ic.rbdClientset.RainbondV1alpha1().RbdComponents(ic.namespace).Get(data.Name, metav1.GetOptions{})
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
			_, err = ic.rbdClientset.RainbondV1alpha1().RbdComponents(ic.namespace).Create(data)
			if err != nil {
				return err
			}
		} else {
			data.ResourceVersion = old.ResourceVersion
			_, err = ic.rbdClientset.RainbondV1alpha1().RbdComponents(ic.namespace).Update(data)
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
func downloadFile(filepath string, downloadURL string) error {
	if filepath == "" {
		filepath = os.Getenv("RBD_ARCHIVE")
		if filepath == "" {
			filepath = defaultRainbondFilePath
		}
	}
	if downloadURL == "" {
		downloadURL = os.Getenv("RBD_DOWNLOAD_URL")
		if downloadURL == "" {
			downloadURL = defaultRainbondDownloadURL
		}
	}
	// Get the data
	resp, err := http.Get(downloadURL)
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
