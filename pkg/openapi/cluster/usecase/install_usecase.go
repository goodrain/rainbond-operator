package usecase

import (
	"fmt"
	"os"

	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/customerror"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"

	"github.com/sirupsen/logrus"

	"github.com/GLYASAI/rainbond-operator/pkg/util/downloadutil"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	version                    = "V5.2-dev"
	defaultRainbondDownloadURL = "192.168.2.222" // TODO fanyangyang download url
	defaultRainbondFilePath    = "/opt/rainbond/rainbond.tar"

	componentClaims []componentClaim
)

// TODO fanyangyang use logrus

type componentClaim struct {
	namespace string
	name      string
	version   string
	image     string
}

func (c *componentClaim) getImage() string {
	return fmt.Sprintf("%s/%s:%s", c.namespace, c.name, c.version)
}

func init() {
	componentClaims = []componentClaim{
		{name: "rbd-api", image: "goodrain.me/rbd-api:V5.2-dev"},
		{name: "rbd-app-ui", image: "goodrain.me/rbd-app-ui:V5.2-dev"},
		{name: "rbd-chaos", image: "goodrain.me/rbd-chaos:V5.2-dev"},
		{name: "rbd-db", image: "goodrain.me/mariadb"},
		{name: "rbd-dns", image: "goodrain.me/rbd-dns:5.1.0"},
		{name: "rbd-etcd", image: "quay.io/coreos/etcd:latest"},
		{name: "rbd-eventlog", image: "goodrain.me/rbd-eventlog:V5.2-dev"},
		{name: "rbd-gateway", image: "abewang/rbd-gateway:V5.2-dev"},
		{name: "rbd-hub", image: "rainbond/rbd-registry:2.6.2"},
		{name: "rbd-monitor", image: "goodrain.me/rbd-monitor:V5.2-dev"},
		{name: "rbd-mq", image: "goodrain.me/rbd-mq:V5.2-dev"},
		{name: "rbd-nfs-provisioner", image: "abewang/nfs-provisioner:v2.2.1-k8s1.12"},
		{name: "rbd-node", image: "abewang/rbd-node:V5.2-dev"},
		{name: "rbd-worker", image: "goodrain.me/rbd-worker:V5.2-dev"},
		{name: "rbd-webcli", image: "goodrain.me/rbd-webcli:V5.2-dev"}, // not now
		{name: "rbd-repo", image: "goodrain.me/rbd-repo:6.16.0"},
	}
}

func parseComponentClaim(claim componentClaim) *v1alpha1.RbdComponent {
	component := &v1alpha1.RbdComponent{}
	component.Namespace = claim.namespace
	component.Name = claim.name
	component.Spec.Version = claim.version
	component.Spec.Image = claim.image
	component.Spec.LogLevel = "debug"
	component.Spec.Type = claim.name
	return component
}

// InstallUseCaseImpl install case
type InstallUseCaseImpl struct {
	cfg              *option.Config
	downloadListener *downloadutil.DownloadWithProgress
}

// NewInstallUseCase new install case
func NewInstallUseCase(cfg *option.Config) *InstallUseCaseImpl {
	return &InstallUseCaseImpl{cfg: cfg}
}

// Install install
func (ic *InstallUseCaseImpl) Install() error {
	if err := ic.BeforeInstall(); err != nil {
		return err
	}

	// step 3 create custom resource
	return ic.createComponents(componentClaims...) // TODO fanyangyang do not install for test download
}

func (ic *InstallUseCaseImpl) canInstallOrNot() error {
	if ic.downloadListener != nil {
		return customerror.NewDownloadingError("install process is processon, please hold on")
	}

	if _, err := os.Stat(ic.cfg.ArchiveFilePath); os.IsNotExist(err) {
		logrus.Info("rainbond archive file does not exists, downloading background ...")

		// step 2 download archive
		if err := ic.downloadFile(); err != nil {
			logrus.Errorf("download rainbond file error: %s", err.Error())
			return customerror.NewDownLoadError("download rainbond.tar error, please try again or upload it using /uploads")
		}

	}
	return nil
}

func (ic *InstallUseCaseImpl) initRainbondPackage() error {
	// rainbondpackage
	pkg := &v1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondpackage",
			Namespace: ic.cfg.Namespace,
		},
		Spec: v1alpha1.RainbondPackageSpec{PkgPath: ic.cfg.ArchiveFilePath},
	}
	_, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Get("rainbondpackage", metav1.GetOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get rainbondpackage: %v", err)
		}
		log.Info("no rainbondpackage found, create a new one.")
		_, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Create(pkg)
		if err != nil {
			return fmt.Errorf("failed to create rainbondpackage: %v", err)
		}
	}
	return nil
}

