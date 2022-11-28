/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// WutongVolumeSpec defines the desired state of WutongVolume
type WutongVolumeSpec struct {
	// The name of StorageClass, which is a kind of kubernetes resource.
	// It will used to create pvc for wutong components.
	// More info: https://kubernetes.io/docs/concepts/storage/storage-classes/
	StorageClassName       string                  `json:"storageClassName,omitempty"`
	StorageClassParameters *StorageClassParameters `json:"storageClassParameters,omitempty"`
	// CSIPlugin holds the image
	CSIPlugin       *CSIPluginSource `json:"csiPlugin,omitempty"`
	StorageRequest  *int32           `json:"storageRequest,omitempty"`
	ImageRepository string           `json:"imageRepository"`
}

// WutongVolumeConditionType -
type WutongVolumeConditionType string

const (
	// WutongVolumeReady means the raionbondvolume is ready.
	WutongVolumeReady WutongVolumeConditionType = "Ready"
	// WutongVolumeProgressing means the raionbondvolume is progressing.
	WutongVolumeProgressing WutongVolumeConditionType = "Progressing"
)

// WutongVolumeCondition represents one current condition of an WutongVolume.
type WutongVolumeCondition struct {
	// Type of WutongVolume condition.
	Type WutongVolumeConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// WutongVolumeStatus defines the observed state of WutongVolume
type WutongVolumeStatus struct {
	// Condition keeps track of all WutongVolume conditions, if they exist.
	Conditions []WutongVolumeCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WutongVolume is the Schema for the WutongVolumes API
type WutongVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WutongVolumeSpec   `json:"spec,omitempty"`
	Status WutongVolumeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WutongVolumeList contains a list of WutongVolume
type WutongVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WutongVolume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WutongVolume{}, &WutongVolumeList{})
}

// GetWutongVolumeCondition returns a condition based on the given type.
func (in *WutongVolumeStatus) GetWutongVolumeCondition(t WutongVolumeConditionType) (int, *WutongVolumeCondition) {
	for i, c := range in.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

// UpdateWutongVolumeCondition updates existing WutongVolume condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if WutongVolume condition has changed or has been added.
func (in *WutongVolumeStatus) UpdateWutongVolumeCondition(condition *WutongVolumeCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this WutongVolume condition.
	conditionIndex, oldCondition := in.GetWutongVolumeCondition(condition.Type)

	if oldCondition == nil {
		// We are adding new WutongVolume condition.
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

// func (in *WutongVolumeStatus) setWutongVolumeCondition(r WutongVolumeCondition) {
// 	pos, cp := in.GetWutongVolumeCondition(r.Type)
// 	if cp != nil &&
// 		cp.Status == r.Status && cp.Reason == r.Reason && cp.Message == r.Message {
// 		return
// 	}

// 	if cp != nil {
// 		in.Conditions[pos] = r
// 	} else {
// 		in.Conditions = append(in.Conditions, r)
// 	}
// }
