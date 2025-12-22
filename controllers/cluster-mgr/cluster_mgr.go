package clustermgr

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers/cluster-mgr/precheck"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"github.com/pquerna/ffjson/ffjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// RdbHubCredentialsName name for rbd-hub-credentials
	RdbHubCredentialsName = "rbd-hub-credentials"
)

var provisionerAccessModes = map[string]corev1.PersistentVolumeAccessMode{
	// Kubernetes Internal Provisioner.
	// More info: https://github.com/kubernetes/kubernetes/tree/v1.17.3/pkg/volume
	"kubernetes.io/aws-ebs":         corev1.ReadWriteOnce,
	"kubernetes.io/azure-disk":      corev1.ReadWriteOnce,
	"kubernetes.io/azure-file":      corev1.ReadWriteMany,
	"kubernetes.io/cephfs":          corev1.ReadWriteMany,
	"kubernetes.io/cinder":          corev1.ReadWriteOnce,
	"kubernetes.io/fc":              corev1.ReadWriteOnce,
	"kubernetes.io/flocker":         corev1.ReadWriteOnce,
	"kubernetes.io/gce-pd":          corev1.ReadWriteOnce,
	"kubernetes.io/glusterfs":       corev1.ReadWriteMany,
	"kubernetes.io/iscsi":           corev1.ReadWriteOnce,
	"kubernetes.io/nfs":             corev1.ReadWriteMany,
	"kubernetes.io/portworx-volume": corev1.ReadWriteMany,
	"kubernetes.io/quobyte":         corev1.ReadWriteMany,
	"kubernetes.io/rbd":             corev1.ReadWriteMany,
	"kubernetes.io/scaleio":         corev1.ReadWriteMany,
	"kubernetes.io/storageos":       corev1.ReadWriteMany,
	// Alibaba csi plugins for kubernetes.
	// More info: https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/tree/master/pkg
	"cpfsplugin.csi.alibabacloud.com": corev1.ReadWriteMany,
	"diskplugin.csi.alibabacloud.com": corev1.ReadWriteOnce,
	"alicloud/disk":                   corev1.ReadWriteOnce,
	"lvmplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
	"memplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
	"nasplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
	"ossplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
}

type k8sNodesSortByName []*rainbondv1alpha1.K8sNode

func (s k8sNodesSortByName) Len() int           { return len(s) }
func (s k8sNodesSortByName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s k8sNodesSortByName) Less(i, j int) bool { return s[i].Name < s[j].Name }

// RainbondClusteMgr -
type RainbondClusteMgr struct {
	ctx    context.Context
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger

	cluster *rainbondv1alpha1.RainbondCluster
}

// NewClusterMgr new Cluster Mgr
func NewClusterMgr(ctx context.Context, client client.Client, log logr.Logger, cluster *rainbondv1alpha1.RainbondCluster, scheme *runtime.Scheme) *RainbondClusteMgr {
	mgr := &RainbondClusteMgr{
		ctx:     ctx,
		client:  client,
		log:     log,
		cluster: cluster,
		scheme:  scheme,
	}
	return mgr
}

func (r *RainbondClusteMgr) listStorageClasses() []*rainbondv1alpha1.StorageClass {
	r.log.V(6).Info("start listing available storage classes")

	storageClassList := &storagev1.StorageClassList{}
	var opts []client.ListOption
	ctx, cancel := context.WithTimeout(r.ctx, time.Second*10)
	defer cancel()
	if err := r.client.List(ctx, storageClassList, opts...); err != nil {
		r.log.Error(err, "list storageclass")
		return nil
	}

	var storageClasses []*rainbondv1alpha1.StorageClass
	for _, sc := range storageClassList.Items {
		storageClass := &rainbondv1alpha1.StorageClass{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
			AccessMode:  provisionerAccessModes[sc.Provisioner],
		}
		storageClasses = append(storageClasses, storageClass)
	}
	r.log.V(6).Info("listing available storage classes success")
	return storageClasses
}

