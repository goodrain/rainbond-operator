package rainbondcluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"

	"github.com/go-logr/logr"
	"github.com/pquerna/ffjson/ffjson"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

type rainbondClusteMgr struct {
	ctx    context.Context
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger

	cluster *rainbondv1alpha1.RainbondCluster
}

func newRbdcomponentMgr(ctx context.Context, client client.Client, log logr.Logger, cluster *rainbondv1alpha1.RainbondCluster, scheme *runtime.Scheme) *rainbondClusteMgr {
	mgr := &rainbondClusteMgr{
		ctx:     ctx,
		client:  client,
		log:     log,
		cluster: cluster,
		scheme:  scheme,
	}
	return mgr
}

func (r *rainbondClusteMgr) listStorageClasses() []*rainbondv1alpha1.StorageClass {
	r.log.V(6).Info("start listing available storage classes")

	storageClassList := &storagev1.StorageClassList{}
	var opts []client.ListOption
	if err := r.client.List(r.ctx, storageClassList, opts...); err != nil {
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

	return storageClasses
}

// generateRainbondClusterStatus creates the final rainbondcluster status for a rainbondcluster, given the
// internal rainbondcluster status.
func (r *rainbondClusteMgr) generateRainbondClusterStatus() (*rainbondv1alpha1.RainbondClusterStatus, error) {
	r.log.V(6).Info("start generating status")

	masterRoleLabel, err := r.getMasterRoleLabel()
	if err != nil {
		return nil, fmt.Errorf("get master role label: %v", err)
	}

	s := &rainbondv1alpha1.RainbondClusterStatus{
		MasterRoleLabel: masterRoleLabel,
		StorageClasses:  r.listStorageClasses(),
	}
	if r.cluster.Status != nil {
		if r.cluster.Status.ImagePullUsername == "" || r.cluster.Status.ImagePullPassword == "" {
			s.ImagePullUsername = "admin"
			s.ImagePullPassword = uuidutil.NewUUID()[0:8]
		} else {
			s.ImagePullUsername = r.cluster.Status.ImagePullUsername
			s.ImagePullPassword = r.cluster.Status.ImagePullPassword
		}
	}

	if r.checkIfImagePullSecretExists() {
		s.ImagePullSecret = corev1.LocalObjectReference{Name: RdbHubCredentialsName}
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

	k8sVersion, err := r.getKubernetesInfo()
	if err != nil {
		return nil, err
	}
	s.KubernetesVersoin = k8sVersion

	return s, nil
}

func (r *rainbondClusteMgr) getMasterRoleLabel() (string, error) {
	nodes := &corev1.NodeList{}
	if err := r.client.List(r.ctx, nodes); err != nil {
		log.Error(err, "list nodes: %v", err)
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

func (r *rainbondClusteMgr) listSpecifiedGatewayNodes() []*rainbondv1alpha1.K8sNode {
	nodes := r.listNodesByLabels(map[string]string{
		constants.SpecialGatewayLabelKey: "",
	})
	// Filtering nodes with port conflicts
	// check gateway ports
	return rbdutil.FilterNodesWithPortConflicts(nodes)
}

func (r *rainbondClusteMgr) listSpecifiedChaosNodes() []*rainbondv1alpha1.K8sNode {
	return r.listNodesByLabels(map[string]string{
		constants.SpecialChaosLabelKey: "",
	})
}

func (r *rainbondClusteMgr) listNodesByLabels(labels map[string]string) []*rainbondv1alpha1.K8sNode {
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(labels),
	}
	if err := r.client.List(r.ctx, nodeList, listOpts...); err != nil {
		log.Error(err, "list nodes")
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

func (r *rainbondClusteMgr) listMasterNodesForGateway(masterLabel string) []*rainbondv1alpha1.K8sNode {
	nodes := r.listMasterNodes(masterLabel)
	// Filtering nodes with port conflicts
	// check gateway ports
	return rbdutil.FilterNodesWithPortConflicts(nodes)
}

func (r *rainbondClusteMgr) listMasterNodes(masterRoleLabelKey string) []*rainbondv1alpha1.K8sNode {
	labels := k8sutil.MaterRoleLabel(masterRoleLabelKey)
	return r.listNodesByLabels(labels)
}

func (r *rainbondClusteMgr) createImagePullSecret() error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RdbHubCredentialsName,
			Namespace: r.cluster.Namespace,
		},
		Data: map[string][]byte{
			".dockerconfigjson": r.generateDockerConfig(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	if err := controllerutil.SetControllerReference(r.cluster, secret, r.scheme); err != nil {
		return fmt.Errorf("set controller reference for secret %s: %v", RdbHubCredentialsName, err)
	}

	err := r.client.Create(r.ctx, secret)
	if err != nil && !k8sErrors.IsAlreadyExists(err) {
		return fmt.Errorf("create secret for pulling images: %v", err)
	}

	return nil
}

func (r *rainbondClusteMgr) checkIfImagePullSecretExists() bool {
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

func (r *rainbondClusteMgr) generateDockerConfig() []byte {
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

func (r *rainbondClusteMgr) getKubernetesInfo() (string, error) {
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{}
	if err := r.client.List(r.ctx, nodeList, listOpts...); err != nil {
		log.Error(err, "list nodes")
		return "", fmt.Errorf("list nodes: %v", err)
	}

	var version string
	for _, node := range nodeList.Items {
		if node.Status.NodeInfo.KubeletVersion == "" {
			continue
		}
		version = node.Status.NodeInfo.KubeletVersion
		break
	}

	if version == "" {
		return "", fmt.Errorf("failed to get kubernetes version")
	}

	return version, nil
}

func (r *rainbondClusteMgr) checkIfRbdNodeReady() error {
	cpt := &rainbondv1alpha1.RbdComponent{}
	if err := r.client.Get(r.ctx, types.NamespacedName{Namespace: r.cluster.Namespace, Name: "rbd-node"}, cpt); err != nil {
		return err
	}

	if cpt.Status == nil {
		return fmt.Errorf("no status for rbdcomponent rbd-node")
	}

	if cpt.Status.ReadyReplicas == 0 || cpt.Status.ReadyReplicas != cpt.Status.Replicas {
		return fmt.Errorf("no ready replicas for rbdcomponent rbd-node")
	}

	return nil
}
