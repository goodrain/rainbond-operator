/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallMode is the mode of Wutong cluster installation
type InstallMode string

const (
	// InstallationModeWithoutPackage means all Wutong images are from the specified image repository, but still needs wutong package.
	InstallationModeWithoutPackage InstallMode = "Online"
	// InstallationModeFullOnline means all Wutong images are from the specified image repository.
	InstallationModeFullOnline InstallMode = "FullOnline"
	// InstallationModeOffline install by local resource
	InstallationModeOffline InstallMode = "Offline"

	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"
	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"
)

// WutongClusterConditionType is the type of wutongclsuter condition.
type WutongClusterConditionType string

// These are valid conditions of WutongCluster.
const (
	WutongClusterConditionTypeDatabaseRegion    = "DatabaseRegion"
	WutongClusterConditionTypeDatabaseConsole   = "DatabaseConsole"
	WutongClusterConditionTypeImageRepository   = "ImageRepository"
	WutongClusterConditionTypeKubernetesVersion = "KubernetesVersion"
	WutongClusterConditionTypeKubernetesStatus  = "KubernetesStatus"
	WutongClusterConditionTypeStorage           = "Storage"
	WutongClusterConditionTypeDNS               = "DNS"
	WutongClusterConditionTypeContainerNetwork  = "ContainerNetwork"
	WutongClusterConditionTypeRunning           = "Running"
	WutongClusterConditionTypeMemory            = "Memory"
)

// WutongClusterCondition contains condition information for WutongCluster.
type WutongClusterCondition struct {
	// Type of wutongclsuter condition.
	Type WutongClusterConditionType `json:"type" `
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status" `
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty" `
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" `
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// ImageHub image hub
type ImageHub struct {
	Domain    string `json:"domain,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
}

// Database defines the connection information of database.
type Database struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}

// EtcdConfig defines the configuration of etcd client.
type EtcdConfig struct {
	// Endpoints is a list of URLs.
	Endpoints []string `json:"endpoints,omitempty"`
	// Whether to use tls to connect to etcd
	SecretName string `json:"secretName,omitempty"`
}

// WutongClusterSpec defines the desired state of WutongCluster
type WutongClusterSpec struct {
	// EnableHA is a highly available switch.
	EnableHA bool `json:"enableHA,omitempty"`
	// Repository of each Wutong component image, eg. docker.io/wutong.
	// +optional
	WutongImageRepository string `json:"wutongImageRepository,omitempty"`
	// Suffix of component default domain name
	SuffixHTTPHost string `json:"suffixHTTPHost"`
	// Ingress IP addresses of wt-gateway. If not specified,
	// the IP of the node where the wt-gateway is located will be used.
	GatewayIngressIPs []string `json:"gatewayIngressIPs,omitempty"`
	// Specify the nodes where the wt-gateway will running.
	NodesForGateway []*K8sNode `json:"nodesForGateway,omitempty"`
	// Specify the nodes where the wt-gateway will running.
	NodesForChaos []*K8sNode `json:"nodesForChaos,omitempty"`
	// InstallMode is the mode of Wutong cluster installation.
	InstallMode InstallMode `json:"installMode,omitempty"`
	// User-specified private image repository, replacing wutong.me.
	ImageHub *ImageHub `json:"imageHub,omitempty"`
	// the region database information that wutong component will be used.
	// wutong-operator will create one if DBInfo is empty
	RegionDatabase *Database `json:"regionDatabase,omitempty"`
	// the ui database information that wutong component will be used.
	// wutong-operator will create one if DBInfo is empty
	UIDatabase *Database `json:"uiDatabase,omitempty"`
	// the etcd connection information that wutong component will be used.
	// wutong-operator will create one if EtcdConfig is empty
	EtcdConfig *EtcdConfig `json:"etcdConfig,omitempty"`
	// define install wutong version, This is usually image tag
	InstallVersion string `json:"installVersion,omitempty"`
	// CIVersion define builder and runner version
	CIVersion string `json:"ciVersion,omitempty"`
	// Whether the configuration has been completed
	ConfigCompleted bool `json:"configCompleted,omitempty"`

	WutongVolumeSpecRWX *WutongVolumeSpec `json:"wutongVolumeSpecRWX,omitempty"`
	WutongVolumeSpecRWO *WutongVolumeSpec `json:"wutongVolumeSpecRWO,omitempty"`

	// SentinelImage is the image for wutong operator sentinel
	SentinelImage string `json:"sentinelImage,omitempty"`

	CacheMode string `json:"cacheMode,omitempty"`
}

//InstallPackageConfig define install package download config
type InstallPackageConfig struct {
	URL string `json:"url,omitempty"`
	MD5 string `json:"md5,omitempty"`
}

// StorageClass storage class
type StorageClass struct {
	Name        string                            `json:"name"`
	Provisioner string                            `json:"provisioner"`
	AccessMode  corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty"`
}

// K8sNode holds the information about a kubernetes node.
type K8sNode struct {
	Name       string `json:"name,omitempty"`
	InternalIP string `json:"internalIP,omitempty"`
	ExternalIP string `json:"externalIP,omitempty"`
}

// AvailableNodes contains nodes available for special wutong components to run,
// such as wt-gateway, wt-chaos.
type AvailableNodes struct {
	// The nodes with user-specified labels.
	SpecifiedNodes []*K8sNode `json:"specifiedNodes,omitempty"`
	// A list of kubernetes master nodes.
	MasterNodes []*K8sNode `json:"masterNodes,omitempty"`
}

// WutongClusterStatus defines the observed state of WutongCluster
type WutongClusterStatus struct {
	// Versoin of Kubernetes
	KubernetesVersoin string `json:"kubernetesVersoin,omitempty"`
	// List of existing StorageClasses in the cluster
	// +optional
	StorageClasses []*StorageClass `json:"storageClasses,omitempty"`
	// Destination path of the installation package extraction.
	MasterRoleLabel string `json:"masterRoleLabel,omitempty"`
	// holds some recommend nodes available for wt-gateway to run.
	GatewayAvailableNodes *AvailableNodes `json:"gatewayAvailableNodes,omitempty"`
	// holds some recommend nodes available for wt-chaos to run.
	ChaosAvailableNodes *AvailableNodes `json:"chaosAvailableNodes,omitempty"`
	// Deprecated. ImagePullUsername is the username to pull any of images used by PodSpec
	ImagePullUsername string `json:"imagePullUsername,omitempty"`
	// Deprecated. ImagePullPassword is the password to pull any of images used by PodSpec
	ImagePullPassword string `json:"imagePullPassword,omitempty"`
	// ImagePullSecret is an optional references to secret in the same namespace to use for pulling any of the images used by PodSpec.
	ImagePullSecret *corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	Conditions []WutongClusterCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WutongCluster is the Schema for the WutongClusters API
type WutongCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WutongClusterSpec   `json:"spec,omitempty"`
	Status WutongClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WutongClusterList contains a list of WutongCluster
type WutongClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WutongCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WutongCluster{}, &WutongClusterList{})
}