// GenerateRainbondClusterStatus creates the final rainbondcluster status for a rainbondcluster, given the
// internal rainbondcluster status.
func (r *RainbondClusteMgr) GenerateRainbondClusterStatus() (*rainbondv1alpha1.RainbondClusterStatus, error) {
	r.log.V(6).Info("start generating status")

	masterRoleLabel, err := r.getMasterRoleLabel()
	if err != nil {
		return nil, fmt.Errorf("get master role label: %v", err)
	}

	s := &rainbondv1alpha1.RainbondClusterStatus{
		MasterRoleLabel: masterRoleLabel,
		StorageClasses:  r.listStorageClasses(),
	}

	if r.checkIfImagePullSecretExists() {
		s.ImagePullSecret = &corev1.LocalObjectReference{Name: RdbHubCredentialsName}
	}

	var masterNodesForGateway []*rainbondv1alpha1.K8sNode
	var masterNodesForChaos []*rainbondv1alpha1.K8sNode
	if masterRoleLabel != "" {
		masterNodesForGateway = r.listMasterNodesForGateway(masterRoleLabel)
		masterNodesForChaos = r.listMasterNodes(masterRoleLabel)
	}
	s.GatewayAvailableNodes = &rainbondv1alpha1.AvailableNodes{
		SpecifiedNodes: r.listSpecifiedGatewayNodes(),
		MasterNodes:    masterNodesForGateway,
	}
	s.ChaosAvailableNodes = &rainbondv1alpha1.AvailableNodes{
		SpecifiedNodes: r.listSpecifiedChaosNodes(),
		MasterNodes:    masterNodesForChaos,
	}

	// conditions for rainbond cluster status
	s.Conditions = r.generateConditions()
	r.log.V(6).Info("generating status success")
	return s, nil
}

func (r *RainbondClusteMgr) getMasterRoleLabel() (string, error) {
	nodes := &corev1.NodeList{}
	if err := r.client.List(r.ctx, nodes); err != nil {
		r.log.Error(err, "list nodes: %v", err)
		return "", nil
	}
	var label string
	for _, node := range nodes.Items {
		for key := range node.Labels {
			if key == rainbondv1alpha1.LabelNodeRolePrefix+"master" {
				label = key
			}
			if key == rainbondv1alpha1.NodeLabelRole && label != rainbondv1alpha1.LabelNodeRolePrefix+"master" {
				label = key
			}
		}
	}
	return label, nil
}

func (r *RainbondClusteMgr) listSpecifiedGatewayNodes() []*rainbondv1alpha1.K8sNode {
	nodes := r.listNodesByLabels(map[string]string{
		constants.SpecialGatewayLabelKey: "",
	})
	// Filtering nodes with port conflicts
	// check gateway ports
	return rbdutil.FilterNodesWithPortConflicts(nodes)
}

func (r *RainbondClusteMgr) listSpecifiedChaosNodes() []*rainbondv1alpha1.K8sNode {
	return r.listNodesByLabels(map[string]string{
		constants.SpecialChaosLabelKey: "",
	})
}

func (r *RainbondClusteMgr) listNodesByLabels(labels map[string]string) []*rainbondv1alpha1.K8sNode {
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(labels),
	}
	ctx, cancel := context.WithTimeout(r.ctx, time.Second*10)
	defer cancel()
	if err := r.client.List(ctx, nodeList, listOpts...); err != nil {
		r.log.Error(err, "list nodes")
		return nil
	}

	findIP := func(addresses []corev1.NodeAddress, addressType corev1.NodeAddressType) string {
		for _, address := range addresses {
			if address.Type == addressType {
				return address.Address
			}
		}
		return ""
	}

	var k8sNodes []*rainbondv1alpha1.K8sNode
	for _, node := range nodeList.Items {
		k8sNode := &rainbondv1alpha1.K8sNode{
			Name:       node.Name,
			InternalIP: findIP(node.Status.Addresses, corev1.NodeInternalIP),
			ExternalIP: findIP(node.Status.Addresses, corev1.NodeExternalIP),
		}
		k8sNodes = append(k8sNodes, k8sNode)
	}

	sort.Sort(k8sNodesSortByName(k8sNodes))

	return k8sNodes
}

