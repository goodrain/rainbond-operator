package usecase

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/customerror"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/downloadutil"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
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

var rbdVersion = "V5.2-dev"

func init() {
	componentClaims = []componentClaim{
		{name: "rbd-etcd", image: "registry.cn-hangzhou.aliyuncs.com/abewang/etcd:v3.3.18"}, // TODO: custom domain
		{name: "rbd-gateway", image: "registry.cn-hangzhou.aliyuncs.com/abewang/rbd-gateway:" + rbdVersion},
		{name: "rbd-hub", image: "registry.cn-hangzhou.aliyuncs.com/abewang/registry:2.6.2"},
		{name: "rbd-node", image: "registry.cn-hangzhou.aliyuncs.com/abewang/rbd-node:" + rbdVersion},
		{name: "rbd-nfs", image: "registry.cn-hangzhou.aliyuncs.com/abewang/nfs-provisioner:v2.2.1-k8s1.12"},
		{name: "rbd-api", image: "goodrain.me/rbd-api:" + rbdVersion},
		{name: "rbd-app-ui", image: "goodrain.me/rbd-app-ui:" + rbdVersion},
		{name: "rbd-chaos", image: "goodrain.me/rbd-chaos:" + rbdVersion},
		{name: "rbd-db", image: "goodrain.me/rbd-db:v5.1.9"},
		{name: "rbd-dns", image: "goodrain.me/rbd-dns"},
		{name: "rbd-eventlog", image: "goodrain.me/rbd-eventlog:" + rbdVersion},
		{name: "rbd-monitor", image: "goodrain.me/rbd-monitor:" + rbdVersion},
		{name: "rbd-mq", image: "goodrain.me/rbd-mq:" + rbdVersion},
		{name: "rbd-worker", image: "goodrain.me/rbd-worker:" + rbdVersion},
		{name: "rbd-webcli", image: "goodrain.me/rbd-webcli:" + rbdVersion},
		{name: "rbd-grctl", image: "goodrain.me/rbd-grctl:" + rbdVersion},
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

const (
	md5CheckStatusWait int32 = iota
	md5CheckStatusProcess
	md5CheckStatusPass
	md5CheckStatusFailed
)

type md5Check struct {
	checked int32
}

// InstallUseCaseImpl install case
type InstallUseCaseImpl struct {
	cfg              *option.Config
	componentUsecase ComponentUseCase
	downloadListener *downloadutil.DownloadWithProgress
	downloadError    error
	md5checkInfo     md5Check
	downloaded       bool //only when precheck check file exists set as true
}

// NewInstallUseCase new install case
func NewInstallUseCase(cfg *option.Config, componentUsecase ComponentUseCase) *InstallUseCaseImpl {
	return &InstallUseCaseImpl{cfg: cfg, componentUsecase: componentUsecase}
}

// InstallPreCheck pre check
func (ic *InstallUseCaseImpl) InstallPreCheck() (model.StatusRes, error) {
	defer commonutil.TimeConsume(time.Now())
	statusres := model.StatusRes{}
	statuses := make([]model.InstallStatus, 0)
	statuses = append(statuses, ic.stepSetting())
	downStatus := model.InstallStatus{StepName: StepDownload}
	// step 1 check if archive is exists or not
	if err := ic.canInstallOrNot(StepDownload); err != nil {
		if _, ok := err.(*customerror.DownloadingError); ok {
			downStatus.Status = InstallStatusProcessing
			downStatus.Progress = ic.downloadListener.Percent
		} else {
			downStatus.Status = InstallStatusFailed
			downStatus.Message = err.Error()
		}
	} else {
		if atomic.LoadInt32(&ic.md5checkInfo.checked) == md5CheckStatusPass {
			downStatus.Progress = 100
			downStatus.Status = InstallStatusFinished
			ic.downloaded = true
		} else {
			downStatus.Status = InstallStatusWaiting
		}
	}
	statuses = append(statuses, downStatus)

	finalStatus := InstallStatusFinished
	for _, status := range statuses {
		if status.Status != InstallStatusFinished {
			finalStatus = InstallStatusProcessing
			break
		}
	}
	statuses = append(statuses, model.InstallStatus{StepName: StepPrepareInfrastructure, Status: InstallStatusWaiting})
	statuses = append(statuses, model.InstallStatus{StepName: StepUnpack, Status: InstallStatusWaiting})
	statuses = append(statuses, model.InstallStatus{StepName: StepHandleImage, Status: InstallStatusWaiting})
	statuses = append(statuses, model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusWaiting})

	statusres.StatusList = statuses
	statusres.FinalStatus = finalStatus
	return statusres, nil
}

// Install install
func (ic *InstallUseCaseImpl) Install() error {
	defer commonutil.TimeConsume(time.Now())
	if err := ic.BeforeInstall(); err != nil {
		return err
	}
	return ic.createComponents(componentClaims...)
}

func (ic *InstallUseCaseImpl) canInstallOrNot(step string) error {
	defer commonutil.TimeConsume(time.Now())
	if ic.downloadListener != nil {
		return customerror.NewDownloadingError("install process is processon, please hold on")
	}

	if _, err := os.Stat(ic.cfg.ArchiveFilePath); os.IsNotExist(err) {
		logrus.Info("rainbond archive file does not exists, downloading background ...")
		if step == StepDownload {
			// the latest download progress failed
			if ic.downloadError != nil {
				ic.downloadError = nil
				return customerror.NewDownLoadError("download rainbond.tar error, please try again or upload it using /uploads")
			}
			// step 2 download archive
			if err := ic.downloadFile(); err != nil {
				logrus.Errorf("download rainbond file error: %s", err.Error())
				return customerror.NewDownLoadError("download rainbond.tar error, please try again or upload it using /uploads")
			}
			return customerror.NewDownloadingError("install process is processon, please hold on")
		}
		logrus.Error("rainbond tar do not exists")
		return customerror.NewRainbondTarNotExistError("rainbond tar do not exists")

	}
	if step == StepDownload {
		if atomic.LoadInt32(&ic.md5checkInfo.checked) == md5CheckStatusWait {
			// check md5
			go func() {
				atomic.StoreInt32(&ic.md5checkInfo.checked, md5CheckStatusProcess)

				logrus.Info("check rainbond.tar's md5")
				dp := downloadutil.DownloadWithProgress{Wanted: ic.cfg.DownloadMD5}
				target, err := os.Open(ic.cfg.ArchiveFilePath)
				if err != nil {
				}
				if err := dp.CheckMD5(target); err != nil {
					logrus.Warn("download tar md5 check error, waiting new download progress")
					atomic.StoreInt32(&ic.md5checkInfo.checked, md5CheckStatusFailed)
					return
				}
				atomic.StoreInt32(&ic.md5checkInfo.checked, md5CheckStatusPass)
			}()
		}
		if atomic.LoadInt32(&ic.md5checkInfo.checked) == md5CheckStatusFailed {
			logrus.Warn("exists rainbond.tar md5 check failed, re-download it")
			// step 2 download archive (it will reset md5check info)
			if err := ic.downloadFile(); err != nil {
				logrus.Errorf("download rainbond file error: %s", err.Error())
				return customerror.NewDownLoadError("download rainbond.tar error, please try again or upload it using /uploads")
			}
			return customerror.NewDownLoadError("download rainbond.tar error, please try again or upload it using /uploads")
		}
	}

	return nil
}

func (ic *InstallUseCaseImpl) initRainbondPackage() error {
	defer commonutil.TimeConsume(time.Now())
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
	defer commonutil.TimeConsume(time.Now())
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
	defer commonutil.TimeConsume(time.Now())
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
	defer commonutil.TimeConsume(time.Now())
	// step 1 check if archive is exists or not
	if err := ic.canInstallOrNot(""); err != nil {
		return err
	}

	if err := ic.initResourceDep(); err != nil {
		return err
	}
	return nil
}

func (ic *InstallUseCaseImpl) createComponents(components ...componentClaim) error {
	defer commonutil.TimeConsume(time.Now())
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
func (ic *InstallUseCaseImpl) InstallStatus() (model.StatusRes, error) {
	defer commonutil.TimeConsume(time.Now())
	statusres := model.StatusRes{}
	clusterInfo, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(ic.cfg.Namespace).Get(ic.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		return model.StatusRes{}, err
	}
	if clusterInfo != nil && clusterInfo.Status != nil {
		statusres = ic.parseInstallStatus(clusterInfo.Status)
	} else {
		logrus.Warn("cluster config has not be created yet, what happened ? ")
		statusres = model.StatusRes{
			FinalStatus: InstallStatusWaiting,
			StatusList:  nil,
		}
	}
	return statusres, nil
}

func (ic *InstallUseCaseImpl) parseInstallStatus(source *v1alpha1.RainbondClusterStatus) (statusres model.StatusRes) {
	defer commonutil.TimeConsume(time.Now())
	if source == nil {
		return
	}
	statuses := make([]model.InstallStatus, 0)

	statuses = append(statuses, ic.stepSetting())

	downloadStatus := ic.stepDownload()
	statuses = append(statuses, downloadStatus)

	infrastructureStatus := ic.stepPrepareInfrastructure(source, downloadStatus.Status != InstallStatusFinished)
	statuses = append(statuses, infrastructureStatus)

	unpackStatus := ic.stepUnpack(source, infrastructureStatus.Status != InstallStatusFinished)
	statuses = append(statuses, unpackStatus)

	handleImageStatus := ic.stepHandleImage(source, unpackStatus.Status != InstallStatusFinished)
	statuses = append(statuses, handleImageStatus)

	createComponentStatus := ic.stepCreateComponent(source, handleImageStatus.Status != InstallStatusFinished)
	statuses = append(statuses, createComponentStatus)

	finalStatus := InstallStatusFinished
	if downloadStatus.Status == InstallStatusWaiting {
		finalStatus = InstallStatusWaiting // have not installed
	} else {
		for _, status := range statuses {
			if status.Status != InstallStatusFinished {
				finalStatus = InstallStatusProcessing
				break
			}
		}
	}

	statusres = model.StatusRes{
		FinalStatus: finalStatus,
		StatusList:  statuses,
	}
	return
}

// step 1 setting cluster
func (ic *InstallUseCaseImpl) stepSetting() model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	return model.InstallStatus{
		StepName: StepSetting,
		Status:   InstallStatusFinished,
		Progress: 100,
	}
}

// step 2 download rainbond
func (ic *InstallUseCaseImpl) stepDownload() model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if ic.downloadListener != nil {
		// downloading
		return model.InstallStatus{
			StepName: StepDownload,
			Status:   InstallStatusProcessing,
			Progress: ic.downloadListener.Percent,
		}
	}

	if !ic.downloaded {
		// have not installed
		return model.InstallStatus{
			StepName: StepDownload,
			Status:   InstallStatusWaiting,
		}
	}

	installStatus := model.InstallStatus{
		StepName: StepDownload,
		Status:   InstallStatusFinished,
		Progress: 100,
	}
	return installStatus
}

