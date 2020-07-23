package usecase

import (
	"fmt"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/nodestore"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type clusterUsecase struct {
	cfg            *option.Config
	clientset      kubernetes.Interface
	rainbondClient versioned.Interface
	namespace      string
	clusterName    string

	cptUcase   cluster.ComponentUsecase
	repo       cluster.Repository
	nodestorer nodestore.Interface
}

// NewClusterUsecase creates a new cluster.Usecase.
func NewClusterUsecase(cfg *option.Config, repo cluster.Repository, cptUcase cluster.ComponentUsecase, nodestorer nodestore.Interface) cluster.Usecase {
	return &clusterUsecase{
		cfg:            cfg,
		clientset:      cfg.KubeClient,
		rainbondClient: cfg.RainbondKubeClient,
		namespace:      cfg.Namespace,
		clusterName:    cfg.ClusterName,
		repo:           repo,
		cptUcase:       cptUcase,
		nodestorer:     nodestorer,
	}
}

func (c *clusterUsecase) PreCheck() (*v1.ClusterPreCheckResp, error) {
	cls, err := c.getCluster()
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("get rainbond clsuter: %v", err)
		}
	}

	conditions := convertClusterConditions(cls)
	pass := true
	for _, condition := range conditions {
		if condition.Status != "True" {
			pass = false
			break
		}
	}

	return &v1.ClusterPreCheckResp{
		Pass:       pass,
		Conditions: conditions,
	}, nil

}

// UnInstall uninstall cluster reset cluster
func (c *clusterUsecase) UnInstall() error {
	deleteOpts := &metav1.DeleteOptions{
		GracePeriodSeconds: commonutil.Int64(0),
	}
	// delete pv based on pvc
	claims, err := c.cfg.KubeClient.CoreV1().PersistentVolumeClaims(c.namespace).List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list pv: %v", err)
	}
	for _, claim := range claims.Items {
		if claim.Spec.VolumeName == "" {
			// unbound pvc
			continue
		}
		if err := c.cfg.KubeClient.CoreV1().PersistentVolumes().Delete(claim.Spec.VolumeName, &metav1.DeleteOptions{}); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("delete persistent volume claims: %v", err)
		}
	}
	// delete pvc
	if err := c.cfg.KubeClient.CoreV1().PersistentVolumeClaims(c.namespace).DeleteCollection(deleteOpts, metav1.ListOptions{}); err != nil {
		return fmt.Errorf("delete persistent volume claims: %v", err)
	}
	// delte rainbond components
	if err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(c.cfg.Namespace).DeleteCollection(deleteOpts, metav1.ListOptions{}); err != nil {
		return fmt.Errorf("delete component: %v", err)
	}
	// delete rainbond packages
	if err := c.rainbondClient.RainbondV1alpha1().RainbondPackages(c.namespace).DeleteCollection(deleteOpts, metav1.ListOptions{}); err != nil {
		return fmt.Errorf("delete rainbond package: %v", err)
	}
	// delete rainbondvolume
	if err := c.rainbondClient.RainbondV1alpha1().RainbondVolumes(c.namespace).DeleteCollection(deleteOpts, metav1.ListOptions{}); err != nil {
		return fmt.Errorf("delete rainbond volume: %v", err)
	}
	// delete storage class and csidriver
	rainbondLabelSelector := fields.SelectorFromSet(rbdutil.LabelsForRainbond(nil)).String()
	if err := c.clientset.StorageV1().StorageClasses().DeleteCollection(deleteOpts, metav1.ListOptions{LabelSelector: rainbondLabelSelector}); err != nil {
		return fmt.Errorf("delete storageclass: %v", err)
	}
	if err := c.clientset.StorageV1().StorageClasses().Delete("rainbondslsc", &metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("delete storageclass rainbondslsc: %v", err)
		}
	}
	if err := c.clientset.StorageV1().StorageClasses().Delete("rainbondsssc", &metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("delete storageclass rainbondsssc: %v", err)
		}
	}
	if err := c.clientset.StorageV1beta1().CSIDrivers().DeleteCollection(deleteOpts, metav1.ListOptions{LabelSelector: rainbondLabelSelector}); err != nil {
		return fmt.Errorf("delete csidriver: %v", err)
	}
	// delete rainbond cluster
	if err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(c.cfg.Namespace).Delete(c.cfg.ClusterName, deleteOpts); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("delete rainbond cluster: %v", err)
		}
	}
	return nil
}