func (r *RainbondClusteMgr) listMasterNodesForGateway(masterLabel string) []*rainbondv1alpha1.K8sNode {
	nodes := r.listMasterNodes(masterLabel)
	// Filtering nodes with port conflicts
	// check gateway ports
	return rbdutil.FilterNodesWithPortConflicts(nodes)
}

func (r *RainbondClusteMgr) listMasterNodes(masterRoleLabelKey string) []*rainbondv1alpha1.K8sNode {
	labels := k8sutil.MaterRoleLabel(masterRoleLabelKey)
	return r.listNodesByLabels(labels)
}

// CreateImagePullSecret create image pull secret
func (r *RainbondClusteMgr) CreateImagePullSecret() error {
	var secret corev1.Secret
	if err := r.client.Get(r.ctx, types.NamespacedName{Namespace: r.cluster.Namespace, Name: RdbHubCredentialsName}, &secret); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	}
	if config, exist := secret.Data[".dockerconfigjson"]; exist && string(config) == string(r.generateDockerConfig()) {
		r.log.V(5).Info("dockerconfig not change")
		return nil
	}
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RdbHubCredentialsName,
			Namespace: r.cluster.Namespace,
		},
		Data: map[string][]byte{
			".dockerconfigjson": r.generateDockerConfig(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	if err := controllerutil.SetControllerReference(r.cluster, &secret, r.scheme); err != nil {
		return fmt.Errorf("set controller reference for secret %s: %v", RdbHubCredentialsName, err)
	}

	err := r.client.Create(r.ctx, &secret)
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(7).Info("update image pull secret", "name", RdbHubCredentialsName)
			err = r.client.Update(r.ctx, &secret)
			if err == nil {
				return nil
			}
		}
		return fmt.Errorf("create secret for pulling images: %v", err)
	}

	return nil
}

func (r *RainbondClusteMgr) checkIfImagePullSecretExists() bool {
	secret := &corev1.Secret{}
	err := r.client.Get(r.ctx, types.NamespacedName{Namespace: r.cluster.Namespace, Name: RdbHubCredentialsName}, secret)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			r.log.Info(fmt.Sprintf("get secret %s: %v", RdbHubCredentialsName, err))
		}
		return false
	}
	return true
}

func (r *RainbondClusteMgr) generateDockerConfig() []byte {
	type dockerConfig struct {
		Auths map[string]map[string]string `json:"auths"`
	}

	username, password := r.cluster.Spec.ImageHub.Username, r.cluster.Spec.ImageHub.Password
	auth := map[string]string{
		"username": username,
		"password": password,
		"auth":     base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
	}

	dockercfg := dockerConfig{
		Auths: map[string]map[string]string{
			r.cluster.Spec.ImageHub.Domain: auth,
		},
	}

	bytes, _ := ffjson.Marshal(dockercfg)
	return bytes
}

func (r *RainbondClusteMgr) checkIfRbdNodeReady() error {
	cpt := &rainbondv1alpha1.RbdComponent{}
	if err := r.client.Get(r.ctx, types.NamespacedName{Namespace: r.cluster.Namespace, Name: "rbd-node"}, cpt); err != nil {
		return err
	}

	if cpt.Status.ReadyReplicas == 0 || cpt.Status.ReadyReplicas != cpt.Status.Replicas {
		return fmt.Errorf("no ready replicas for rbdcomponent rbd-node")
	}

	return nil
}

