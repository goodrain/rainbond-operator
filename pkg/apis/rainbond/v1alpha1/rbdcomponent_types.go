package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogLevel -
type LogLevel string

const (
	// LogLevelDebug -
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo -
	LogLevelInfo LogLevel = "info"
	// LogLevelWarning -
	LogLevelWarning LogLevel = "warning"
	// LogLevelError -
	LogLevelError LogLevel = "error"
)

// PullPolicy describes a policy for if/when to pull a container image
type PullPolicy string

const (
	// PullAlways means that kubelet always attempts to pull the latest image. Container will fail If the pull fails.
	PullAlways PullPolicy = "Always"
	// PullNever means that kubelet never pulls an image, but only uses a local image. Container will fail if the image isn't present
	PullNever PullPolicy = "Never"
	// PullIfNotPresent means that kubelet pulls if the image isn't present on disk. Container will fail if the image isn't present and the pull fails.
	PullIfNotPresent PullPolicy = "IfNotPresent"
)

// RbdComponentSpec defines the desired state of RbdComponent
type RbdComponentSpec struct {
	// type of rainbond component
	Type string `json:"type,omitempty"`
	// version of rainbond component
	Version  string   `json:"version,omitempty"`
	LogLevel LogLevel `json:"logLevel,omitempty"`
	// Docker image name.
	Image string `json:"image,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	ImagePullPolicy PullPolicy `json:"imagePullPolicy,omitempty"`
}

// ControllerType -
type ControllerType string

const (
	// ControllerTypeDeployment -
	ControllerTypeDeployment ControllerType = "deployment"
	// ControllerTypeDaemonSet -
	ControllerTypeDaemonSet ControllerType = "daemonset"
	// ControllerTypeStatefulSet -
	ControllerTypeStatefulSet ControllerType = "statefuleset"
	// ControllerTypeUnknown -
	ControllerTypeUnknown ControllerType = "unknown"
)

func (c ControllerType) String() string {
	return string(c)
}

// RbdComponentStatus defines the observed state of RbdComponent
type RbdComponentStatus struct {
	// Type of Controller owned by RbdComponent
	ControllerType ControllerType `json:"controller_type"`
	// ControllerName represents the Controller associated with RbdComponent
	// The controller could be Deployment, StatefulSet or DaemonSet
	ControllerName string `json:"controller_name"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RbdComponent is the Schema for the rbdcomponents API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rbdcomponents,scope=Namespaced
type RbdComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RbdComponentSpec    `json:"spec,omitempty"`
	Status *RbdComponentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RbdComponentList contains a list of RbdComponent
type RbdComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RbdComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RbdComponent{}, &RbdComponentList{})
}

func (in *RbdComponent) Labels() map[string]string {
	return map[string]string{
		"creator": "Rainbond",
		"belongTo": "RainbondOperator",
		"name":    in.Name,
	}
}
