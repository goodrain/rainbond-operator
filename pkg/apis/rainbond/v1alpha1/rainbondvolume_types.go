package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AliyunCloudDiskCSIPluginSource represents a aliyun cloud disk CSI plugin.
// More info: https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/master/docs/disk.md
type AliyunCloudDiskCSIPluginSource struct {
	// The AccessKey ID provided by Alibaba Cloud for access control.
	AccessKeyID string `json:"accessKeyID"`
	// The AccessKey Secret provided by Alibaba Cloud for access control
	AccessKeySecret string `json:"accessKeySecret"`
}

// AliyunNasCSIPluginSource represents a aliyun cloud nas CSI plugin.
// More info: https://github.com/GLYASAI/alibaba-cloud-csi-driver/blob/master/docs/nas.md
type AliyunNasCSIPluginSource struct {
	// The AccessKey ID provided by Alibaba Cloud for access control.
	AccessKeyID string `json:"accessKeyID"`
	// The AccessKey Secret provided by Alibaba Cloud for access control
	AccessKeySecret string `json:"accessKeySecret"`
}

// CSIPluginSource represents the source of a csi driver to create.
// Only one of its members may be specified.
type CSIPluginSource struct {
	// AliyunCloudDiskCSIPluginSource represents a aliyun cloud disk CSI plugin.
	// More info: https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/master/docs/disk.md
	AliyunCloudDisk *AliyunCloudDiskCSIPluginSource `json:"aliyunCloudDisk,omitempty"`
	// AliyunNasCSIPluginSource represents a aliyun cloud nas CSI plugin.
	// More info: https://github.com/GLYASAI/alibaba-cloud-csi-driver/blob/master/docs/nas.md
	AliyunNas *AliyunNasCSIPluginSource `json:"aliyunNas,omitempty"`
}

// RainbondVolumeSpec defines the desired state of RainbondVolume
type RainbondVolumeSpec struct {
	// The name of StorageClass, which is a kind of kubernetes resource.
	// It will used to create pvc for rainbond components.
	// More info: https://kubernetes.io/docs/concepts/storage/storage-classes/
	StorageClassName string `json:"storageClassName,omitempty"`
	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	StorageClassParameters map[string]string `json:"storageClassParameters,omitempty"`
	// +kubebuilder:validation:MaxProperties=100
	// CSIPlugin holds the image
	CSIPlugin CSIPluginSource `json:"csiPlugin,omitempty"`
}

// RainbondVolumeStatus defines the observed state of RainbondVolume
type RainbondVolumeStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RainbondVolume is the Schema for the rainbondvolumes API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rainbondvolumes,scope=Namespaced
type RainbondVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RainbondVolumeSpec    `json:"spec,omitempty"`
	Status *RainbondVolumeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RainbondVolumeList contains a list of RainbondVolume
type RainbondVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RainbondVolume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RainbondVolume{}, &RainbondVolumeList{})
}