// Status status
func (c *clusterUsecase) Status() (*model.ClusterStatus, error) {
	rainbondCluster, err := c.getCluster()
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("get rainbond clsuter: %v", err)
		}
		rainbondCluster = nil
	}
	rainbondPackage, err := c.getRainbondPackage()
	if err != nil {
		return nil, fmt.Errorf("get package: %v", err)
	}
	componentStatuses, err := c.cptUcase.List(false)
	if err != nil {
		return nil, fmt.Errorf("list rainobnd componentStatuses: %v", err)
	}
	cpts := c.cptUcase.ListComponents()

	status := c.handleStatus(rainbondCluster, rainbondPackage, componentStatuses)
	c.hackClusterInfo(rainbondCluster, &status)
	c.wrapFailureReason(&status, rainbondPackage, cpts)
	status.TestMode = c.cfg.TestMode

	return &status, nil
}

// StatusInfo returns the information of rainbondcluster status.
func (c *clusterUsecase) StatusInfo() (*v1.ClusterStatusInfo, error) {
	cluster, err := c.getCluster()
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, bcode.NotFound
		}
		return nil, err
	}

	// StorageClass
	var storageClasses []*v1.StorageClass
	for _, c := range cluster.Status.StorageClasses {
		class := &v1.StorageClass{
			Name:        c.Name,
			Provisioner: c.Provisioner,
			AccessMode:  string(c.AccessMode),
		}
		if class.AccessMode == "" {
			class.AccessMode = "Unknown"
		}
		storageClasses = append(storageClasses, class)
	}

	// nodes for gateway and chaos
	getNodes := func(nodes *rainbondv1alpha1.AvailableNodes) *v1.AvailableNodes {
		setNodes := func(k8sNodes []*rainbondv1alpha1.K8sNode) []*v1.K8sNode {
			var nodes []*v1.K8sNode
			for _, node := range k8sNodes {
				nodes = append(nodes, &v1.K8sNode{
					Name:       node.Name,
					InternalIP: node.InternalIP,
					ExternalIP: node.ExternalIP,
				})
			}
			return nodes
		}
		specifiedNodes := setNodes(nodes.SpecifiedNodes)
		masterNodes := setNodes(nodes.MasterNodes)
		return &v1.AvailableNodes{
			SpecifiedNodes: specifiedNodes,
			MasterNodes:    masterNodes,
		}
	}
	gatewayNodes := getNodes(cluster.Status.GatewayAvailableNodes)
	chaosNodes := getNodes(cluster.Status.ChaosAvailableNodes)

	result := &v1.ClusterStatusInfo{
		GatewayAvailableNodes: gatewayNodes,
		ChaosAvailableNodes:   chaosNodes,
		StorageClasses:        storageClasses,
	}

	return result, nil
}

func (c *clusterUsecase) ClusterNodes(query string, runGateway bool) []*v1.K8sNode {
	return c.nodestorer.SearchByIIP(query, runGateway)
}

func (c *clusterUsecase) ValidateNodes(nodes []*v1.K8sNode, runGateway bool) []*v1.K8sNode {
	var invalidNodes []*v1.K8sNode
	for _, node := range nodes {
		found := c.nodestorer.SearchByIIP(node.InternalIP, runGateway)
		if len(found) == 0 {
			invalidNodes = append(invalidNodes, node)
		}
	}
	return invalidNodes
}

func (c *clusterUsecase) CompleteNodes(nodes []*v1.K8sNode, runGateway bool) ([]*v1.K8sNode, []*v1.K8sNode) {
	var validNodes []*v1.K8sNode
	var invalidNodes []*v1.K8sNode
	for _, node := range nodes {
		found := c.nodestorer.SearchByIIP(node.InternalIP, runGateway)
		if len(found) == 0 {
			invalidNodes = append(invalidNodes, node)
			continue
		}
		validNodes = append(validNodes, found[0])
	}
	return validNodes, invalidNodes
}

