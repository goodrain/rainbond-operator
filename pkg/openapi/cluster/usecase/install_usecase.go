package usecase

import (
	"fmt"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"strconv"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"

	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	componentClaims []componentClaim
)

var (
	// StepSetting          StepSetting
	StepSetting = "step_setting"
	// StepPrepareHub step prepare hub
	StepPrepareHub = "step_prepare_hub"
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
	logLevel  string
	Configs   map[string]string
	isInit    bool
}

// TODO: custom domain
var existHubDomain = "registry.cn-hangzhou.aliyuncs.com/goodrain"

func parseComponentClaim(claim componentClaim) *v1alpha1.RbdComponent {
	component := &v1alpha1.RbdComponent{}
	component.Namespace = claim.namespace
	component.Name = claim.name
	component.Spec.Version = claim.version
	component.Spec.Image = claim.image
	component.Spec.Configs = claim.Configs
	component.Spec.LogLevel = v1alpha1.ParseLogLevel(claim.logLevel)
	component.Spec.Type = claim.name
	labels := map[string]string{"name": claim.name}
	log.Info(fmt.Sprintf("component %s labels:%+v", component.Name, component.Labels))
	if claim.isInit {
		component.Spec.PriorityComponent = true
		labels["priorityComponent"] = "true"
	}
	component.Labels = labels
	return component
}

// InstallUseCaseImpl install case
type InstallUseCaseImpl struct {
	cfg                *option.Config
	namespace          string
	rainbondKubeClient versioned.Interface

	componentUsecase ComponentUseCase
}

// NewInstallUseCase new install case
func NewInstallUseCase(cfg *option.Config, rainbondKubeClient versioned.Interface, componentUsecase ComponentUseCase) *InstallUseCaseImpl {
	return &InstallUseCaseImpl{
		cfg:                cfg,
		namespace:          cfg.Namespace,
		rainbondKubeClient: rainbondKubeClient,
		componentUsecase:   componentUsecase,
	}
}

// Install install
func (ic *InstallUseCaseImpl) Install(req *v1.ClusterInstallReq) error {
	// create rainbond volume
	if err := ic.createRainbondVolumes(req); err != nil {
		return err
	}
	if err := ic.initRainbondPackage(); err != nil {
		return err
	}
	return ic.createComponents(componentClaims...)
}

func (ic *InstallUseCaseImpl) createRainbondVolumes(req *v1.ClusterInstallReq) error {
	rwx := setRainbondVolume("rainbondvolumerwx", ic.namespace, rbdutil.LabelsForAccessModeRWX(), req.RainbondVolumes.RWX)
	if err := ic.createRainbondVolumeIfNotExists(rwx); err != nil {
		return err
	}
	if req.RainbondVolumes.RWO != nil {
		rwo := setRainbondVolume("rainbondvolumerwo", ic.namespace, rbdutil.LabelsForAccessModeRWX(), req.RainbondVolumes.RWO)
		if err := ic.createRainbondVolumeIfNotExists(rwo); err != nil {
			return err
		}
	}
	return nil
}

func (ic *InstallUseCaseImpl) createRainbondVolumeIfNotExists(volume *v1alpha1.RainbondVolume) error {
	reqLogger := log.WithValues("Namespace", volume.Namespace, "Name", volume.Name)
	_, err := ic.rainbondKubeClient.RainbondV1alpha1().RainbondVolumes(ic.namespace).Create(volume)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			reqLogger.Info("rainbond volume already exists")
			return nil
		}
		reqLogger.Error(err, "create rainbond volume")
		return bcode.ErrCreateRainbondVolume
	}
	return nil
}

