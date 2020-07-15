package v1alpha1

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallMode is the mode of Rainbond cluster installation
type InstallMode string

const (
	// InstallationModeWithPackage means some Rainbond images are from the specified image repository, some are from the installation package.
	InstallationModeWithPackage InstallMode = "WithPackage"
	// InstallationModeWithoutPackage means all Rainbond images are from the specified image repository, but still needs rainbond package.
	InstallationModeWithoutPackage InstallMode = "WithoutPackage"
	// InstallationModeFullOnline means all Rainbond images are from the specified image repository.
	InstallationModeFullOnline InstallMode = "FullOnline"

	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"
	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"
)

// RainbondClusterConditionType is the type of rainbondclsuter condition.
type RainbondClusterConditionType string

// These are valid conditions of rainbondcluster.
const (
	RainbondClusterConditionTypeDatabaseRegion  = "DatabaseRegion"
	RainbondClusterConditionTypeDatabaseConsole = "DatabaseConsole"
	RainbondClusterConditionTypeImageRepository = "imagerepository"
)

// RainbondClusterCondition contains condition information for rainbondcluster.
type RainbondClusterCondition struct {
	// Type of rainbondclsuter condition.
	Type RainbondClusterConditionType `json:"type" `
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

// RainbondClusterSpec defines the desired state of RainbondCluster
type RainbondClusterSpec struct {
	// EnableHA is a highly available switch.
	EnableHA bool `json:"enableHA"`
	// Repository of each Rainbond component image, eg. docker.io/rainbond.
	// +optional
	RainbondImageRepository string `json:"rainbondImageRepository,omitempty"`
	// Suffix of component default domain name
	SuffixHTTPHost string `json:"suffixHTTPHost"`
	// Ingress IP addresses of rbd-gateway. If not specified,
	// the IP of the node where the rbd-gateway is located will be used.
	GatewayIngressIPs []string `json:"gatewayIngressIPs,omitempty"`
	// Specify the nodes where the rbd-gateway will running.
	NodesForGateway []*K8sNode `json:"nodesForGateway,omitempty"`
	// Specify the nodes where the rbd-gateway will running.
	NodesForChaos []*K8sNode `json:"nodesForChaos,omitempty"`
	// InstallMode is the mode of Rainbond cluster installation.
	InstallMode InstallMode `json:"installMode,omitempty"`
	// User-specified private image repository, replacing goodrain.me.
	ImageHub *ImageHub `json:"imageHub,omitempty"`
	// the storage class that rainbond component will be used.
	// rainbond-operator will create one if StorageClassName is empty
	StorageClassName string `json:"storageClassName,omitempty"`
	// the region database information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	RegionDatabase *Database `json:"regionDatabase,omitempty"`
	// the ui database information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	UIDatabase *Database `json:"uiDatabase,omitempty"`
	// the etcd connection information that rainbond component will be used.
	// rainbond-operator will create one if EtcdConfig is empty
	EtcdConfig *EtcdConfig `json:"etcdConfig,omitempty"`
	// define install rainbond version, This is usually image tag
	InstallVersion string `json:"installVersion,omitempty"`
	// Whether the configuration has been completed
	ConfigCompleted bool `json:"configCompleted,omitempty"`
	//InstallPackageConfig define install package download config
	InstallPackageConfig InstallPackageConfig `json:"installPackageConfig,omitempty"`
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

// AvailableNodes contains nodes available for special rainbond components to run,
// such as rbd-gateway, rbd-chaos.
type AvailableNodes struct {
	// The nodes with user-specified labels.
	SpecifiedNodes []*K8sNode `json:"specifiedNodes,omitempty"`
	// A list of kubernetes master nodes.
	MasterNodes []*K8sNode `json:"masterNodes,omitempty"`
}

// RainbondClusterStatus defines the observed state of RainbondCluster
type RainbondClusterStatus struct {
	// Versoin of Kubernetes
	KubernetesVersoin string `json:"kubernetesVersoin,omitempty"`
	// List of existing StorageClasses in the cluster
	// +optional
	StorageClasses []*StorageClass `json:"storageClasses,omitempty"`
	// Destination path of the installation package extraction.
	MasterRoleLabel string `json:"masterRoleLabel,omitempty"`
	// holds some recommend nodes available for rbd-gateway to run.
	GatewayAvailableNodes *AvailableNodes `json:"gatewayAvailableNodes,omitempty"`
	// holds some recommend nodes available for rbd-chaos to run.
	ChaosAvailableNodes *AvailableNodes `json:"chaosAvailableNodes,omitempty"`
	// ImagePullUsername is the username to pull any of images used by PodSpec
	ImagePullUsername string `json:"imagePullUsername,omitempty"`
	// ImagePullPassword is the password to pull any of images used by PodSpec
	ImagePullPassword string `json:"imagePullPassword,omitempty"`
	// ImagePullSecret is an optional references to secret in the same namespace to use for pulling any of the images used by PodSpec.
	ImagePullSecret corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RainbondCluster is the Schema for the rainbondclusters API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rainbondclusters,scope=Namespaced
type RainbondCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RainbondClusterSpec    `json:"spec,omitempty"`
	Status *RainbondClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RainbondClusterList contains a list of RainbondCluster
type RainbondClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RainbondCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RainbondCluster{}, &RainbondClusterList{})
}

// GatewayIngressIP returns the gateway ip, or take the internal ip
// of the first node for gateway if it's not exists.
func (in *RainbondCluster) GatewayIngressIP() string {
	if len(in.Spec.GatewayIngressIPs) > 0 && in.Spec.GatewayIngressIPs[0] != "" {
		return in.Spec.GatewayIngressIPs[0]
	}
	if len(in.Spec.NodesForGateway) > 0 {
		return in.Spec.NodesForGateway[0].InternalIP
	}
	return ""
}

//GatewayIngressIPs get all gateway ips
func (in *RainbondCluster) GatewayIngressIPs() (ips []string) {
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
	name := os.Getenv("REGION_DB_NAME")
	if name == "" {
		name = "region"
	}
	return fmt.Sprintf("--mysql=%s:%s@tcp(%s:%d)/%s", in.Username, in.Password, in.Host, in.Port, name)
}