func (c *clusterUsecase) hackClusterInfo(rainbondCluster *rainbondv1alpha1.RainbondCluster, status *model.ClusterStatus) {
	if status.FinalStatus == model.Waiting || status.FinalStatus == model.Initing {
		log.Info("cluster is not ready")
		return
	}
	// init not finished
	if rainbondCluster.Status == nil {
		log.Info("cluster's status is not ready")
		return
	}
	//now cluster has init successfully, prepare cluster info
	for _, sc := range rainbondCluster.Status.StorageClasses {
		if sc.Name != "rainbondslsc" && sc.Name != "rainbondsssc" {
			status.ClusterInfo.Storage = append(status.ClusterInfo.Storage, model.Storage{
				Name:        sc.Name,
				Provisioner: sc.Provisioner,
			})
		}
	}

	// get install version from config
	status.ClusterInfo.InstallVersion = c.cfg.RainbondVersion + "-v1.0.0"

	// get installID from cluster's annotations
	var eid string
	if rainbondCluster.Annotations != nil {
		if value, ok := rainbondCluster.Annotations["install_id"]; ok && value != "" {
			status.ClusterInfo.InstallID = value
		}
		if value, ok := rainbondCluster.Annotations["enterprise_id"]; ok && value != "" {
			eid = value
		}
	}

	// get enterprise from repo
	status.ClusterInfo.EnterpriseID = c.repo.EnterpriseID(eid)
}

// no rainbondcluster cr means cluster status is waiting
// rainbondcluster cr without status parameter means cluster status is initing
// rainbondcluster cr with status parameter means cluster status is setting
// rainbondpackage cr means cluster status is installing or running
// rbdcomponent cr means cluster stauts is installing or running
// all rbdcomponent cr are running means cluster status is running
// rbdcomponent cr has pod with status terminal means cluster status is uninstalling
func (c *clusterUsecase) handleStatus(rainbondCluster *rainbondv1alpha1.RainbondCluster, rainbondPackage *rainbondv1alpha1.RainbondPackage, componentStatusList []*v1.RbdComponentStatus) model.ClusterStatus {
	reqLogger := log.WithValues("Namespace", c.cfg.Namespace)

	rainbondClusterStatus := c.handleRainbondClusterStatus(rainbondCluster)
	rainbondPackageStatus := c.handlePackageStatus(rainbondCluster, rainbondPackage)
	componentStatus := c.handleComponentStatus(rainbondCluster, componentStatusList)
	reqLogger.V(6).Info(fmt.Sprintf("cluster: %s; package: %s; component: %s \n", rainbondClusterStatus.FinalStatus, rainbondPackageStatus.FinalStatus, componentStatus.FinalStatus))
	if componentStatus.FinalStatus == model.UnInstalling {
		rainbondClusterStatus.FinalStatus = model.UnInstalling
		return rainbondClusterStatus
	}
	if rainbondClusterStatus.FinalStatus == model.Waiting {
		return rainbondClusterStatus
	}

	if rainbondClusterStatus.FinalStatus == model.Initing {
		return rainbondClusterStatus
	}

	if rainbondPackageStatus.FinalStatus == model.Setting && componentStatus.FinalStatus == model.Setting {
		reqLogger.Info("setting status")
		rainbondClusterStatus.FinalStatus = model.Setting
		return rainbondClusterStatus
	}

	if componentStatus.FinalStatus == model.Running {
		reqLogger.Info("running status")
		rainbondClusterStatus.FinalStatus = model.Running
		return rainbondClusterStatus
	}

	if rainbondPackageStatus.FinalStatus == model.Installing || componentStatus.FinalStatus == model.Installing {
		reqLogger.Info("installing status")
		rainbondClusterStatus.FinalStatus = model.Installing
		return rainbondClusterStatus
	}

	return rainbondClusterStatus
}

func (c *clusterUsecase) handleRainbondClusterStatus(rainbondCluster *rainbondv1alpha1.RainbondCluster) model.ClusterStatus {
	status := model.ClusterStatus{
		FinalStatus: model.Waiting,
		ClusterInfo: model.ClusterInfo{},
	}

	if rainbondCluster == nil {
		return status
	}
	if rainbondCluster.Status == nil {
		status.FinalStatus = model.Initing
		return status
	}
	status.FinalStatus = model.Setting

	return status
}

func (c *clusterUsecase) handlePackageStatus(cluster *rainbondv1alpha1.RainbondCluster, rainbondPackage *rainbondv1alpha1.RainbondPackage) model.ClusterStatus {
	status := model.ClusterStatus{
		FinalStatus: model.Setting,
	}
	if cluster == nil || cluster.Status == nil || !cluster.Spec.ConfigCompleted {
		return status
	}
	if rainbondPackage == nil {
		return status
	}
	status.FinalStatus = model.Installing
	return status
}

