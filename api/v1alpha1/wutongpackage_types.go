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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WutongPackagePhase is a label for the condition of a WutongCluster at the current time.
type WutongPackagePhase string

//PackageConditionType PackageConditionType
type PackageConditionType string

// These are valid conditions of package.
const (
	// PackageConditionType means this package handle status
	Init            PackageConditionType = "Init"
	DownloadPackage PackageConditionType = "DownloadPackage"
	UnpackPackage   PackageConditionType = "UnpackPackage"
	PushImage       PackageConditionType = "PushImage"
	Ready           PackageConditionType = "Ready"
)

//PackageConditionStatus condition status
type PackageConditionStatus string

const (
	//Waiting waiting
	Waiting PackageConditionStatus = "Waiting"
	//Running Running
	Running PackageConditionStatus = "Running"
	//Completed Completed
	Completed PackageConditionStatus = "Completed"
	//Failed Failed
	Failed PackageConditionStatus = "Failed"
)

// PackageCondition contains condition information for package.
type PackageCondition struct {
	// Type of package condition.
	Type PackageConditionType `json:"type" `
	// Status of the condition, one of True, False, Unknown.
	Status PackageConditionStatus `json:"status" `
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty" `
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" `
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
	// The progress of the condition
	// +optional
	Progress int `json:"progress,omitempty"`
}

//WutongPackageImage image
type WutongPackageImage struct {
	//Name image name
	Name string `json:"name,omitempty"`
}

// WutongPackageSpec defines the desired state of WutongPackage
type WutongPackageSpec struct {
	// Deprecated: The path where the wutong package is located.
	PkgPath string `json:"pkgPath"`
	// install source image hub user
	ImageHubUser string `json:"imageHubUser"`
	// install source image hub password
	ImageHubPass string `json:"imageHubPass"`
}

// WutongPackageStatus defines the observed state of WutongPackage
type WutongPackageStatus struct {
	//worker and master maintenance
	Conditions []PackageCondition `json:"conditions,omitempty"`
	// The number of images that should be load and pushed.
	ImagesNumber int32 `json:"imagesNumber"`
	// ImagesPushed contains the images have been pushed.
	ImagesPushed []WutongPackageImage `json:"images,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WutongPackage is the Schema for the WutongPackages API
type WutongPackage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WutongPackageSpec   `json:"spec,omitempty"`
	Status WutongPackageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WutongPackageList contains a list of WutongPackage
type WutongPackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WutongPackage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WutongPackage{}, &WutongPackageList{})
}

// GetCondition returns a WutongPackage condition based on the given type.
func (r *WutongPackageStatus) GetCondition(t PackageConditionType) (int, *PackageCondition) {
	for i, c := range r.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}