func (r *RainbondClusteMgr) generateConditions() []rainbondv1alpha1.RainbondClusterCondition {
	// region database
	spec := r.cluster.Spec
	if spec.RegionDatabase != nil && !r.isConditionTrue(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseRegion) {
		preChecker := precheck.NewDatabasePrechecker(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseRegion, spec.RegionDatabase)
		condition := preChecker.Check()
		r.cluster.Status.UpdateCondition(&condition)
	}

	// console database
	if spec.UIDatabase != nil && !r.isConditionTrue(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseConsole) {
		preChecker := precheck.NewDatabasePrechecker(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseConsole, spec.UIDatabase)
		condition := preChecker.Check()
		r.cluster.Status.UpdateCondition(&condition)
	}

	// image repository
	if spec.ImageHub != nil && !r.isConditionTrue(rainbondv1alpha1.RainbondClusterConditionTypeImageRepository) {
		preChecker := precheck.NewImageRepoPrechecker(r.ctx, r.log, r.cluster)
		condition := preChecker.Check()
		r.cluster.Status.UpdateCondition(&condition)
	}

	// kubernetes version
	if !r.isConditionTrue(rainbondv1alpha1.RainbondClusterConditionTypeKubernetesVersion) {
		k8sVersion := precheck.NewK8sVersionPrechecker(r.ctx, r.log, r.client)
		condition := k8sVersion.Check()
		r.cluster.Status.UpdateCondition(&condition)
	}

	storagePreChecker := precheck.NewStorage(r.ctx, r.client, r.cluster.GetNamespace(), r.cluster.Spec.RainbondVolumeSpecRWX)
	storageCondition := storagePreChecker.Check()
	r.cluster.Status.UpdateCondition(&storageCondition)

	if r.cluster.Spec.InstallMode != rainbondv1alpha1.InstallationModeOffline {
		dnsPrechecker := precheck.NewDNSPrechecker(r.cluster, r.log)
		dnsCondition := dnsPrechecker.Check()
		r.cluster.Status.UpdateCondition(&dnsCondition)
	}
	// disable kube-system namespace pod check
	// k8sStatusPrechecker := precheck.NewK8sStatusPrechecker(r.ctx, r.cluster, r.client, r.log)
	// k8sStatusCondition := k8sStatusPrechecker.Check()
	// r.cluster.Status.UpdateCondition(&k8sStatusCondition)

	memory := precheck.NewMemory(r.ctx, r.log, r.client)
	memoryCondition := memory.Check()
	r.cluster.Status.UpdateCondition(&memoryCondition)

	// container network
	if r.cluster.Spec.SentinelImage != "" {
		containerNetworkPrechecker := precheck.NewContainerNetworkPrechecker(r.ctx, r.client, r.scheme, r.log, r.cluster)
		containerNetworkCondition := containerNetworkPrechecker.Check()
		r.cluster.Status.UpdateCondition(&containerNetworkCondition)
	}

	if idx, condition := r.cluster.Status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeRunning); idx == -1 || condition.Status != corev1.ConditionTrue {
		running := r.runningCondition()
		r.cluster.Status.UpdateCondition(&running)
	}

	return r.cluster.Status.Conditions
}

func (r *RainbondClusteMgr) isConditionTrue(typ3 rainbondv1alpha1.RainbondClusterConditionType) bool {

	_, condition := r.cluster.Status.GetCondition(typ3)

	if condition != nil && condition.Status == corev1.ConditionTrue {
		return true
	}
	return false
}

func (r *RainbondClusteMgr) falseConditionNow(typ3 rainbondv1alpha1.RainbondClusterConditionType) *rainbondv1alpha1.RainbondClusterCondition {
	idx, _ := r.cluster.Status.GetCondition(typ3)
	if idx != -1 {
		return nil
	}
	return &rainbondv1alpha1.RainbondClusterCondition{
		Type:              typ3,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
		Reason:            "InProgress",
		Message:           fmt.Sprintf("precheck for %s is in progress", string(typ3)),
	}
}