// step 3 prepare storage and image hub
func (ic *InstallUseCaseImpl) stepPrepareInfrastructure(source *v1alpha1.RainbondClusterStatus, waiting bool) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if waiting {
		return model.InstallStatus{
			StepName: StepPrepareInfrastructure,
			Status:   InstallStatusWaiting,
		}
	}
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
func (ic *InstallUseCaseImpl) stepUnpack(source *v1alpha1.RainbondClusterStatus, waiting bool) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if waiting {
		return model.InstallStatus{
			StepName: StepUnpack,
			Status:   InstallStatusWaiting,
		}
	}
	status := model.InstallStatus{
		StepName: StepUnpack,
	}
	rbdpkgStatus := ic.getRainbondPackageStatus()
	if rbdpkgStatus != nil {
		status.Message = rbdpkgStatus.Message
		status.Reason = rbdpkgStatus.Reason
		switch rbdpkgStatus.Phase {
		case v1alpha1.RainbondPackageFailed:
			status.Status = InstallStatusFailed
		case v1alpha1.RainbondPackageWaiting:
			status.Status = InstallStatusWaiting
		case v1alpha1.RainbondPackageExtracting:
			status.Status = InstallStatusProcessing
			if rbdpkgStatus.FilesNumber != 0 {
				status.Progress = int(100 * rbdpkgStatus.NumberExtracted / rbdpkgStatus.FilesNumber)
			}
		case v1alpha1.RainbondPackagePushing, v1alpha1.RainbondPackageCompleted:
			status.Status = InstallStatusFinished
			status.Progress = 100
		}
	}
	return status
}

