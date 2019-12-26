package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageProvisionerSpec defines the desired state of StorageProvisioner
type StorageProvisionerSpec struct {
	CustomProvisioner string `json:"custom_provisioner,omitempty"`
}

// ProvisionerPhase is the provisoner running phase
type ProvisionerPhase string

const (
	// ProvisionerPhaseNone -
	ProvisionerPhaseNone ProvisionerPhase = ""
	// ProvisionerPhaseCreating -
	ProvisionerPhaseCreating ProvisionerPhase = "Creating"
	// ProvisionerPhaseRunning -
	ProvisionerPhaseRunning ProvisionerPhase = "Running"
	// ProvisionerPhaseFailed -
	ProvisionerPhaseFailed ProvisionerPhase = "Failed"
)

// StorageProvisionerStatus defines the observed state of StorageProvisioner
type StorageProvisionerStatus struct {
	// Phase is the provisoner running phase
	Phase ProvisionerPhase `json:"phase"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageProvisioner is the Schema for the storageprovisioners API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=storageprovisioners,scope=Namespaced
type StorageProvisioner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageProvisionerSpec   `json:"spec,omitempty"`
	Status StorageProvisionerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageProvisionerList contains a list of StorageProvisioner
type StorageProvisionerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageProvisioner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageProvisioner{}, &StorageProvisionerList{})
}
