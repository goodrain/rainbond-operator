package handler

type ComponentHandler interface {
	// Before will do something before creating component, such as checking the prerequisites, etc.
	Before() error
	Resources() []interface{}
	After() error
}

type StorageClassRWXer interface {
	SetStorageClassNameRWX(sc string)
}

type StorageClassRWOer interface {
	SetStorageClassNameRWO(sc string)
}

// K8sResourcesInterface provides methods to create or update k8s resources,
// such as deployment, daemonset, etc.
type K8sResourcesInterface interface {
	// returns the resources that should be created if not exists
	ResourcesCreateIfNotExists() []interface{}
}
