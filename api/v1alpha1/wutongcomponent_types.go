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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WutongComponentSpec defines the desired state of WutongComponent
type WutongComponentSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Docker image name.
	Image string `json:"image,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// +optional
	Args []string `json:"args,omitempty" protobuf:"bytes,4,rep,name=args"`
	//  Whether this component needs to be created first
	PriorityComponent bool `json:"priorityComponent"`
	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,7,rep,name=env"`
	// Compute Resources required by this container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
	// Pod volumes to mount into the container's filesystem.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=mountPath
	// +patchStrategy=merge
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" patchStrategy:"merge" patchMergeKey:"mountPath" protobuf:"bytes,9,rep,name=volumeMounts"`
	// List of volumes that can be mounted by containers belonging to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=volumes"`
}

// WutongComponentConditionType is a valid value for WutongComponentCondition.Type
type WutongComponentConditionType string

// These are valid conditions of pod.
const (
	// ClusterConfigCompeleted indicates whether the configuration of the WutongCluster cluster is complete.
	ClusterConfigCompeleted WutongComponentConditionType = "ClusterConfigCompeleted"
	// ClusterConfigCompeleted indicates whether the WutongPackage is ready.
	WutongPackageReady WutongComponentConditionType = "WutongPackageReady"
	// WutongComponentReady means all pods related to the WutongComponent are ready.
	WutongComponentReady WutongComponentConditionType = "Ready"
)

// WutongComponentCondition contains details for the current condition of this WutongComponent.
type WutongComponentCondition struct {
	// Type is the type of the condition.
	Type WutongComponentConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=PodConditionType"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions
	Status corev1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// WutongComponentStatus defines the observed state of WutongComponent
type WutongComponentStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`

	// Total number of ready pods targeted by this deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,7,opt,name=readyReplicas"`

	// Current state of wutong component.
	Conditions []WutongComponentCondition `json:"conditions,omitempty"`

	// A list of pods
	Pods []corev1.LocalObjectReference `json:"pods,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WutongComponent is the Schema for the WutongComponents API
type WutongComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WutongComponentSpec   `json:"spec,omitempty"`
	Status WutongComponentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WutongComponentList contains a list of WutongComponent
type WutongComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WutongComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WutongComponent{}, &WutongComponentList{})
}

// ImagePullPolicy returns the ImagePullPolicy, or  return PullAlways if it is empty.
func (in *WutongComponent) ImagePullPolicy() corev1.PullPolicy {
	if in.Spec.ImagePullPolicy == "" {
		return corev1.PullIfNotPresent
	}
	return in.Spec.ImagePullPolicy
}

// NewWutongComponentCondition creates a new WutongComponent condition.
func NewWutongComponentCondition(condType WutongComponentConditionType, status v1.ConditionStatus, reason, message string) *WutongComponentCondition {
	return &WutongComponentCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// SetCondition setups the given WutongComponent condition.
func (r *WutongComponentStatus) SetCondition(c WutongComponentCondition) {
	pos, cp := r.GetCondition(c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		return
	}

	if cp != nil {
		r.Conditions[pos] = c
	} else {
		r.Conditions = append(r.Conditions, c)
	}
}

// GetCondition returns a WutongComponent condition based on the given type.
func (r *WutongComponentStatus) GetCondition(t WutongComponentConditionType) (int, *WutongComponentCondition) {
	for i, c := range r.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

// UpdateCondition updates existing WutongComponent condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if WutongComponent condition has changed or has been added.
func (r *WutongComponentStatus) UpdateCondition(condition *WutongComponentCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this WutongVolume condition.
	conditionIndex, oldCondition := r.GetCondition(condition.Type)

	if oldCondition == nil {
		// We are adding new WutongVolume condition.
		r.Conditions = append(r.Conditions, *condition)
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

	r.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}