func (ic *InstallUseCaseImpl) getRainbondPackageStatus() *v1alpha1.RainbondPackageStatus {
	defer commonutil.TimeConsume(time.Now())
	rbdpkg, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Get(ic.cfg.Rainbondpackage, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("get rainbondpackage error: %s", err.Error())
		return nil
	}
	return rbdpkg.Status
}

// step 5 handle image, load and push image to image hub
func (ic *InstallUseCaseImpl) stepHandleImage(source *v1alpha1.RainbondClusterStatus, waiting bool) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if waiting {
		return model.InstallStatus{
			StepName: StepHandleImage,
			Status:   InstallStatusWaiting,
		}
	}
	status := model.InstallStatus{
		StepName: StepHandleImage,
	}
	rbdpkgStatus := ic.getRainbondPackageStatus()
	if rbdpkgStatus != nil {
		status.Message = rbdpkgStatus.Message
		status.Reason = rbdpkgStatus.Reason
		switch rbdpkgStatus.Phase {
		case v1alpha1.RainbondPackageFailed:
			status.Status = InstallStatusFailed
		case v1alpha1.RainbondPackageWaiting, v1alpha1.RainbondPackageExtracting:
			status.Status = InstallStatusWaiting
		case v1alpha1.RainbondPackagePushing:
			status.Status = InstallStatusProcessing
			if rbdpkgStatus.ImagesNumber != 0 {
				pushed := len(rbdpkgStatus.ImagesPushed)
				status.Progress = int(100 * int32(pushed) / rbdpkgStatus.ImagesNumber)
			}
		case v1alpha1.RainbondPackageCompleted:
			status.Status = InstallStatusFinished
			status.Progress = 100
		}
	}
	return status
}