func (r *RainbondClusteMgr) runningCondition() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeRunning,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	// list all rbdcomponents
	rbdcomponents, err := r.listRbdComponents()
	if err != nil {
		return rbdutil.FailCondition(condition, "ListRbdComponentFailed", err.Error())
	}

	if len(rbdcomponents) < 10 {
		return rbdutil.FailCondition(condition, "InsufficientRbdComponent",
			fmt.Sprintf("insufficient number of rbdcomponents. expect %d rbdcomponents, but got %d", 10, len(rbdcomponents)))
	}

	for _, cpt := range rbdcomponents {
		idx, c := cpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
		if idx == -1 {
			return rbdutil.FailCondition(condition, "RbdComponentReadyNotFound",
				fmt.Sprintf("condition 'RbdComponentReady' not found for %s", cpt.GetName()))
		}
		if c.Status == corev1.ConditionFalse {
			return rbdutil.FailCondition(condition, "RbdComponentNotReady",
				fmt.Sprintf("rbdcomponent(%s) not ready", cpt.GetName()))
		}
	}

	return condition
}

func (r *RainbondClusteMgr) listRbdComponents() ([]rainbondv1alpha1.RbdComponent, error) {
	rbdcomponentList := &rainbondv1alpha1.RbdComponentList{}
	err := r.client.List(r.ctx, rbdcomponentList, client.InNamespace(r.cluster.Namespace))
	if err != nil {
		return nil, err
	}
	return rbdcomponentList.Items, nil
}

// CreateOrUpdateMonitoringResources 在集群就绪后创建监控资源
func (r *RainbondClusteMgr) CreateOrUpdateMonitoringResources() error {
	r.log.Info("Creating health-console monitoring resources after cluster is ready")

	monitorNamespace := rbdutil.GetenvDefault("RBD_MONITOR_NAMESPACE", "rbd-monitor-system")

	// 1. 创建命名空间
	if err := r.createMonitorNamespace(monitorNamespace); err != nil {
		return fmt.Errorf("create monitor namespace: %v", err)
	}

	// 2. 创建 ConfigMap
	if err := r.createHealthConsoleConfigMap(monitorNamespace); err != nil {
		return fmt.Errorf("create health-console configmap: %v", err)
	}

	// 3. 创建 Secret (从实际配置获取)
	if err := r.createHealthConsoleSecret(monitorNamespace); err != nil {
		return fmt.Errorf("create health-console secret: %v", err)
	}

	// 4. 创建 Deployment
	if err := r.createHealthConsoleDeployment(monitorNamespace); err != nil {
		return fmt.Errorf("create health-console deployment: %v", err)
	}

	// 5. 创建 Service
	if err := r.createHealthConsoleService(monitorNamespace); err != nil {
		return fmt.Errorf("create health-console service: %v", err)
	}

	// 6. 创建 node-exporter DaemonSet
	if err := r.createNodeExporterDaemonSet(monitorNamespace); err != nil {
		return fmt.Errorf("create node-exporter daemonset: %v", err)
	}

	// 7. 创建 node-exporter Service
	if err := r.createNodeExporterService(monitorNamespace); err != nil {
		return fmt.Errorf("create node-exporter service: %v", err)
	}

	r.log.Info("Health-console and node-exporter monitoring resources created successfully")
	return nil
}

// createMonitorNamespace 创建监控命名空间
func (r *RainbondClusteMgr) createMonitorNamespace(namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"belongTo": "rainbond-operator",
				"creator":  "Rainbond",
			},
		},
	}

	if err := r.client.Create(r.ctx, ns); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("namespace already exists", "namespace", namespace)
			return nil
		}
		return err
	}

	r.log.Info("namespace created", "namespace", namespace)
	return nil
}

// createHealthConsoleConfigMap 创建 health-console ConfigMap
func (r *RainbondClusteMgr) createHealthConsoleConfigMap(namespace string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "health-console-config",
			Namespace: namespace,
			Labels: map[string]string{
				"belongTo": "rainbond-operator",
				"creator":  "Rainbond",
				"name":     "health-console",
			},
		},
		Data: map[string]string{
			"METRICS_PORT":     "9090",
			"COLLECT_INTERVAL": "30s",
			"IN_CLUSTER":       "true",
		},
	}

	if err := r.client.Create(r.ctx, cm); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("configmap already exists, updating", "name", cm.Name)
			return r.client.Update(r.ctx, cm)
		}
		return err
	}

	r.log.Info("configmap created", "name", cm.Name)
	return nil
}

