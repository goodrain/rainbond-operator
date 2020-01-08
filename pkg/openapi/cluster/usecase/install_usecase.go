package usecase

import (
	"io"
	"net/http"
	"os"

	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	v1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/translate"

	"github.com/sirupsen/logrus"

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
	statuses := make([]*model.InstallStatus, 0)
	clusterInfo, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(ic.cfg.Namespace).Get(ic.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if clusterInfo != nil {
		statuses = parseInstallStatus(clusterInfo.Status)
	} else {
		logrus.Warn("cluster config has not be created yet, something occured ? ")
	}
	return statuses, nil
}

func parseInstallStatus(source *v1alpha1.RainbondClusterStatus) (statuses []*model.InstallStatus) {
	if source == nil {
		return
	}
	statuses = append(statuses, stepSetting())
	statuses = append(statuses, stepDownload())
	statuses = append(statuses, stepPrepareStorage(source))
	statuses = append(statuses, stepPrepareImageHub(source))
	statuses = append(statuses, stepUnpack(source))
	statuses = append(statuses, stepLoadImage(source))
	statuses = append(statuses, stepPushImage(source))
	statuses = append(statuses, stepCreateComponent(source))
	return
}

// step 1 setting cluster
func stepSetting() *model.InstallStatus {
	return &model.InstallStatus{
		StepName: translate.Translation("step_setting"),
		Status:   translate.Translation("status_finished"), // TODO fanyangyang waiting
		Progress: 100,
		Message:  "",
	}
}

// step 2 download rainbond
func stepDownload() *model.InstallStatus {
	return &model.InstallStatus{
		StepName: translate.Translation("step_download"),
		Status:   translate.Translation("status_finished"), // TODO fanyangyang waiting
		Progress: 100,
		Message:  "",
	}
}

func stepPrepare(stepName string, conditionType v1alpha1.RainbondClusterConditionType, source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	var status *model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting:
		status = &model.InstallStatus{
			StepName: stepName,
			Status:   translate.Translation("status_waiting"), // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterPreparing:
		status = &model.InstallStatus{
			StepName: stepName,
			Message:  "",
		}
		found := false
		for _, condition := range source.Conditions {
			if condition.Type == conditionType && !found {
				status.Status = translate.Translation("status_finished")
				status.Progress = 100
				found = true
				break
			}
		}
		if !found {
			status.Status = translate.Translation("status_processing")
			status.Progress = 0
		}
	case v1alpha1.RainbondClusterPending, v1alpha1.RainbondClusterRunning:
		status = &model.InstallStatus{
			StepName: stepName,
			Status:   translate.Translation("status_finished"),
			Progress: 100,
			Message:  "",
		}
	}
	return status
}

// step 3 prepare storage
func stepPrepareStorage(source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	return stepPrepare(translate.Translation("step_prepare_storage"), v1alpha1.StorageReady, source)
}

// step 4 prepare image hub
func stepPrepareImageHub(source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	return stepPrepare(translate.Translation("step_prepare_image_hub"), v1alpha1.ImageRepositoryInstalled, source)
}

func stepPending(stepName string, conditionType v1alpha1.RainbondClusterConditionType, source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	var status *model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting, v1alpha1.RainbondClusterPreparing:
		status = &model.InstallStatus{
			StepName: stepName,
			Status:   translate.Translation("status_waiting"), // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterPending:
		status = &model.InstallStatus{
			StepName: stepName,
			Message:  "",
		}
		found := false
		for _, condition := range source.Conditions {
			if condition.Type == conditionType && !found {
				status.Status = translate.Translation("status_finished")
				status.Progress = 100
				found = true
				break
			}
		}
		if !found {
			status.Status = translate.Translation("status_processing")
			status.Progress = 0
		}
	case v1alpha1.RainbondClusterRunning:
		status = &model.InstallStatus{
			StepName: stepName,
			Status:   translate.Translation("status_finished"),
			Progress: 100,
			Message:  "",
		}
	}
	return status
}

// step 5 unpack rainbond
func stepUnpack(source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	return stepPending(translate.Translation("step_unpacke"), v1alpha1.PackageExtracted, source)
}

// step 6 load image
func stepLoadImage(source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	return stepPending(translate.Translation("step_load_image"), v1alpha1.ImagesLoaded, source)
}

// step 7 push image
func stepPushImage(source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	return stepPending(translate.Translation("step_push_image"), v1alpha1.ImagesPushed, source)
}

// step 8 create component
func stepCreateComponent(source *v1alpha1.RainbondClusterStatus) *model.InstallStatus {
	var status *model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting, v1alpha1.RainbondClusterPreparing:
		status = &model.InstallStatus{
			StepName: translate.Translation("step_install_component"),
			Status:   translate.Translation("status_waiting"), // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterPending:
		status = &model.InstallStatus{
			StepName: translate.Translation("step_install_component"),
			Status:   translate.Translation("status_processing"),
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterRunning:
		status = &model.InstallStatus{
			StepName: translate.Translation("step_install_component"),
			Status:   translate.Translation("status_finished"),
			Progress: 100,
			Message:  "",
		}
	}
	return status
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