func setRainbondVolume(name, namespace string, labels map[string]string, rv *v1.RainbondVolume) *v1alpha1.RainbondVolume {
	spec := v1alpha1.RainbondVolumeSpec{
		StorageClassName: rv.StorageClassName,
	}
	if rv.StorageClassParameters != nil {
		spec.StorageClassParameters = &v1alpha1.StorageClassParameters{
			Provisioner: rv.StorageClassParameters.Provisioner,
			Parameters:  rv.StorageClassParameters.Parameters,
		}
	}

	if rv.CSIPlugin != nil {
		csiplugin := &v1alpha1.CSIPluginSource{}
		switch {
		case rv.CSIPlugin.AliyunCloudDisk != nil:
			csiplugin.AliyunCloudDisk = &v1alpha1.AliyunCloudDiskCSIPluginSource{
				AccessKeyID:      rv.CSIPlugin.AliyunCloudDisk.AccessKeyID,
				AccessKeySecret:  rv.CSIPlugin.AliyunCloudDisk.AccessKeySecret,
				MaxVolumePerNode: strconv.Itoa(rv.CSIPlugin.AliyunCloudDisk.MaxVolumePerNode),
			}
		case rv.CSIPlugin.AliyunNas != nil:
			csiplugin.AliyunNas = &v1alpha1.AliyunNasCSIPluginSource{
				AccessKeyID:     rv.CSIPlugin.AliyunNas.AccessKeyID,
				AccessKeySecret: rv.CSIPlugin.AliyunNas.AccessKeySecret,
			}
		case rv.CSIPlugin.NFS != nil:
			csiplugin.NFS = &v1alpha1.NFSCSIPluginSource{}
		}
		spec.CSIPlugin = csiplugin
	}

	volume := &v1alpha1.RainbondVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    rbdutil.LabelsForRainbond(labels),
		},
		Spec: spec,
	}

	return volume
}

func (ic *InstallUseCaseImpl) initRainbondPackage() error {
	reqLogger := log.WithValues("Namespace", ic.cfg.Namespace, "Name", ic.cfg.Rainbondpackage)
	pkg := &v1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ic.cfg.Rainbondpackage,
			Namespace: ic.cfg.Namespace,
		},
		Spec: v1alpha1.RainbondPackageSpec{PkgPath: ic.cfg.ArchiveFilePath},
	}
	_, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Create(pkg)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			reqLogger.Info("rainbondpackage already exists.")
			return nil
		}
		reqLogger.Error(err, "create rainbond package")
		return bcode.ErrCreateRainbondPackage
	}
	reqLogger.Info("successfully create rainbondpackage")
	return nil
}

func (ic *InstallUseCaseImpl) createComponents(components ...componentClaim) error {
	cluster, err := ic.rainbondKubeClient.RainbondV1alpha1().RainbondClusters(ic.cfg.Namespace).Get("rainbondcluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	componentClaims = []componentClaim{
		{name: "rbd-api", image: "goodrain.me/rbd-api:" + ic.cfg.RainbondVersion},
		{name: "rbd-app-ui", image: "goodrain.me/rbd-app-ui:" + ic.cfg.RainbondVersion},
		{name: "rbd-chaos", image: "goodrain.me/rbd-chaos:" + ic.cfg.RainbondVersion},
		{name: "rbd-dns", image: "goodrain.me/rbd-dns"},
		{name: "rbd-eventlog", image: "goodrain.me/rbd-eventlog:" + ic.cfg.RainbondVersion},
		{name: "rbd-monitor", image: "goodrain.me/rbd-monitor:" + ic.cfg.RainbondVersion},
		{name: "rbd-mq", image: "goodrain.me/rbd-mq:" + ic.cfg.RainbondVersion},
		{name: "rbd-worker", image: "goodrain.me/rbd-worker:" + ic.cfg.RainbondVersion},
		{name: "rbd-webcli", image: "goodrain.me/rbd-webcli:" + ic.cfg.RainbondVersion},
		{name: "rbd-repo", image: "goodrain.me/rbd-repo:6.16.0"},
		{name: "metrics-server", image: "goodrain.me/metrics-server:v0.3.6"},
	}
	if cluster.Spec.RegionDatabase == nil || cluster.Spec.UIDatabase == nil {
		componentClaims = append(componentClaims, componentClaim{name: "rbd-db", image: "goodrain.me/rbd-db:v5.1.9"})
	}

	var isInit bool
	var imageRepository string
	if cluster.Spec.ImageHub == nil {
		isInit = true
		imageRepository = existHubDomain
		componentClaims = append(componentClaims, componentClaim{name: "rbd-hub", image: imageRepository + "/registry:2.6.2", isInit: isInit})
	} else {
		imageRepository = cluster.Spec.ImageHub.Domain
	}
	componentClaims = append(componentClaims, componentClaim{name: "rbd-gateway", image: imageRepository + "/rbd-gateway:" + ic.cfg.RainbondVersion, isInit: isInit})
	componentClaims = append(componentClaims, componentClaim{name: "rbd-node", image: imageRepository + "/rbd-node:" + ic.cfg.RainbondVersion, isInit: isInit})

	if cluster.Spec.EtcdConfig == nil {
		componentClaims = append(componentClaims, componentClaim{name: "rbd-etcd", image: imageRepository + "/etcd:v3.3.18", isInit: isInit})
	}
	if cluster.Spec.StorageClassName == "" {
		componentClaims = append(componentClaims, componentClaim{name: "rbd-nfs", image: imageRepository + "/nfs-provisioner:v2.2.1-k8s1.12", isInit: isInit})
	}

	for _, rbdComponent := range componentClaims {
		reqLogger := log.WithValues("Namespace", rbdComponent.namespace, rbdComponent.name)
		data := parseComponentClaim(rbdComponent)
		// init component
		data.Namespace = ic.cfg.Namespace
		if _, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(ic.cfg.Namespace).Create(data); err != nil {
			if errors.IsAlreadyExists(err) {
				reqLogger.Info("component already exists")
				continue
			}
			reqLogger.Error(err, "create rainbond component")
			return bcode.ErrCreateRbdComponent
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
		log.Error(err, "get rainbond cluster error")
		return model.StatusRes{FinalStatus: InstallStatusWaiting}, nil
	}
	packageInfo, err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Get(ic.cfg.Rainbondpackage, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "get rainbondpackage error")
		return model.StatusRes{FinalStatus: InstallStatusWaiting}, nil
	}
	componentStatuses, err := ic.componentUsecase.List(false)
	if err != nil {
		log.Error(err, "get rbdcomponent error")
		return model.StatusRes{FinalStatus: InstallStatusWaiting}, nil
	}
	statusres = ic.parseInstallStatus(clusterInfo, packageInfo, componentStatuses)
	if clusterInfo != nil {

	} else {
		logrus.Warn("cluster config has not be created yet, something occured ? ")
	}
	return statusres, nil
}

