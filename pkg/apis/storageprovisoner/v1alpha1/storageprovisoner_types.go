package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageProvisonerSpec defines the desired state of StorageProvisoner
type StorageProvisonerSpec struct {
	CustomProvisioner string `json:"custom_provisioner,omitempty"`
}

// ProvisonerPhase is the provisoner running phase
type ProvisonerPhase string

const (
	// ProvisonerPhaseNone -
	ProvisonerPhaseNone ProvisonerPhase = ""
	// ProvisonerPhaseCreating -
	ProvisonerPhaseCreating ProvisonerPhase = "Creating"
	// ProvisonerPhaseRunning -
	ProvisonerPhaseRunning ProvisonerPhase = "Running"
	// ProvisonerPhaseFailed -
	ProvisonerPhaseFailed ProvisonerPhase = "Failed"
)

// StorageProvisonerStatus defines the observed state of StorageProvisoner
type StorageProvisonerStatus struct {
	// Phase is the provisoner running phase
	Phase ProvisonerPhase `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageProvisoner is the Schema for the storageprovisoners API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=storageprovisoners,scope=Namespaced
type StorageProvisoner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageProvisonerSpec   `json:"spec,omitempty"`
	Status StorageProvisonerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageProvisonerList contains a list of StorageProvisoner
type StorageProvisonerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageProvisoner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageProvisoner{}, &StorageProvisonerList{})
}