// createHealthConsoleSecret 创建 health-console Secret，从实际配置获取信息
func (r *RainbondClusteMgr) createHealthConsoleSecret(namespace string) error {
	secretData := make(map[string]string)

	// 获取数据库配置 - Region Database
	if r.cluster.Spec.RegionDatabase != nil {
		secretData["DB_1_NAME"] = "rbd-db-region"
		secretData["DB_1_HOST"] = r.cluster.Spec.RegionDatabase.Host
		secretData["DB_1_PORT"] = strconv.Itoa(r.cluster.Spec.RegionDatabase.Port)
		secretData["DB_1_USER"] = r.cluster.Spec.RegionDatabase.Username
		secretData["DB_1_PASSWORD"] = r.cluster.Spec.RegionDatabase.Password
		secretData["DB_1_DATABASE"] = "region"
	} else {
		// 使用默认的 rbd-db 服务
		secretData["DB_1_NAME"] = "rbd-db-region"
		secretData["DB_1_HOST"] = "rbd-db-rw"
		secretData["DB_1_PORT"] = "3306"
		secretData["DB_1_USER"] = "root"
		// 尝试从 rbd-db secret 获取密码
		dbPassword, err := r.getDBPasswordFromSecret()
		if err != nil {
			r.log.Error(err, "failed to get db password from secret, using default")
			secretData["DB_1_PASSWORD"] = "21ce5b9f"
		} else {
			secretData["DB_1_PASSWORD"] = dbPassword
		}
		secretData["DB_1_DATABASE"] = "region"
	}

	// 获取数据库配置 - Console Database
	if r.cluster.Spec.UIDatabase != nil {
		secretData["DB_2_NAME"] = "rbd-db-console"
		secretData["DB_2_HOST"] = r.cluster.Spec.UIDatabase.Host
		secretData["DB_2_PORT"] = strconv.Itoa(r.cluster.Spec.UIDatabase.Port)
		secretData["DB_2_USER"] = r.cluster.Spec.UIDatabase.Username
		secretData["DB_2_PASSWORD"] = r.cluster.Spec.UIDatabase.Password
		secretData["DB_2_DATABASE"] = "console"
	} else {
		// 使用默认的 rbd-db 服务
		secretData["DB_2_NAME"] = "rbd-db-console"
		secretData["DB_2_HOST"] = "rbd-db-rw"
		secretData["DB_2_PORT"] = "3306"
		secretData["DB_2_USER"] = "root"
		dbPassword, err := r.getDBPasswordFromSecret()
		if err != nil {
			r.log.Error(err, "failed to get db password from secret, using default")
			secretData["DB_2_PASSWORD"] = "21ce5b9f"
		} else {
			secretData["DB_2_PASSWORD"] = dbPassword
		}
		secretData["DB_2_DATABASE"] = "console"
	}

	// 获取镜像仓库配置
	if r.cluster.Spec.ImageHub != nil {
		secretData["REGISTRY_1_NAME"] = "rbd-registry"
		secretData["REGISTRY_1_URL"] = r.cluster.Spec.ImageHub.Domain
		secretData["REGISTRY_1_USER"] = r.cluster.Spec.ImageHub.Username
		secretData["REGISTRY_1_PASSWORD"] = r.cluster.Spec.ImageHub.Password
		// 判断是否为不安全的 registry
		if strings.HasPrefix(r.cluster.Spec.ImageHub.Domain, "http://") ||
			!strings.Contains(r.cluster.Spec.ImageHub.Domain, ".") {
			secretData["REGISTRY_1_INSECURE"] = "true"
		} else {
			secretData["REGISTRY_1_INSECURE"] = "false"
		}
	} else {
		secretData["REGISTRY_1_NAME"] = "rbd-registry"
		secretData["REGISTRY_1_URL"] = "rbd-hub:5000"
		secretData["REGISTRY_1_USER"] = "admin"
		secretData["REGISTRY_1_PASSWORD"] = rbdutil.GetenvDefault("RBD_HUB_PASSWORD", "admin1234")
		secretData["REGISTRY_1_INSECURE"] = "true"
	}

	// MinIO 配置 - 可选
	secretData["MINIO_ENDPOINT"] = "minio-service:9000"
	secretData["MINIO_ACCESS_KEY"] = "admin"
	secretData["MINIO_SECRET_KEY"] = "admin1234"
	secretData["MINIO_USE_SSL"] = "false"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "health-console-secrets",
			Namespace: namespace,
			Labels: map[string]string{
				"belongTo": "rainbond-operator",
				"creator":  "Rainbond",
				"name":     "health-console",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	if err := r.client.Create(r.ctx, secret); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("secret already exists, updating", "name", secret.Name)
			return r.client.Update(r.ctx, secret)
		}
		return err
	}

	r.log.Info("secret created", "name", secret.Name)
	return nil
}

