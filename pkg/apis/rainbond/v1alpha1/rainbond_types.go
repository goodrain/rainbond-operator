package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RainbondSpec defines the desired state of Rainbond
type RainbondSpec struct {
	// Rainbond's version
	Version string `json:"versoin,omitempty"`
}

// RainbondPhase is the Rainbond running phase
type RainbondPhase string

const (
	// RainbondPhaseNone -
	RainbondPhaseNone RainbondPhase = ""
	// RainbondPhaseCreating -
	RainbondPhaseCreating RainbondPhase = "Creating"
	// RainbondPhaseRunning -
	RainbondPhaseRunning RainbondPhase = "Running"
	// RainbondPhaseFailed -
	RainbondPhaseFailed RainbondPhase = "Failed"
)

// RainbondStatus defines the observed state of Rainbond
type RainbondStatus struct {
	// Phase is the rainbond running phase
	Phase RainbondPhase `json:"phase"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Rainbond is the Schema for the rainbonds API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rainbonds,scope=Namespaced
type Rainbond struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RainbondSpec   `json:"spec,omitempty"`
	Status RainbondStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RainbondList contains a list of Rainbond
type RainbondList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rainbond `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Rainbond{}, &RainbondList{})
}