func (ic *InstallUseCaseImpl) parseInstallStatus(clusterInfo *v1alpha1.RainbondCluster, pkgInfo *v1alpha1.RainbondPackage, componentStatues []*v1.RbdComponentStatus) (statusres model.StatusRes) {
	defer commonutil.TimeConsume(time.Now())

	statusres.StatusList = append(statusres.StatusList, ic.stepSetting())
	statusres.StatusList = append(statusres.StatusList, ic.stepHub(clusterInfo, componentStatues))
	statusres.StatusList = append(statusres.StatusList, ic.stepDownload(clusterInfo, pkgInfo))
	statusres.StatusList = append(statusres.StatusList, ic.stepUnpack(clusterInfo, pkgInfo))
	statusres.StatusList = append(statusres.StatusList, ic.stepHandleImage(clusterInfo, pkgInfo))
	statusres.StatusList = append(statusres.StatusList, ic.stepCreateComponent(componentStatues, pkgInfo))

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

func (ic *InstallUseCaseImpl) stepHub(clusterInfo *v1alpha1.RainbondCluster, componentStatues []*v1.RbdComponentStatus) model.InstallStatus {
	if clusterInfo.Spec.ImageHub != nil { // custom image hub, do not prepare it by rainbond operator, progress set 100 directly
		return model.InstallStatus{
			StepName: StepPrepareHub,
			Status:   InstallStatusFinished,
			Progress: 100,
		}
	}

	status := model.InstallStatus{
		StepName: StepPrepareHub,
	}

	// prepare init component list
	initComponents := []*v1.RbdComponentStatus{}
	for _, cs := range componentStatues {
		if cs.ISInitComponent {
			initComponents = append(initComponents, cs)
		}
	}

	if len(initComponents) == 0 {
		// component not ready
		status.Status = InstallStatusWaiting
		return status
	}

	readyCount := 0
	for _, cs := range initComponents {
		if cs.Status == v1.ComponentStatusRunning {
			readyCount++
		}
	}

	if readyCount == len(initComponents) {
		status.Status = InstallStatusFinished
		status.Progress = 100
		return status
	}

	status.Status = InstallStatusProcessing
	status.Progress = (readyCount * 100) / len(initComponents)

	return status
}

// step 2 download rainbond
func (ic *InstallUseCaseImpl) stepDownload(clusterInfo *v1alpha1.RainbondCluster, pkgInfo *v1alpha1.RainbondPackage) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if pkgInfo.Status == nil {
		return model.InstallStatus{
			StepName: StepDownload,
			Status:   InstallStatusWaiting,
		}
	}

	condition := ic.handleRainbondPackageConditions(pkgInfo.Status.Conditions, v1alpha1.DownloadPackage)
	if condition == nil {
		return model.InstallStatus{
			StepName: StepDownload,
			Status:   InstallStatusWaiting,
		}
	}
	status := model.InstallStatus{
		StepName: StepDownload,
	}
	switch condition.Status {
	case v1alpha1.Running:
		status.Status = InstallStatusProcessing
		status.Progress = condition.Progress
	case v1alpha1.Completed:
		status.Status = InstallStatusFinished
		status.Progress = 100
	case v1alpha1.Failed:
		status.Status = InstallStatusFailed
		status.Progress = condition.Progress
		status.Message = condition.Message
		status.Reason = condition.Reason
	default:
		status.Status = InstallStatusWaiting
	}

	return status
}

