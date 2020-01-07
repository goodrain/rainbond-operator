package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeAvailPorts struct {
	NodeName string `json:"nodeName,omitempty"`
	NodeIP   string `json:"nodeIP,omitempty"`
	Ports    []int  `json:"ports,omitempty"`
}

// RainbondClusterSpec defines the desired state of RainbondCluster
type RainbondClusterSpec struct {
	NodeAvailPorts []NodeAvailPorts `json:"NodeAvailPorts,omitempty"`

	// List of existing StorageClasses in the cluster
	AvailableStorageClasses []string `json:"availableStorageClasses,omitempty"`
}

// RainbondClusterPhase is a label for the condition of a rainbondcluster at the current time.
type RainbondClusterPhase string

// These are the valid statuses of rainbondcluster.
const (
	// RainbondClusterPending means the rainbondcluster has been accepted by the system, but one or more of the rbdcomponent
	// has not been started.
	RainbondClusterPending RainbondClusterPhase = "Pending"
	// RainbondClusterInstalling means the rainbond cluster is in installation.
	RainbondClusterInstalling RainbondClusterPhase = "Installing"
	// RainbondClusterRunning means all of the rainbond components has been created.
	RainbondClusterRunning RainbondClusterPhase = "Running"
)

// RainbondClusterConditionType is a valid value for RainbondClusterConditionType.Type
type RainbondClusterConditionType string

// These are valid conditions of rainbondcluster.
const (
	// StorageReady indicates whether the storage is ready.
	StorageReady RainbondClusterConditionType = "StorageReady"
	// ImageRepositoryReady indicates whether the image repository is ready.
	ImageRepositoryInstalled RainbondClusterConditionType = "ImageRepositoryInstalled"
	// PackageExtracted indicates whether the installation package has been decompressed.
	PackageExtracted RainbondClusterConditionType = "PackageExtracted"
	// ImageLoaded means that all images from the installation package has been loaded successfully.
	ImageLoaded RainbondClusterConditionType = "ImageLoaded"
	// ImageLoaded means that all images from the installation package has been pushed successfully.
	ImagePushed RainbondClusterConditionType = "ImagePushed"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means rainbond operator
// can't decide if a resource is in the condition or not.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// RainbondClusterCondition contains details for the current condition of this rainbondcluster.
type RainbondClusterCondition struct {
	// Type is the type of the condition.
	Type RainbondClusterConditionType `json:"type"`
	// Status is the status of the condition.
	Status ConditionStatus `json:"status"`
	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// RainbondClusterStatus defines the observed state of RainbondCluster
type RainbondClusterStatus struct {
	// Rainbond cluster phase
	Phase      RainbondClusterPhase       `json:"phase,omitempty"`
	Conditions []RainbondClusterCondition `json:"conditions,omitempty"`
	// A human readable message indicating details about why the pod is in this condition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// A brief CamelCase message indicating details about why the pod is in this state.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
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
