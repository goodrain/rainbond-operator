package store

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

//Informer kube-api client cache
type Informer struct {
	Pod          cache.SharedIndexInformer
	RbdComponent cache.SharedIndexInformer
	Event        cache.SharedIndexInformer
}

//Start statrt
func (i *Informer) Start(stopCh chan struct{}) {
	go i.Pod.Run(stopCh)
	go i.RbdComponent.Run(stopCh)
	go i.Event.Run(stopCh)

	// wait for all involved caches to be synced before processing items
	// from the queue
	if !cache.WaitForCacheSync(stopCh,
		i.Pod.HasSynced,
		i.Event.HasSynced,
		i.RbdComponent.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
	}
}