func (ic *InstallUseCaseImpl) initKubeCfg() error {
	kubeCfg := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ic.cfg.KubeCfgSecretName,
			Namespace: ic.cfg.Namespace,
		},
		Data: map[string][]byte{
			"ca":   ic.cfg.RestConfig.CAData,
			"cert": ic.cfg.RestConfig.CertData,
			"key":  ic.cfg.RestConfig.KeyData,
		},
	}
	_, err := ic.cfg.KubeClient.CoreV1().Secrets(ic.cfg.Namespace).Get(kubeCfg.Name, metav1.GetOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get kubecfg secret: %v", err)
		}
		log.Info("no kubecfg secret found, create a new one.")
		_, err := ic.cfg.KubeClient.CoreV1().Secrets(ic.cfg.Namespace).Create(&kubeCfg)
		if err != nil {
			return fmt.Errorf("failed to create kubecfg secret : %v", err)
		}
	}
	return nil
}

func (ic *InstallUseCaseImpl) initResourceDep() error {
	if err := ic.initRainbondPackage(); err != nil {
		return err
	}
	if err := ic.initKubeCfg(); err != nil {
		return err
	}

	return nil
}

// BeforeInstall before install check
func (ic *InstallUseCaseImpl) BeforeInstall() error {
	// step 1 check if archive is exists or not
	if err := ic.canInstallOrNot(); err != nil {
		return err
	}
	logger := log.WithValues("install")
	logger.Info("rainbond archive file already exists")

	if err := ic.initResourceDep(); err != nil {
		return err
	}
	return nil
}

