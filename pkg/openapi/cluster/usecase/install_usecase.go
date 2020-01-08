package usecase

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/sirupsen/logrus"

	v1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	version                    = "V5.2-dev"
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
	// componentClaims = append(componentClaims, "rbd-storage")
	componentClaims = append(componentClaims, "rbd-hub")
	componentClaims = append(componentClaims, "rbd-package")
	componentClaims = append(componentClaims, "rbd-node")
	componentClaims = append(componentClaims, "rbd-etcd")
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

// InstallUseCaseImpl install case
type InstallUseCaseImpl struct {
	cfg *option.Config
}

// NewInstallUseCase new install case
func NewInstallUseCase(cfg *option.Config) *InstallUseCaseImpl {
	return &InstallUseCaseImpl{cfg: cfg}
}

// Install install
func (ic *InstallUseCaseImpl) Install() error {
	// // step 1 check if archive is exists or not
	// if _, err := os.Stat(ic.cfg.ArchiveFilePath); os.IsNotExist(err) {
	// 	logrus.Warnf("rainbond archive file does not exists, downloading background ...")

	// 	// step 2 download archive
	// 	if err := downloadFile(ic.cfg.ArchiveFilePath, ""); err != nil {
	// 		logrus.Errorf("download rainbond file error: %s", err.Error())
	// 		return fmt.Errorf(translate.Translation("download rainbond.tar error, please try again or upload it using /uploads")) // TODO fanyangyang bad smell code, fix it
	// 	}

	// } else {
	// 	logrus.Debug("rainbond archive file already exits, do not download again")
	// }

	// step 3 create custom resource
	return ic.createComponents(componentClaims...)
}

func (ic *InstallUseCaseImpl) createComponents(components ...string) error {
	for _, rbdComponent := range components {
		component := &componentClaim{name: rbdComponent, version: version, namespace: ic.cfg.Namespace}
		data := parseComponentClaim(component)
		// init component
		data.Namespace = ic.cfg.Namespace
		old, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(ic.cfg.Namespace).Get(data.Name, metav1.GetOptions{})
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
			_, err = ic.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(ic.cfg.Namespace).Create(data)
			if err != nil {
				return err
			}
		} else {
			data.ResourceVersion = old.ResourceVersion
			_, err = ic.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(ic.cfg.Namespace).Update(data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// InstallStatus install status
func (ic *InstallUseCaseImpl) InstallStatus() ([]*model.InstallStatus, error) {
	clusterInfo, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(ic.cfg.Namespace).Get(ic.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// status := "" // TODO fanyangyang if install process is downloading rainbond, what status it is?
	if clusterInfo != nil {
		fmt.Println(clusterInfo.Status.Phase)
		fmt.Println(clusterInfo.Status.Conditions)
		// status = string(configs.Status.Phase)
	} else {
		logrus.Warn("cluster config has not be created yet, something occured ? ")
	}
	return nil, nil
}

func parseInstallStatus(source *v1alpha1.RainbondClusterStatus) []*model.InstallStatus {

	return nil
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
