package handler

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentHandler will check the prerequisites, create resources for WutongComponent.
type ComponentHandler interface {
	// Before will do something before creating component, such as checking the prerequisites, etc.
	Before() error
	Resources() []client.Object
	After() error
	ListPods() ([]corev1.Pod, error)
}

// StorageClassRWXer provides methods to setup storageclass with
// access mode RWX for WutongComponent.
type StorageClassRWXer interface {
	SetStorageClassNameRWX(pvcParameters *pvcParameters)
}

// StorageClassRWOer provides methods to setup storageclass with
// access mode RWO for WutongComponent.
type StorageClassRWOer interface {
	SetStorageClassNameRWO(pvcParameters *pvcParameters)
}

// ResourcesCreator provides methods to create or update k8s resources,
// such as daemonset, daemonset, etc.
type ResourcesCreator interface {
	// returns the resources that should be created if not exists
	ResourcesCreateIfNotExists() []client.Object
}

// ClusterScopedResourcesCreator provides methods to create or update k8s resources which in cluster-scoped ,
// such as daemonset, daemonset, etc.
type ClusterScopedResourcesCreator interface {
	// returns the resources that should be created if not exists
	CreateClusterScoped() []client.Object
}

// ResourcesDeleter -
type ResourcesDeleter interface {
	// returns the resources that need to be delete if exists.
	ResourcesNeedDelete() []client.Object
	// TODO: wait until deleting successfully
}

// Replicaser provides methods to get replicas for WutongComponent.
// This interface is generally used when the actual number of component is different from the spec definition.
type Replicaser interface {
	// return replicas for WutongComponent.
	Replicas() *int32
}
