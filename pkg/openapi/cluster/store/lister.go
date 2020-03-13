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

type PodLister struct {
	cache.Store
}

type RbdComponentLister struct {
	cache.Store
}

type EventLister struct {
	cache.Store
}
