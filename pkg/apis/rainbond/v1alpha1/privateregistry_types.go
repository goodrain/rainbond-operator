package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrivateRegistrySpec defines the desired state of PrivateRegistry
type PrivateRegistrySpec struct {
	CustomPrivateRegistry string `json:"custom_private_registry,omitempty"`
}

// PrivateRegistryPhase is the PrivateRegistry running phase
type PrivateRegistryPhase string

const (
	// PrivateRegistryPhaseNone -
	PrivateRegistryPhaseNone PrivateRegistryPhase = ""
	// PrivateRegistryPhaseCreating -
	PrivateRegistryPhaseCreating PrivateRegistryPhase = "Creating"
	// PrivateRegistryPhaseRunning -
	PrivateRegistryPhaseRunning PrivateRegistryPhase = "Running"
	// PrivateRegistryPhaseFailed -
	PrivateRegistryPhaseFailed PrivateRegistryPhase = "Failed"
)

// PrivateRegistryStatus defines the observed state of PrivateRegistry
type PrivateRegistryStatus struct {
	// Phase is the provisoner running phase
	Phase PrivateRegistryPhase `json:"phase"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PrivateRegistry is the Schema for the privateregistries API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=privateregistries,scope=Namespaced
type PrivateRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrivateRegistrySpec   `json:"spec,omitempty"`
	Status PrivateRegistryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PrivateRegistryList contains a list of PrivateRegistry
type PrivateRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrivateRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrivateRegistry{}, &PrivateRegistryList{})
}
