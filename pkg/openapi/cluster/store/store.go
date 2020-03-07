package store

import (
	"sync"
	"time"

	rainboondInformers "github.com/goodrain/rainbond-operator/pkg/generated/informers/externalversions"

	corev1 "k8s.io/api/core/v1"

	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	"k8s.io/client-go/informers"

	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// Storer rainbond component store interface
type Storer interface {
	Start() error
	Stop() error
	Ready() bool
	ListRbdComponent(isInit bool) []*v1alpha1.RbdComponent
	ListPod() []*corev1.Pod
	ListEvent() []*corev1.Event
}

type componentRuntimeStore struct {
	namespace      string
	rainbondClient *versioned.Clientset
	k8sClient      *kubernetes.Clientset
	informers      *Informer
	listers        *Lister
	appServices    sync.Map
	stopch         chan struct{}
}

// NewStore TODO close it
func NewStore(namespace string, rainbondClient *versioned.Clientset, k8sClient *kubernetes.Clientset) Storer {
	store := &componentRuntimeStore{
		rainbondClient: rainbondClient,
		k8sClient:      k8sClient,
		informers:      &Informer{},
		listers:        &Lister{},
		appServices:    sync.Map{},
	}
	// create informers factory, enable and assign required informers
	infFactory := informers.NewSharedInformerFactoryWithOptions(k8sClient, time.Second, informers.WithNamespace(namespace))
	rainbondInfFactory := rainboondInformers.NewSharedInformerFactoryWithOptions(rainbondClient, time.Second, rainboondInformers.WithNamespace(namespace))

	store.informers.Pod = infFactory.Core().V1().Pods().Informer()
	store.listers.Pod.Store = store.informers.Pod.GetStore()

	store.informers.RbdComponent = rainbondInfFactory.Rainbond().V1alpha1().RbdComponents().Informer()
	store.listers.RbdComponent.Store = store.informers.RbdComponent.GetStore()

	store.informers.Event = infFactory.Core().V1().Events().Informer()
	store.listers.Event.Store = store.informers.Event.GetStore()

	return store
}

// Start start store
func (s *componentRuntimeStore) Start() error {
	stopch := make(chan struct{})
	s.informers.Start(stopch)
	s.stopch = stopch
	for !s.Ready() {
	}
	return nil
}

// Stop to do
func (s *componentRuntimeStore) Stop() error {
	s.stopch <- struct{}{}
	return nil
}

// store is ready or not
func (s *componentRuntimeStore) Ready() bool {
	return s.informers.Ready()
}

// ListRbdComponent list rbdcomponent
func (s *componentRuntimeStore) ListRbdComponent(isInit bool) []*v1alpha1.RbdComponent {
	var components []*v1alpha1.RbdComponent
	for _, obj := range s.listers.RbdComponent.List() {
		component := obj.(*v1alpha1.RbdComponent)
		if component.Namespace == s.namespace {
			if !component.Spec.PriorityComponent && isInit {
				continue
			}
			components = append(components, component)
		}
	}
	return components
}

// ListPod list all pods
func (s *componentRuntimeStore) ListPod() []*corev1.Pod {
	var pods []*corev1.Pod
	for _, obj := range s.listers.Pod.List() {
		pod := obj.(*corev1.Pod)
		if pod.Namespace == s.namespace {
			pods = append(pods, pod)
		}
	}
	return pods
}

// ListEvent list events
func (s *componentRuntimeStore) ListEvent() []*corev1.Event {
	var events []*corev1.Event
	for _, obj := range s.listers.Event.List() {
		event := obj.(*corev1.Event)
		if event.Namespace == s.namespace {
			events = append(events, event)
		}
	}
	return events
}