// step 4 unpack rainbond
func (ic *InstallUseCaseImpl) stepUnpack(clusterInfo *v1alpha1.RainbondCluster, pkgInfo *v1alpha1.RainbondPackage) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if pkgInfo.Status == nil {
		return model.InstallStatus{
			StepName: StepUnpack,
			Status:   InstallStatusWaiting,
		}
	}
	condition := ic.handleRainbondPackageConditions(pkgInfo.Status.Conditions, v1alpha1.UnpackPackage)
	if condition == nil {
		return model.InstallStatus{
			StepName: StepUnpack,
			Status:   InstallStatusWaiting,
		}
	}
	status := model.InstallStatus{
		StepName: StepUnpack,
	}
	switch condition.Status {
	case v1alpha1.Running:
		status.Status = InstallStatusProcessing
		status.Progress = condition.Progress
	case v1alpha1.Completed:
		status.Status = InstallStatusFinished
		status.Progress = 100
	case v1alpha1.Failed:
		status.Status = InstallStatusFailed
		status.Progress = condition.Progress
		status.Message = condition.Message
		status.Reason = condition.Reason
	default:
		status.Status = InstallStatusWaiting
	}

	return status
}

// step 5 handle image, load and push image to image hub
func (ic *InstallUseCaseImpl) stepHandleImage(clusterInfo *v1alpha1.RainbondCluster, pkgInfo *v1alpha1.RainbondPackage) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())
	if pkgInfo.Status == nil {
		return model.InstallStatus{
			StepName: StepHandleImage,
			Status:   InstallStatusWaiting,
		}
	}
	condition := ic.handleRainbondPackageConditions(pkgInfo.Status.Conditions, v1alpha1.PushImage)
	if condition == nil {
		return model.InstallStatus{
			StepName: StepHandleImage,
			Status:   InstallStatusWaiting,
		}
	}
	status := model.InstallStatus{
		StepName: StepHandleImage,
	}
	switch condition.Status {
	case v1alpha1.Running:
		status.Status = InstallStatusProcessing
		status.Progress = condition.Progress
	case v1alpha1.Completed:
		status.Status = InstallStatusFinished
		status.Progress = 100
	case v1alpha1.Failed:
		status.Status = InstallStatusFailed
		status.Progress = condition.Progress
		status.Message = condition.Message
		status.Reason = condition.Reason
	default:
		status.Status = InstallStatusWaiting
	}

	return status
}

// step 6 create component
func (ic *InstallUseCaseImpl) stepCreateComponent(componentStatues []*v1.RbdComponentStatus, pkgInfo *v1alpha1.RainbondPackage) model.InstallStatus {
	defer commonutil.TimeConsume(time.Now())

	status := model.InstallStatus{
		StepName: StepInstallComponent,
		Status:   InstallStatusWaiting,
	}
	if pkgInfo.Status == nil {
		return status
	}

	condition := ic.handleRainbondPackageConditions(pkgInfo.Status.Conditions, v1alpha1.Ready)
	if condition == nil || condition.Status != v1alpha1.Completed {
		status.Status = InstallStatusWaiting
		return status
	}

	readyCount := 0
	for _, cs := range componentStatues {
		if cs.Status == v1.ComponentStatusRunning {
			readyCount++
		}
	}

	if readyCount == len(componentStatues) {
		status.Status = InstallStatusFinished
		status.Progress = 100
		return status
	}

	status.Status = InstallStatusProcessing
	status.Progress = (readyCount * 100) / len(componentStatues)

	return status
}

func (ic *InstallUseCaseImpl) handleRainbondPackageConditions(pkgConditions []v1alpha1.PackageCondition, wanted v1alpha1.PackageConditionType) *v1alpha1.PackageCondition {
	for _, condition := range pkgConditions {
		if condition.Type == wanted {
			return &condition
		}
	}
	return nil
}