func (c *clusterUsecase) handleComponentStatus(cluster *rainbondv1alpha1.RainbondCluster, componentList []*v1.RbdComponentStatus) model.ClusterStatus {
	status := model.ClusterStatus{
		FinalStatus: model.Setting,
	}
	if cluster == nil || cluster.Status == nil || !cluster.Spec.ConfigCompleted {
		return status
	}
	if len(componentList) == 0 {
		return status
	}
	status.FinalStatus = model.Installing

	readyCount := 0
	terminal := false
	for _, component := range componentList {
		if component.Status == v1.ComponentStatusRunning {
			readyCount++
		}
		if component.Status == v1.ComponentStatusTerminating { //TODO terminal uninstalling
			terminal = true
		}
	}
	if terminal {
		status.FinalStatus = model.UnInstalling
		return status
	}
	if readyCount != len(componentList) {
		return status
	}
	status.FinalStatus = model.Running
	return status
}

// Init init
func (c *clusterUsecase) Init() error {
	_, err := c.createCluster()
	return err
}

func (c *clusterUsecase) getCluster() (*rainbondv1alpha1.RainbondCluster, error) {
	return c.rainbondClient.RainbondV1alpha1().RainbondClusters(c.namespace).Get(c.clusterName, metav1.GetOptions{})
}

func (c *clusterUsecase) createCluster() (*rainbondv1alpha1.RainbondCluster, error) {
	installMode := rainbondv1alpha1.InstallationModeWithoutPackage
	if c.cfg.InstallMode == string(rainbondv1alpha1.InstallationModeWithPackage) {
		installMode = rainbondv1alpha1.InstallationModeWithPackage
	}
	if c.cfg.InstallMode == string(rainbondv1alpha1.InstallationModeFullOnline) {
		installMode = rainbondv1alpha1.InstallationModeFullOnline
	}

	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.cfg.Namespace,
			Name:      c.cfg.ClusterName,
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			RainbondImageRepository: c.cfg.RainbondImageRepository,
			InstallMode:             installMode,
			SentinelImage:           c.cfg.SentinelImage,
		},
	}

	annotations := make(map[string]string)
	annotations["install_id"] = uuidutil.NewUUID()
	annotations["enterprise_id"] = c.repo.EnterpriseID("")
	cluster.Annotations = annotations

	return c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondClusters(c.cfg.Namespace).Create(cluster)
}

func (c *clusterUsecase) getRainbondPackage() (*rainbondv1alpha1.RainbondPackage, error) {
	pkg, err := c.cfg.RainbondKubeClient.RainbondV1alpha1().RainbondPackages(c.cfg.Namespace).Get(c.cfg.Rainbondpackage, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return pkg, nil
}

func (c *clusterUsecase) wrapFailureReason(status *model.ClusterStatus, pkg *rainbondv1alpha1.RainbondPackage, cpts []*rainbondv1alpha1.RbdComponent) {
	// reason of failure from rainbond package
	var reasons []string
	if pkg != nil && pkg.Status != nil {
		for _, cdt := range pkg.Status.Conditions {
			if cdt.Status != rainbondv1alpha1.Failed {
				continue
			}
			reasons = append(reasons, cdt.Reason)
		}
	}

	// reason of failure from rainbond components
	for _, cpt := range cpts {
		// No status has been generated yet
		if cpt.Status == nil {
			continue
		}
		// component is ready
		if cpt.Status.Replicas == cpt.Status.ReadyReplicas {
			continue
		}
		// If the number of container restarts in the component exceeds 3 times,
		// the status of the component is considered failed
		pods, err := c.cptUcase.ListPodsByComponent(cpt)
		if err != nil {
			log.V(4).Info(fmt.Sprintf("list pods by component: %v", err), "component name", cpt.Name)
			continue
		}
		for _, pod := range pods {
			if containerRestartsTimes(pod) <= 3 {
				continue
			}
			reasons = append(reasons, fmt.Sprintf("%s failed", cpt.Name))
			break
		}
	}
	status.Reasons = reasons
}

func containerRestartsTimes(pod *corev1.Pod) int {
	for _, status := range pod.Status.ContainerStatuses {
		return int(status.RestartCount)
	}
	return 0
}