// getDBPasswordFromSecret 从 rbd-db secret 获取密码
func (r *RainbondClusteMgr) getDBPasswordFromSecret() (string, error) {
	secret := &corev1.Secret{}
	err := r.client.Get(r.ctx, types.NamespacedName{
		Namespace: r.cluster.Namespace,
		Name:      "rbd-db",
	}, secret)
	if err != nil {
		return "", err
	}

	// Try different possible password field names
	passwordFields := []string{"password", "mysql-password", "mysql-root-password"}
	for _, field := range passwordFields {
		if password, ok := secret.Data[field]; ok && len(password) > 0 {
			r.log.V(5).Info("found database password", "field", field)
			return string(password), nil
		}
	}

	return "", fmt.Errorf("no password field found in secret (tried: %v)", passwordFields)
}

// createHealthConsoleDeployment 创建 health-console Deployment
func (r *RainbondClusteMgr) createHealthConsoleDeployment(namespace string) error {
	labels := map[string]string{
		"belongTo": "rainbond-operator",
		"creator":  "Rainbond",
		"name":     "health-console",
	}

	replicas := int32(1)
	maxSurge := intstr.FromString("25%")
	maxUnavailable := intstr.FromString("25%")

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "health-console",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: func() *int32 { v := int32(600); return &v }(),
			Replicas:                &replicas,
			RevisionHistoryLimit:    func() *int32 { v := int32(10); return &v }(),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "health-console",
					Labels: labels,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9090",
						"prometheus.io/path":   "/metrics",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator",
					Containers: []corev1.Container{
						{
							Name:            "health-console",
							Image:           rbdutil.GetenvDefault("HEALTH_CONSOLE_IMAGE", "registry.cn-hangzhou.aliyuncs.com/zhangqihang/rainbond-health-console:122201"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9090,
									Name:          "metrics",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "status.podIP",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.name",
										},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "health-console-config",
										},
									},
								},
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "health-console-secrets",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: func() resource.Quantity { q, _ := resource.ParseQuantity("128Mi"); return q }(),
									corev1.ResourceCPU:    func() resource.Quantity { q, _ := resource.ParseQuantity("100m"); return q }(),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: func() resource.Quantity { q, _ := resource.ParseQuantity("256Mi"); return q }(),
									corev1.ResourceCPU:    func() resource.Quantity { q, _ := resource.ParseQuantity("200m"); return q }(),
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(9090),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       30,
								TimeoutSeconds:      10,
								FailureThreshold:    3,
								SuccessThreshold:    1,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(9090),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
								SuccessThreshold:    1,
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 "default-scheduler",
					SecurityContext:               &corev1.PodSecurityContext{},
					TerminationGracePeriodSeconds: func() *int64 { v := int64(30); return &v }(),
				},
			},
		},
	}

	if err := r.client.Create(r.ctx, deployment); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("deployment already exists, updating", "name", deployment.Name)
			return r.client.Update(r.ctx, deployment)
		}
		return err
	}

	r.log.Info("deployment created", "name", deployment.Name)
	return nil
}

