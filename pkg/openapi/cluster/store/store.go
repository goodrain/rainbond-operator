package store

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	rainboondInformers "github.com/goodrain/rainbond-operator/pkg/generated/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// NotExistsError is returned when an object does not exist in a local store.
type NotExistsError string

// Error implements the error interface.
func (e NotExistsError) Error() string {
	return fmt.Sprintf("no object matching key %q in local store", string(e))
}

// Storer rainbond component store interface
type Storer interface {
	Start() error
	Stop()
	ListRbdComponent(isInit bool) []*v1alpha1.RbdComponent
	ListRbdComponents() []*v1alpha1.RbdComponent
	GetPodByKey(key string) (*corev1.Pod, error)
	ListPod() []*corev1.Pod
	ListEvent() []*corev1.Event
}

type componentRuntimeStore struct {
	namespace           string
	rainbondClient      *versioned.Clientset
	k8sClient           *kubernetes.Clientset
	informers           *Informer
	listers             *Lister
	stopch              chan struct{}
	enableMysqlOperator bool
}

// NewStore TODO close it
func NewStore(namespace string, rainbondClient *versioned.Clientset, k8sClient *kubernetes.Clientset) Storer {
	store := &componentRuntimeStore{
		namespace:      namespace,
		rainbondClient: rainbondClient,
		k8sClient:      k8sClient,
		informers:      &Informer{},
		listers:        &Lister{},
		stopch:         make(chan struct{}),
	}
	if enableMysqlOperator, _ := strconv.ParseBool(os.Getenv("ENABLE_MYSQL_OPERATOR")); enableMysqlOperator {
		store.enableMysqlOperator = true
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
	s.informers.Start(s.stopch)
	return nil
}

// Stop to do
func (s *componentRuntimeStore) Stop() {
	close(s.stopch)
}

// ListRbdComponents -
func (s *componentRuntimeStore) ListRbdComponents() []*v1alpha1.RbdComponent {
	var components []*v1alpha1.RbdComponent
	for _, obj := range s.listers.RbdComponent.List() {
		components = append(components, obj.(*v1alpha1.RbdComponent))
	}
	return components
}

// ListRbdComponent list rbdcomponent
func (s *componentRuntimeStore) ListRbdComponent(isInit bool) []*v1alpha1.RbdComponent {
	var components []*v1alpha1.RbdComponent
	for _, obj := range s.listers.RbdComponent.List() {
		component := obj.(*v1alpha1.RbdComponent)
		if !component.Spec.PriorityComponent && isInit {
			continue
		}
		components = append(components, component)
	}
	return components
}

// ListPod list all pods
func (s *componentRuntimeStore) ListPod() []*corev1.Pod {
	var pods []*corev1.Pod
	for _, obj := range s.listers.Pod.List() {
		pod := obj.(*corev1.Pod)
		if s.enableMysqlOperator { // hack db name
			if name, ok := pod.Labels["v1alpha1.mysql.oracle.com/cluster"]; ok && name != "" {
				pod.Labels["name"] = name
			}
		}
		pods = append(pods, pod)
	}
	return pods
}

func (s *componentRuntimeStore) GetPodByKey(key string) (*corev1.Pod, error) {
	i, exists, err := s.listers.Pod.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return i.(*corev1.Pod), nil
}

// ListEvent list pod and rbdcomponent events
func (s *componentRuntimeStore) ListEvent() []*corev1.Event {
	var events []*corev1.Event
	for _, obj := range s.listers.Event.List() {
		event := obj.(*corev1.Event)
		if event.Type == corev1.EventTypeWarning && event.InvolvedObject.Kind == "Pod" {
			events = append(events, event)
		}
	}
	return events
}
