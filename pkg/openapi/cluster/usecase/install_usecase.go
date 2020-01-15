package usecase

import (
	"fmt"
	"os"
	"time"

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

var (

	// StepSetting          StepSetting
	StepSetting = "step_setting"
	// StepDownload         StepDownload
	StepDownload = "step_download"
	// StepPrepareInfrastructure  StepPrepareInfrastructure
	StepPrepareInfrastructure = "step_prepare_infrastructure"
	// StepUnpack           StepUnpack
	StepUnpack = "step_unpacke"
	// StepHandleImage      StepHandleImage
	StepHandleImage = "step_handle_image"
	// StepInstallComponent StepInstallComponent
	StepInstallComponent = "step_install_component"
)

var (
	// InstallStatusWaiting    InstallStatus_Waiting
	InstallStatusWaiting = "status_waiting"
	// InstallStatusProcessing InstallStatus_Processing
	InstallStatusProcessing = "status_processing"
	// InstallStatusFinished   InstallStatus_Finished
	InstallStatusFinished = "status_finished"
	// InstallStatusFailed     InstallStatus_Failed
	InstallStatusFailed = "status_failed"
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
		{name: "rbd-nfs", image: "abewang/nfs-provisioner:v2.2.1-k8s1.12"},
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
	downloadError    error
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
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if ic.downloadListener == nil {
				logrus.Info("download progress finished, create resource")
				ic.createComponents(componentClaims...)
				break
			}
			logrus.Debug("waiting for download progress finish")
		}
	}()

	return nil
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
	logrus.Debug("create rainbondpackage resource start")
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
	logrus.Debug("create rainbondpackage resource finish")
	return nil
}

func (ic *InstallUseCaseImpl) initKubeCfg() error {
	logrus.Debug("create kubernetes secret resource start")
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
	logrus.Debug("create kubernetes secret resource finish")
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
	statuses = append(statuses, ic.stepPrepareInfrastructure(source))
	statuses = append(statuses, ic.stepUnpack(source))
	statuses = append(statuses, ic.stepHandleImage(source))
	statuses = append(statuses, ic.stepCreateComponent(source))
	return
}

// step 1 setting cluster
func (ic *InstallUseCaseImpl) stepSetting() model.InstallStatus {
	return model.InstallStatus{
		StepName: StepSetting,
		Status:   InstallStatusFinished,
		Progress: 100,
	}
}

// step 2 download rainbond
func (ic *InstallUseCaseImpl) stepDownload() model.InstallStatus {
	installStatus := model.InstallStatus{StepName: StepDownload}
	if ic.downloadListener != nil && !ic.downloadListener.Finished {
		installStatus.Progress = ic.downloadListener.Percent
		installStatus.Status = InstallStatusProcessing
		return installStatus
	}

	if _, err := os.Stat(ic.cfg.ArchiveFilePath); os.IsNotExist(err) {
		if ic.downloadError != nil { // download error
			installStatus.Status = InstallStatusFailed
			installStatus.Message = "download rainbond.tar error, please try again or upload it using /uploads"
			return installStatus
		}
		// file not found
		installStatus.Status = InstallStatusWaiting
		return installStatus
	}

	installStatus.Status = InstallStatusFinished
	installStatus.Progress = 100
	return installStatus
}

// step 3 prepare storage and image hub
func (ic *InstallUseCaseImpl) stepPrepareInfrastructure(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	var status model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting:
		status = model.InstallStatus{
			StepName: StepPrepareInfrastructure,
			Status:   InstallStatusWaiting,
		}
	case v1alpha1.RainbondClusterPreparing:
		status = model.InstallStatus{
			StepName: StepPrepareInfrastructure,
			Status:   InstallStatusProcessing,
			Message:  source.Message,
			Reason:   source.Reason,
		}
		for _, condition := range source.Conditions {
			if condition.Type == v1alpha1.ImageRepositoryInstalled || condition.Type == v1alpha1.StorageReady {
				if condition.Status == v1alpha1.ConditionTrue {
					status.Progress += 50
				}
			}
		}
	case v1alpha1.RainbondClusterPackageProcessing, v1alpha1.RainbondClusterPending, v1alpha1.RainbondClusterRunning:
		status = model.InstallStatus{
			StepName: StepPrepareInfrastructure,
			Status:   InstallStatusFinished,
			Progress: 100,
		}
	default:
		status = model.InstallStatus{
			StepName: StepPrepareInfrastructure,
			Status:   InstallStatusWaiting,
		}
	}
	return status
}

