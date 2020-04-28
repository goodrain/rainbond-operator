package store

import (
	"k8s.io/client-go/tools/cache"
)

//Lister kube-api client cache
type Lister struct {
	Pod          PodLister
	RbdComponent RbdComponentLister
	Event        EventLister
}

// PodLister -
type PodLister struct {
	cache.Store
}

// RbdComponentLister -
type RbdComponentLister struct {
	cache.Store
}

// EventLister -
type EventLister struct {
	cache.Store
}
