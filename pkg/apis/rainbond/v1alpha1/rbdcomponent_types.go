package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RbdComponentType is the type of rainbond component
type RbdComponentType string

const (
	// RbdComponentTypeAPI rbd-api
	RbdComponentTypeAPI RbdComponentType = "rbd-api"
	// RbdComponentTypeWorker rbd-worker
	RbdComponentTypeWorker RbdComponentType = "rbd-worker"
)

// RbdComponentPhase is the phase of rainbond component
type RbdComponentPhase string

const (
	// RbdComponentmPhaseImagePulling -
	RbdComponentmPhaseImagePulling RbdComponentPhase = "ImagePulling"
	// RbdComponentPhaseCreating -
	RbdComponentPhaseCreating RbdComponentPhase = "Creating"
	// RbdComponentPhaseRunning -
	RbdComponentPhaseRunning RbdComponentPhase = "Running"
	// RbdComponentPhasePailed -
	RbdComponentPhasePailed RbdComponentPhase = "Failed"
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

// RbdComponentSpec defines the desired state of RbdComponent
type RbdComponentSpec struct {
	// type of rainbond component
	Type string `json:"type"`
	// version of rainbond component
	Version  string   `json:"version"`
	LogLevel LogLevel `json:"logLevel,omitempty"`
}

// RbdComponentStatus defines the observed state of RbdComponent
type RbdComponentStatus struct {
	Phase     RbdComponentPhase `json:"phase,omitempty"`
	PodStatus PodStatus         `json:"podStatus"`
	Message   string            `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RbdComponent is the Schema for the rbdcomponents API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rbdcomponents,scope=Namespaced
type RbdComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RbdComponentSpec   `json:"spec"`
	Status RbdComponentStatus `json:"status"`
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

// PodStatus -
type PodStatus struct {
	// Ready are the component pods that are ready to serve requests
	// The pod names are the same as the component pod names
	Ready []string `json:"ready,omitempty"`
	// Unready are the components not ready to serve requests
	Unready []string `json:"unready,omitempty"`
	// Healthy are the component pods that pass the liveness.
	Healthy []string `json:"healthy,omitempty"`
	// Healthy are the component pods that pass the de liveness.
	UnHealthy []string `json:"unHealthy,omitempty"`
}
