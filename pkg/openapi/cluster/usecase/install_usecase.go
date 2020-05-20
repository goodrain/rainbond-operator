package usecase

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type componentClaim struct {
	namespace       string
	name            string
	version         string
	imageRepository string
	imageName       string
	logLevel        string
	Configs         map[string]string
	isInit          bool
	replicas        *int32
}

func (c *componentClaim) image() string {
	return path.Join(c.imageRepository, c.imageName) + ":" + c.version
}

func parseComponentClaim(claim *componentClaim) *v1alpha1.RbdComponent {
	component := &v1alpha1.RbdComponent{}
	component.Namespace = claim.namespace
	component.Name = claim.name
	component.Spec.Version = claim.version
	component.Spec.Image = claim.image()
	component.Spec.Replicas = claim.replicas
	component.Spec.Configs = claim.Configs
	component.Spec.LogLevel = v1alpha1.ParseLogLevel(claim.logLevel)
	component.Spec.Type = claim.name
	labels := rbdutil.LabelsForRainbond(map[string]string{"name": claim.name})
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

	componentUsecase cluster.ComponentUsecase
}

// NewInstallUseCase new install case
func NewInstallUseCase(cfg *option.Config, rainbondKubeClient versioned.Interface, componentUsecase cluster.ComponentUsecase) *InstallUseCaseImpl {
	return &InstallUseCaseImpl{
		cfg:                cfg,
		namespace:          cfg.Namespace,
		rainbondKubeClient: rainbondKubeClient,
		componentUsecase:   componentUsecase,
	}
}

// Install install
func (ic *InstallUseCaseImpl) Install(req *v1.ClusterInstallReq) error {
	cluster, err := ic.rainbondKubeClient.RainbondV1alpha1().RainbondClusters(ic.namespace).Get(ic.cfg.ClusterName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return bcode.ErrClusterNotFound
		}
		return err
	}

	// check cluster
	if cluster.Status.KubernetesVersoin < "v1.13.0" {
		return bcode.ErrInvalidKubernetesVersion
	}

	// create rainbond volume
	if err := ic.createRainbondVolumes(req); err != nil {
		return err
	}

	if err := ic.createRainbondPackage(); err != nil {
		return err
	}
	return ic.createComponents(req, cluster)
}

func (ic *InstallUseCaseImpl) createRainbondVolumes(req *v1.ClusterInstallReq) error {
	rwx := setRainbondVolume("rainbondvolumerwx", ic.namespace, rbdutil.LabelsForAccessModeRWX(), req.RainbondVolumes.RWX)
	rwx.Spec.ImageRepository = ic.cfg.RainbondImageRepository
	if err := ic.createRainbondVolumeIfNotExists(rwx); err != nil {
		return err
	}
	if req.RainbondVolumes.RWO != nil {
		rwo := setRainbondVolume("rainbondvolumerwo", ic.namespace, rbdutil.LabelsForAccessModeRWO(), req.RainbondVolumes.RWO)
		rwo.Spec.ImageRepository = ic.cfg.RainbondImageRepository
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
	var storageRequest int32 = 1
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
			storageRequest = 21
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

	spec.StorageRequest = commonutil.Int32(storageRequest)
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

func (ic *InstallUseCaseImpl) createRainbondPackage() error {
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

func (ic *InstallUseCaseImpl) deleteRainbondPackage() error {
	reqLogger := log.WithValues("Namespace", ic.cfg.Namespace, "Name", ic.cfg.Rainbondpackage)
	pkg := &v1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ic.cfg.Rainbondpackage,
			Namespace: ic.cfg.Namespace,
		},
	}
	if err := ic.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(ic.cfg.Namespace).Delete(pkg.Name, &metav1.DeleteOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			reqLogger.Info("rainbondpackage already exists.")
			return nil
		}
		reqLogger.Error(err, "delete rainbond package")
		return fmt.Errorf("delete rainbond package: %v", err)
	}
	reqLogger.Info("successfully delete rainbondpackage")
	return nil
}

