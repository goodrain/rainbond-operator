package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GlobalConfigPhase string

const (
	GlobalConfigPhasePending GlobalConfigPhase = "Pending"

	GlobalConfigPhaseExtractInstallationPackage GlobalConfigPhase = "ExtractInstallationPackage"

	GlobalConfigPhaseLoadImages GlobalConfigPhase = "LoadImages"

	GlobalConfigPhasePushImages GlobalConfigPhase = "PushImages"

	GlobalConfigPhaseInstalling GlobalConfigPhase = "Installing"

	GlobalConfigPhaseInstallationFinished GlobalConfigPhase = "InstallationFinished"
)

// GlobalConfigSpec defines the desired state of GlobalConfig
type GlobalConfigSpec struct {
	// default goodrain.me
	ImageHub ImageHub `json:"imageHub,omitempty"`
	// List of existing StorageClasses in the cluster
	AvailStorageClasses []string `json:"availStorageClasses,omitempty"`
	// the storage class that rainbond component will be used.
	// rainbond-operator will create one if StorageClassName is empty
	StorageClassName string `json:"storageClassName,omitempty"`
	// the region database information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	RegionDatabase Database `json:"regionDatabase,omitempty"`
	// the ui database information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	UIDatabase Database `json:"uiDatabase,omitempty"`
	// the etcd connection information that rainbond component will be used.
	// rainbond-operator will create one if EtcdConfig is empty
	EtcdConfig EtcdConfig `json:"etcdConfig,omitempty"`
	// KubeAPIHost must be a host string, a host:port pair, or a URL to the base of the apiserver.
	// If a URL is given then the (optional) Path of that URL represents a prefix that must
	// be appended to all request URIs used to access the apiserver. This allows a frontend
	// proxy to easily relocate all of the apiserver endpoints.
	KubeAPIHost string `json:"kubeAPIHost,omitempty"`

	NodeAvailPorts []NodeAvailPorts `json:"NodeAvailPorts,omitempty"`
}

type NodeAvailPorts struct {
	NodeName string `json:"nodeName,omitempty"`
	NodeIP   string `json:"nodeIP,omitempty"`
	Ports    []int  `json:"ports,omitempty"`
}

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
}

// EtcdConfig defines the configuration of etcd client.
type EtcdConfig struct {
	// Endpoints is a list of URLs.
	Endpoints []string `json:"endpoints,omitempty"`
	// Whether to use tls to connect to etcd
	UseTLS bool `json:"useTLS,omitempty"`
	// Secret to mount to read certificate files for tls.
	CertSecret metav1.LabelSelector `json:"selector,omitempty"`
}

// GlobalConfigStatus defines the observed state of GlobalConfig
type GlobalConfigStatus struct {
	// Rainbond cluster installation phase
	Phase GlobalConfigPhase `json:"phase,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalConfig is the Schema for the globalconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=globalconfigs,scope=Namespaced
type GlobalConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GlobalConfigSpec   `json:"spec,omitempty"`
	Status GlobalConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalConfigList contains a list of GlobalConfig
type GlobalConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GlobalConfig{}, &GlobalConfigList{})
}