// InnerGatewayIngressIP -
func (in *WutongCluster) InnerGatewayIngressIP() string {
	if len(in.Spec.NodesForGateway) > 0 {
		return in.Spec.NodesForGateway[0].InternalIP
	}
	if len(in.Spec.GatewayIngressIPs) > 0 && in.Spec.GatewayIngressIPs[0] != "" {
		return in.Spec.GatewayIngressIPs[0]
	}
	return ""
}

// GatewayIngressIP returns the gateway ip, or take the internal ip
// of the first node for gateway if it's not exists.
func (in *WutongCluster) GatewayIngressIP() string {
	if len(in.Spec.GatewayIngressIPs) > 0 && in.Spec.GatewayIngressIPs[0] != "" {
		return in.Spec.GatewayIngressIPs[0]
	}
	if len(in.Spec.NodesForGateway) > 0 {
		return in.Spec.NodesForGateway[0].InternalIP
	}
	return ""
}

//GatewayIngressIPs get all gateway ips
func (in *WutongCluster) GatewayIngressIPs() (ips []string) {
	// custom ip ,contain eip
	if len(in.Spec.GatewayIngressIPs) > 0 && in.Spec.GatewayIngressIPs[0] != "" {
		return in.Spec.GatewayIngressIPs
	}
	// user select gateway node ip
	if len(in.Spec.NodesForGateway) > 0 {
		for _, node := range in.Spec.NodesForGateway {
			ips = append(ips, node.InternalIP)
		}
		return
	}
	return nil
}

// RegionDataSource returns the data source for database region.
func (in *Database) RegionDataSource() string {
	return fmt.Sprintf("--mysql=%s:%s@tcp(%s:%d)/%s", in.Username, in.Password, in.Host, in.Port, in.Name)
}

// NewWutongClusterCondition creates a new rianbondcluster condition.
func NewWutongClusterCondition(condType WutongClusterConditionType, status v1.ConditionStatus, reason, message string) *WutongClusterCondition {
	return &WutongClusterCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// SetCondition setups the given WutongCluster condition.
func (r *WutongClusterStatus) SetCondition(c WutongClusterCondition) {
	pos, cp := r.GetCondition(c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		return
	}

	if cp != nil {
		r.Conditions[pos] = c
	} else {
		r.Conditions = append(r.Conditions, c)
	}
}

// GetCondition returns a WutongComponent condition based on the given type.
func (r *WutongClusterStatus) GetCondition(t WutongClusterConditionType) (int, *WutongClusterCondition) {
	for i, c := range r.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

// UpdateCondition updates existing WutongComponent condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if WutongComponent condition has changed or has been added.
func (r *WutongClusterStatus) UpdateCondition(condition *WutongClusterCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this WutongVolume condition.
	conditionIndex, oldCondition := r.GetCondition(condition.Type)

	if oldCondition == nil {
		// We are adding new WutongVolume condition.
		r.Conditions = append(r.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	r.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

//DeleteCondition -
func (r *WutongClusterStatus) DeleteCondition(typ3 WutongClusterConditionType) {
	idx, _ := r.GetCondition(typ3)
	if idx == -1 {
		return
	}
	r.Conditions = append(r.Conditions[:idx], r.Conditions[idx+1:]...)
}