func (ic *InstallUseCaseImpl) genComponentClaims(req *v1.ClusterInstallReq, cluster *v1alpha1.RainbondCluster) map[string]*componentClaim {
	var defReplicas = commonutil.Int32(1)
	if cluster.Spec.EnableHA {
		defReplicas = commonutil.Int32(2)
	}

	var isInit bool
	imageRepository := constants.DefImageRepository
	if cluster.Spec.ImageHub == nil {
		isInit = true
	} else {
		imageRepository = path.Join(cluster.Spec.ImageHub.Domain, cluster.Spec.ImageHub.Namespace)
	}

	newClaim := func(name string) *componentClaim {
		defClaim := componentClaim{name: name, imageRepository: imageRepository, version: ic.cfg.RainbondVersion, replicas: defReplicas}
		defClaim.imageName = name
		return &defClaim
	}
	name2Claim := map[string]*componentClaim{
		"rbd-api":      newClaim("rbd-api"),
		"rbd-chaos":    newClaim("rbd-chaos"),
		"rbd-eventlog": newClaim("rbd-eventlog"),
		"rbd-monitor":  newClaim("rbd-monitor"),
		"rbd-mq":       newClaim("rbd-mq"),
		"rbd-worker":   newClaim("rbd-worker"),
		"rbd-webcli":   newClaim("rbd-webcli"),
	}
	if !ic.cfg.OnlyInstallRegion {
		name2Claim["rbd-app-ui"] = newClaim("rbd-app-ui")
	}
	name2Claim["metrics-server"] = newClaim("metrics-server")
	name2Claim["metrics-server"].version = "v0.3.6"
	name2Claim["rbd-repo"] = newClaim("rbd-repo")
	name2Claim["rbd-repo"].version = "6.16.0"

	if cluster.Spec.RegionDatabase == nil || (cluster.Spec.UIDatabase == nil && !ic.cfg.OnlyInstallRegion) {
		claim := newClaim("rbd-db")
		claim.version = "8.0.19"
		if cluster.Spec.EnableHA {
			claim.replicas = commonutil.Int32(3)
		}
		name2Claim["rbd-db"] = claim
	}

	if cluster.Spec.ImageHub == nil {
		claim := newClaim("rbd-hub")
		claim.imageName = "registry"
		claim.version = "2.6.2"
		claim.isInit = isInit
		name2Claim["rbd-hub"] = claim
	}

	name2Claim["rbd-gateway"] = newClaim("rbd-gateway")
	name2Claim["rbd-gateway"].isInit = isInit
	name2Claim["rbd-node"] = newClaim("rbd-node")
	name2Claim["rbd-node"].isInit = isInit
	name2Claim["rbd-node"].logLevel = "debug"

	if cluster.Spec.EtcdConfig == nil {
		claim := newClaim("rbd-etcd")
		claim.imageName = "etcd"
		claim.version = "v3.3.18"
		claim.isInit = isInit
		if cluster.Spec.EnableHA {
			claim.replicas = commonutil.Int32(3)
		}
		name2Claim["rbd-etcd"] = claim
	}

	if req.RainbondVolumes.RWX != nil && req.RainbondVolumes.RWX.CSIPlugin != nil {
		if req.RainbondVolumes.RWX.CSIPlugin.NFS != nil {
			name2Claim["nfs-provisioner"] = newClaim("nfs-provisioner")
			name2Claim["nfs-provisioner"].replicas = commonutil.Int32(1)
			name2Claim["nfs-provisioner"].isInit = isInit
		}
		if req.RainbondVolumes.RWX.CSIPlugin.AliyunNas != nil {
			name2Claim[constants.AliyunCSINasPlugin] = newClaim(constants.AliyunCSINasPlugin)
			name2Claim[constants.AliyunCSINasPlugin].isInit = isInit
			name2Claim[constants.AliyunCSINasProvisioner] = newClaim(constants.AliyunCSINasProvisioner)
			name2Claim[constants.AliyunCSINasProvisioner].isInit = isInit
			name2Claim[constants.AliyunCSINasProvisioner].replicas = commonutil.Int32(1)
		}
	}
	if req.RainbondVolumes.RWO != nil && req.RainbondVolumes.RWO.CSIPlugin != nil {
		if req.RainbondVolumes.RWO.CSIPlugin.AliyunCloudDisk != nil {
			name2Claim[constants.AliyunCSIDiskPlugin] = newClaim(constants.AliyunCSIDiskPlugin)
			name2Claim[constants.AliyunCSIDiskPlugin].isInit = isInit
			name2Claim[constants.AliyunCSIDiskProvisioner] = newClaim(constants.AliyunCSIDiskProvisioner)
			name2Claim[constants.AliyunCSIDiskProvisioner].isInit = isInit
			name2Claim[constants.AliyunCSIDiskProvisioner].replicas = commonutil.Int32(1)
		}
	}

	return name2Claim
}

func (ic *InstallUseCaseImpl) createComponents(req *v1.ClusterInstallReq, cluster *v1alpha1.RainbondCluster) error {
	claims := ic.genComponentClaims(req, cluster)

	for _, claim := range claims {
		reqLogger := log.WithValues("Namespace", claim.namespace, "Name", claim.name)
		// update image repository for priority components
		if claim.isInit {
			claim.imageRepository = cluster.Spec.RainbondImageRepository
		}
		if ic.cfg.InstallMode == string(v1alpha1.InstallationModeFullOnline) {
			claim.imageRepository = cluster.Spec.RainbondImageRepository
		}
		data := parseComponentClaim(claim)
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

// RestartPackage -
func (ic *InstallUseCaseImpl) RestartPackage() error {
	if err := ic.deleteRainbondPackage(); err != nil {
		return err
	}
	if err := ic.createRainbondPackage(); err != nil {
		return err
	}
	return nil
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