// createHealthConsoleService 创建 health-console Service
func (r *RainbondClusteMgr) createHealthConsoleService(namespace string) error {
	labels := map[string]string{
		"belongTo": "rainbond-operator",
		"creator":  "Rainbond",
		"name":     "health-console",
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "health-console",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
					Name:       "metrics",
				},
			},
			Selector: labels,
		},
	}

	if err := r.client.Create(r.ctx, svc); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("service already exists, updating", "name", svc.Name)
			return r.client.Update(r.ctx, svc)
		}
		return err
	}

	r.log.Info("service created", "name", svc.Name)
	return nil
}

// createNodeExporterDaemonSet 创建 node-exporter DaemonSet
func (r *RainbondClusteMgr) createNodeExporterDaemonSet(namespace string) error {
	labels := map[string]string{
		"app.kubernetes.io/name":    "node-exporter",
		"app.kubernetes.io/version": "v1.7.0",
	}

	maxUnavailable := intstr.FromInt(1)
	hostPathType := corev1.HostPathUnset

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-exporter",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			RevisionHistoryLimit: func() *int32 { v := int32(10); return &v }(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "node-exporter",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					HostPID:     true,
					Containers: []corev1.Container{
						{
							Name:            "prometheus-node-exporter",
							Image:           rbdutil.GetenvDefault("NODE_EXPORTER_IMAGE", "registry.cn-hangzhou.aliyuncs.com/zhangqihang/node-exporter:v1.7.0"),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"--path.procfs=/host/proc",
								"--path.sysfs=/host/sys",
								"--path.rootfs=/host",
								"--web.listen-address=:9100",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 9100,
									HostPort:      9100,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    func() resource.Quantity { q, _ := resource.ParseQuantity("102m"); return q }(),
									corev1.ResourceMemory: func() resource.Quantity { q, _ := resource.ParseQuantity("30Mi"); return q }(),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    func() resource.Quantity { q, _ := resource.ParseQuantity("250m"); return q }(),
									corev1.ResourceMemory: func() resource.Quantity { q, _ := resource.ParseQuantity("50Mi"); return q }(),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "proc",
									MountPath: "/host/proc",
								},
								{
									Name:      "sys",
									MountPath: "/host/sys",
								},
								{
									Name:      "rootfs",
									MountPath: "/host",
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "proc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/proc",
									Type: &hostPathType,
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys",
									Type: &hostPathType,
								},
							},
						},
						{
							Name: "rootfs",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
									Type: &hostPathType,
								},
							},
						},
					},
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 "default-scheduler",
					SecurityContext:               &corev1.PodSecurityContext{},
					TerminationGracePeriodSeconds: func() *int64 { v := int64(30); return &v }(),
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
			},
		},
	}

	if err := r.client.Create(r.ctx, ds); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("daemonset already exists, updating", "name", ds.Name)
			return r.client.Update(r.ctx, ds)
		}
		return err
	}

	r.log.Info("daemonset created", "name", ds.Name)
	return nil
}

// createNodeExporterService 创建 node-exporter Service
func (r *RainbondClusteMgr) createNodeExporterService(namespace string) error {
	labels := map[string]string{
		"app.kubernetes.io/name":    "node-exporter",
		"app.kubernetes.io/version": "v1.0.0",
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-exporter",
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       9100,
					TargetPort: intstr.FromInt(9100),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": "node-exporter",
			},
		},
	}

	if err := r.client.Create(r.ctx, svc); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			r.log.V(5).Info("service already exists, updating", "name", svc.Name)
			return r.client.Update(r.ctx, svc)
		}
		return err
	}

	r.log.Info("service created", "name", svc.Name)
	return nil
}
