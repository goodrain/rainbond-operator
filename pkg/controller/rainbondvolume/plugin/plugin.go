package plugin

import (
	storagev1 "k8s.io/api/storage/v1"
)

type CSIPlugin interface {
	GetResources() []interface{}
	GetStorageClass() *storagev1.StorageClass
}
