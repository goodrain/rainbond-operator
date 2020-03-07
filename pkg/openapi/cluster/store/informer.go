package store

import (
	"k8s.io/client-go/tools/cache"
)

//Informer kube-api client cache
type Informer struct {
	Pod          cache.SharedIndexInformer
	RbdComponent cache.SharedIndexInformer
	Event        cache.SharedIndexInformer
}

//Start statrt
func (i *Informer) Start(stop chan struct{}) {
	go i.Pod.Run(stop)
	go i.RbdComponent.Run(stop)
	go i.Event.Run(stop)
}

//Ready if all kube informers is syncd, store is ready
func (i *Informer) Ready() bool {
	if i.Pod.HasSynced() && i.RbdComponent.HasSynced() && i.Event.HasSynced() {
		return true
	}
	return false
}