// step 4 unpack rainbond
func (ic *InstallUseCaseImpl) stepUnpack(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	status := model.InstallStatus{
		StepName: StepUnpack,
	}
	rbdpkgStatus := ic.getRainbondPackageStatus()
	if rbdpkgStatus != nil {
		switch rbdpkgStatus.Phase {
		case v1alpha1.RainbondPackageFailed:
			status.Status = InstallStatusFailed
			status.Reason = rbdpkgStatus.Reason
			status.Message = rbdpkgStatus.Message
		case v1alpha1.RainbondPackageWaiting:
			status.Status = InstallStatusWaiting
		case v1alpha1.RainbondPackageExtracting:
			status.Status = InstallStatusProcessing
			if rbdpkgStatus.FilesNumber != 0 {
				status.Progress = 100 * int(rbdpkgStatus.NumberExtracted/rbdpkgStatus.FilesNumber)
			}
		case v1alpha1.RainbondPackagePushing, v1alpha1.RainbondPackageCompleted:
			status.Status = InstallStatusFinished
			status.Progress = 100
		}
	}
	return status
}

func (ic *InstallUseCaseImpl) getRainbondPackageStatus() *v1alpha1.RainbondPackageStatus {
	rbdpkg, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Get(ic.cfg.Rainbondpackage, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("get rainbondpackage error: %s", err.Error())
		return nil
	}
	return rbdpkg.Status
}

// step 5 handle image, load and push image to image hub
func (ic *InstallUseCaseImpl) stepHandleImage(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	status := model.InstallStatus{
		StepName: StepHandleImage,
	}
	rbdpkgStatus := ic.getRainbondPackageStatus()
	if rbdpkgStatus != nil {
		switch rbdpkgStatus.Phase {
		case v1alpha1.RainbondPackageFailed:
			status.Status = InstallStatusFailed
			status.Reason = rbdpkgStatus.Reason
			status.Message = rbdpkgStatus.Message
		case v1alpha1.RainbondPackageWaiting, v1alpha1.RainbondPackageExtracting:
			status.Status = InstallStatusWaiting
		case v1alpha1.RainbondPackagePushing:
			status.Status = InstallStatusProcessing
			if rbdpkgStatus.ImagesNumber != 0 {
				status.Progress = 100 * int(rbdpkgStatus.NumberExtracted/rbdpkgStatus.ImagesNumber)
			}
		case v1alpha1.RainbondPackageCompleted:
			status.Status = InstallStatusFinished
			status.Progress = 100
		}
	}
	return status
}

// step 6 create component
func (ic *InstallUseCaseImpl) stepCreateComponent(source *v1alpha1.RainbondClusterStatus) model.InstallStatus {
	var status model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting, v1alpha1.RainbondClusterPreparing, v1alpha1.RainbondClusterPackageProcessing:
		status = model.InstallStatus{
			StepName: StepInstallComponent,
			Status:   InstallStatusWaiting,
		}
	case v1alpha1.RainbondClusterPending:

	case v1alpha1.RainbondClusterRunning:
		status = model.InstallStatus{
			StepName: StepInstallComponent,
			Status:   InstallStatusFinished,
			Progress: 100,
		}
	default:
		status = model.InstallStatus{
			StepName: StepInstallComponent,
			Status:   InstallStatusWaiting,
			Progress: 0,
		}
	}
	status = ic.parsePendingComponentStatus()
	return status
}

func (ic *InstallUseCaseImpl) parsePendingComponentStatus() model.InstallStatus {
	status := model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusProcessing}
	componentUsecase := NewComponentUsecase(ic.cfg)
	componentStatuses, err := componentUsecase.List()
	if err != nil {
		return model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusFailed, Message: err.Error()}
	}
	total := len(componentStatuses)
	if total == 0 {
		return model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusWaiting}
	}
	finished := 0
	for _, status := range componentStatuses {
		if status.Replicas == status.ReadyReplicas {
			finished++
		}
	}
	status.Progress = 100 * finished / total
	return status
}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (ic *InstallUseCaseImpl) downloadFile() error {
	ic.downloadListener = &downloadutil.DownloadWithProgress{URL: ic.cfg.DownloadURL, SavedPath: ic.cfg.ArchiveFilePath}
	go func() {
		if err := ic.downloadListener.Download(); err != nil {
			ic.downloadError = err
		}
		ic.downloadListener = nil // download process finish, delete it
	}()
	return nil

}
