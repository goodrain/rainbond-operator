package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GlobalConfigSpec defines the desired state of GlobalConfig
type GlobalConfigSpec struct {
	// default goodrain.me
	ImageRepositry string `json:"imageRepositry,omitempty"`
	// the storage class that rainbond component will be used.
	// rainbond-operator will create one if StorageClassName is empty
	StorageClassName string `json:"storageClassName,omitempty"`
	// the db connection information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	DBInfo DBInfo `json:"dbInfo,omitempty"`
	// the etcd connection information that rainbond component will be used.
	// rainbond-operator will create one if EtcdInfo is empty
	EtcdInfo EtcdInfo `json:"etcdInfo,omitempty"`
}

// DBInfo defines the connection information of database.
type DBInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// EtcdInfo defines the connection information of ETCD.
type EtcdInfo struct {
	Endpoints []string `json:"endpoints"`
}

// GlobalConfigStatus defines the observed state of GlobalConfig
type GlobalConfigStatus struct {
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