// step 6 create component
func (ic *InstallUseCaseImpl) stepCreateComponent(source *v1alpha1.RainbondClusterStatus, waiting bool) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if waiting {
		return model.InstallStatus{
			StepName: StepInstallComponent,
			Status:   InstallStatusWaiting,
		}
	}
	var status model.InstallStatus
	switch source.Phase {
	case v1alpha1.RainbondClusterWaiting, v1alpha1.RainbondClusterPreparing, v1alpha1.RainbondClusterPackageProcessing:
		status = model.InstallStatus{
			StepName: StepInstallComponent,
			Status:   InstallStatusWaiting,
		}
	case v1alpha1.RainbondClusterPending:
		status = model.InstallStatus{
			StepName: StepInstallComponent,
			Status:   InstallStatusProcessing,
		}

		componentStatuses, err := ic.componentUsecase.List()
		if err != nil {
			return model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusFailed, Message: err.Error()}
		}
		total := len(componentStatuses)
		if total == 0 {
			return model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusWaiting}
		}
		finished := 0 // running components size
		for _, status := range componentStatuses {
			if status.Replicas == status.ReadyReplicas {
				finished++
			}
		}
		// all component size
		allComponent, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(ic.cfg.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return model.InstallStatus{StepName: StepInstallComponent, Status: InstallStatusFailed, Message: err.Error()}
		}
		status.Progress = 100 * finished / len(allComponent.Items)
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

	return status
}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (ic *InstallUseCaseImpl) downloadFile() error {
	defer commonutil.TimeConsume(time.Now())
	ic.downloadListener = &downloadutil.DownloadWithProgress{URL: ic.cfg.DownloadURL, SavedPath: ic.cfg.ArchiveFilePath, Wanted: ic.cfg.DownloadMD5}
	ic.md5checkInfo = md5Check{} // reset md5check info
	logrus.Info("download progress start, reset check as false")
	go func() {
		if err := ic.downloadListener.Download(); err != nil {
			logrus.Error("download rainbondtar error: ", err.Error())
			ic.downloadError = err
		}
		ic.downloadListener = nil // download process finish, delete it
	}()
	return nil

}
