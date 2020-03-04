package rainbondcluster

import (
	"context"
	"fmt"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var provisionerAccessModes map[string]corev1.PersistentVolumeAccessMode = map[string]corev1.PersistentVolumeAccessMode{
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
	"diskplugin.csi.alibabacloud.com": corev1.ReadWriteMany,
	"lvmplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
	"memplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
	"nasplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
	"ossplugin.csi.alibabacloud.com":  corev1.ReadWriteMany,
}

type rainbondClusteMgr struct {
	ctx    context.Context
	client client.Client
	log    logr.Logger

	cluster *rainbondv1alpha1.RainbondCluster
}

func newRbdcomponentMgr(ctx context.Context, client client.Client, log logr.Logger, cluster *rainbondv1alpha1.RainbondCluster) *rainbondClusteMgr {
	mgr := &rainbondClusteMgr{
		ctx:     ctx,
		client:  client,
		log:     log,
		cluster: cluster,
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