func (ic *InstallUseCaseImpl) createComponents(components ...componentClaim) error {
	for _, rbdComponent := range components {
		// component := &componentClaim{name: rbdComponent, version: version, namespace: ic.cfg.Namespace}
		data := parseComponentClaim(rbdComponent)
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
func (ic *InstallUseCaseImpl) InstallStatus() ([]model.InstallStatus, error) {
	statuses := make([]model.InstallStatus, 0)
	clusterInfo, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(ic.cfg.Namespace).Get(ic.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if clusterInfo != nil {
		statuses = ic.parseInstallStatus(clusterInfo.Status)
	} else {
		logrus.Warn("cluster config has not be created yet, something occured ? ")
	}
	return statuses, nil
}

func (ic *InstallUseCaseImpl) parseInstallStatus(source *v1alpha1.RainbondClusterStatus) (statuses []model.InstallStatus) {
	if source == nil {
		return
	}
	statuses = append(statuses, ic.stepSetting())
	statuses = append(statuses, ic.stepDownload())
	statuses = append(statuses, ic.stepPrepareStorage(source))
	statuses = append(statuses, ic.stepPrepareImageHub(source))
	statuses = append(statuses, ic.stepUnpack(source))
	statuses = append(statuses, ic.stepLoadImage(source))
	statuses = append(statuses, ic.stepPushImage(source))
	statuses = append(statuses, ic.stepCreateComponent(source))
	return
}

// step 1 setting cluster
func (ic *InstallUseCaseImpl) stepSetting() model.InstallStatus {
	return model.InstallStatus{
		StepName: "step_setting",
		Status:   "status_finished", // TODO fanyangyang waiting
		Progress: 100,
		Message:  "",
	}
}

// step 2 download rainbond
func (ic *InstallUseCaseImpl) stepDownload() model.InstallStatus {
	installStatus := model.InstallStatus{StepName: "step_download"}
	if _, err := os.Stat(ic.cfg.ArchiveFilePath); os.IsNotExist(err) {
		// file not found
		installStatus.Status = "status_waiting"
		return installStatus
	}
	if ic.downloadListener != nil && !ic.downloadListener.Finished {
		installStatus.Progress = ic.downloadListener.Percent
		installStatus.Status = "status_processing"
		return installStatus
	}
	installStatus.Status = "status_finished"
	installStatus.Progress = 100
	return installStatus
}

func (ic *InstallUseCaseImpl) stepPrepare(stepName string, conditionType v1alpha1.RainbondClusterConditionType, source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	var status model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting:
		status = model.InstallStatus{
			StepName: stepName,
			Status:   "status_waiting", // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterPreparing:
		status = model.InstallStatus{
			StepName: stepName,
			Message:  "",
		}
		found := false
		for _, condition := range source.Conditions {
			if condition.Type == conditionType && !found {
				if condition.Status == v1alpha1.ConditionTrue {
					status.Progress = 100
					status.Status = "status_finished"
				} else {
					status.Progress = 0
					status.Status = "status_processing"
				}
				found = true
				break
			}
		}
		if !found {
			status.Status = "status_processing"
			status.Progress = 0
		}
	case v1alpha1.RainbondClusterPackageProcessing, v1alpha1.RainbondClusterPending, v1alpha1.RainbondClusterRunning:
		status = model.InstallStatus{
			StepName: stepName,
			Status:   "status_finished",
			Progress: 100,
			Message:  "",
		}
	default:
		status = model.InstallStatus{
			StepName: stepName,
			Status:   "status_waiting", // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	}
	return status
}

// step 3 prepare storage
func (ic *InstallUseCaseImpl) stepPrepareStorage(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	return ic.stepPrepare("step_prepare_storage", v1alpha1.StorageReady, source)
}

// step 4 prepare image hub
func (ic *InstallUseCaseImpl) stepPrepareImageHub(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	return ic.stepPrepare("step_prepare_image_hub", v1alpha1.ImageRepositoryInstalled, source)
}

func (ic *InstallUseCaseImpl) stepPackProcess(stepName string, conditionType v1alpha1.RainbondClusterConditionType, source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	var status model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting, v1alpha1.RainbondClusterPreparing:
		status = model.InstallStatus{
			StepName: stepName,
			Status:   "status_waiting", // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterPackageProcessing:
		status = model.InstallStatus{
			StepName: stepName,
			Message:  "",
		}
		found := false
		for _, condition := range source.Conditions {
			if condition.Type == conditionType && !found {
				if condition.Status == v1alpha1.ConditionTrue {
					status.Progress = 100
					status.Status = "status_finished"
				} else {
					status.Progress = 0
					status.Status = "status_processing"
				}

				found = true
				break
			}
		}
		if !found {
			status.Status = "status_processing"
			status.Progress = 0
		}
	case v1alpha1.RainbondClusterPending, v1alpha1.RainbondClusterRunning:
		status = model.InstallStatus{
			StepName: stepName,
			Status:   "status_finished",
			Progress: 100,
			Message:  "",
		}
	default:
		status = model.InstallStatus{
			StepName: stepName,
			Status:   "status_waiting", // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	}
	return status
}

// step 5 unpack rainbond
func (ic *InstallUseCaseImpl) stepUnpack(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	return ic.stepPackProcess("step_unpacke", v1alpha1.PackageExtracted, source)
}

// step 6 load image
func (ic *InstallUseCaseImpl) stepLoadImage(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	return ic.stepPackProcess("step_load_image", v1alpha1.ImagesLoaded, source)
}

// step 7 push image
func (ic *InstallUseCaseImpl) stepPushImage(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	return ic.stepPackProcess("step_push_image", v1alpha1.ImagesPushed, source)
}

// step 8 create component
func (ic *InstallUseCaseImpl) stepCreateComponent(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	var status model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting, v1alpha1.RainbondClusterPreparing, v1alpha1.RainbondClusterPackageProcessing:
		status = model.InstallStatus{
			StepName: "step_install_component",
			Status:   "status_waiting", // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterPending:
		status = model.InstallStatus{
			StepName: "step_install_component",
			Status:   "status_processing",
			Progress: 0,
			Message:  "",
		}
	case v1alpha1.RainbondClusterRunning:
		status = model.InstallStatus{
			StepName: "step_install_component",
			Status:   "status_finished",
			Progress: 100,
			Message:  "",
		}
	default:
		status = model.InstallStatus{
			StepName: "step_install_component",
			Status:   "status_waiting", // TODO fanyangyang waiting
			Progress: 0,
			Message:  "",
		}
	}
	return status
}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (ic *InstallUseCaseImpl) downloadFile() error {
	ic.downloadListener = &downloadutil.DownloadWithProgress{URL: ic.cfg.DownloadURL, SavedPath: ic.cfg.ArchiveFilePath}
	defer func() {
		ic.downloadListener = nil
	}()
	return ic.downloadListener.Download()

}
