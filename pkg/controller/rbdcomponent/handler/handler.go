package handler

import corev1 "k8s.io/api/core/v1"

// ComponentHandler will check the prerequisites, create resources for rbdcomponent.
type ComponentHandler interface {
	// Before will do something before creating component, such as checking the prerequisites, etc.
	Before() error
	Resources() []interface{}
	After() error
	ListPods() ([]corev1.Pod, error)
}

// StorageClassRWXer provides methods to setup storageclass with
// access mode RWX for rbdcomponent.
type StorageClassRWXer interface {
	SetStorageClassNameRWX(pvcParameters *pvcParameters)
}

// StorageClassRWOer provides methods to setup storageclass with
// access mode RWO for rbdcomponent.
type StorageClassRWOer interface {
	SetStorageClassNameRWO(pvcParameters *pvcParameters)
}

// K8sResourcesInterface provides methods to create or update k8s resources,
// such as daemonset, daemonset, etc.
type K8sResourcesInterface interface {
	// returns the resources that should be created if not exists
	ResourcesCreateIfNotExists() []interface{}
}

// Replicaser provides methods to get replicas for rbdcomponent.
// This interface is generally used when the actual number of component is different from the spec definition.
type Replicaser interface {
	// return replicas for rbdcomponent.
	Replicas() *int32
}
