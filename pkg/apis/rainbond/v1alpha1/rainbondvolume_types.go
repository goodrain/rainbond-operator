package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AliyunCloudDiskCSIPluginSource represents a aliyun cloud disk CSI plugin.
// More info: https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/master/docs/disk.md
type AliyunCloudDiskCSIPluginSource struct {
	// The AccessKey ID provided by Alibaba Cloud for access control.
	AccessKeyID string `json:"accessKeyID"`
	// The AccessKey Secret provided by Alibaba Cloud for access control
	AccessKeySecret string `json:"accessKeySecret"`
	// maxVolumePerNode
	MaxVolumePerNode string `json:"maxVolumePerNode"`
}

// AliyunNasCSIPluginSource represents a aliyun cloud nas CSI plugin.
// More info: https://github.com/GLYASAI/alibaba-cloud-csi-driver/blob/master/docs/nas.md
type AliyunNasCSIPluginSource struct {
	// The AccessKey ID provided by Alibaba Cloud for access control.
	AccessKeyID string `json:"accessKeyID"`
	// The AccessKey Secret provided by Alibaba Cloud for access control
	AccessKeySecret string `json:"accessKeySecret"`
}

// NFSCSIPluginSource represents a nfs CSI plugin.
// More info: https://github.com/kubernetes-incubator/external-storage/tree/master/nfs
type NFSCSIPluginSource struct {
}

// StorageClassParameters describes the parameters for a class of storage for
// which PersistentVolumes can be dynamically provisioned.
type StorageClassParameters struct {
	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with these mountOptions, e.g. ["ro", "soft"]. Not validated -
	// mount of the PVs will simply fail if one is invalid.
	// +optional
	MountOptions []string `json:"mountOptions,omitempty" protobuf:"bytes,5,opt,name=mountOptions"`

	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner,omitempty" protobuf:"bytes,2,opt,name=provisioner"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`
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
	// NFSCSIPluginSource represents a nfs CSI plugin.
	// More info: https://github.com/kubernetes-incubator/external-storage/tree/master/nfs
	NFS *NFSCSIPluginSource `json:"nfs,omitempty"`
}

// RainbondVolumeSpec defines the desired state of RainbondVolume
type RainbondVolumeSpec struct {
	// The name of StorageClass, which is a kind of kubernetes resource.
	// It will used to create pvc for rainbond components.
	// More info: https://kubernetes.io/docs/concepts/storage/storage-classes/
	StorageClassName       string                  `json:"storageClassName,omitempty"`
	StorageClassParameters *StorageClassParameters `json:"storageClassParameters,omitempty"`
	// CSIPlugin holds the image
	CSIPlugin       *CSIPluginSource `json:"csiPlugin,omitempty"`
	StorageRequest  *int32           `json:"storageRequest,omitempty"`
	ImageRepository string           `json:"imageRepository"`
}

// RainbondVolumeConditionType -
type RainbondVolumeConditionType string

const (
	// RainbondVolumeReady means the raionbondvolume is ready.
	RainbondVolumeReady RainbondVolumeConditionType = "Ready"
	// RainbondVolumeProgressing means the raionbondvolume is progressing.
	RainbondVolumeProgressing RainbondVolumeConditionType = "Progressing"
)

// RainbondVolumeCondition represents one current condition of an rainbondvolume.
type RainbondVolumeCondition struct {
	// Type of rainbondvolume condition.
	Type RainbondVolumeConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// RainbondVolumeStatus defines the observed state of rainbondvolume.
type RainbondVolumeStatus struct {
	// Condition keeps track of all rainbondvolume conditions, if they exist.
	Conditions []RainbondVolumeCondition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RainbondVolume is the Schema for the rainbondvolumes API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rainbondvolumes,scope=Namespaced
type RainbondVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RainbondVolumeSpec   `json:"spec,omitempty"`
	Status RainbondVolumeStatus `json:"status,omitempty"`
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

// GetRainbondVolumeCondition returns a condition based on the given type.
func (in *RainbondVolumeStatus) GetRainbondVolumeCondition(t RainbondVolumeConditionType) (int, *RainbondVolumeCondition) {
	for i, c := range in.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

// UpdateRainbondVolumeCondition updates existing RainbondVolume condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if RainbondVolume condition has changed or has been added.
func (in *RainbondVolumeStatus) UpdateRainbondVolumeCondition(condition *RainbondVolumeCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this RainbondVolume condition.
	conditionIndex, oldCondition := in.GetRainbondVolumeCondition(condition.Type)

	if oldCondition == nil {
		// We are adding new RainbondVolume condition.
		in.Conditions = append(in.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	in.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

func (in *RainbondVolumeStatus) setRainbondVolumeCondition(r RainbondVolumeCondition) {
	pos, cp := in.GetRainbondVolumeCondition(r.Type)
	if cp != nil &&
		cp.Status == r.Status && cp.Reason == r.Reason && cp.Message == r.Message {
		return
	}

	if cp != nil {
		in.Conditions[pos] = r
	} else {
		in.Conditions = append(in.Conditions, r)
	}
}
