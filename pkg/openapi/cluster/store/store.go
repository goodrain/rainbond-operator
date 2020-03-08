package store

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/tools/cache"

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
	Stop()
	ListRbdComponent(isInit bool) []*v1alpha1.RbdComponent
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
	enableEtcdOperator  bool
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
	if enableEtcdOperator, _ := strconv.ParseBool(os.Getenv("ENABLE_ETCD_OPERATOR")); enableEtcdOperator {
		store.enableEtcdOperator = true
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
	if ready := cache.WaitForCacheSync(s.stopch, s.informers.Pod.HasSynced, s.informers.Event.HasSynced, s.informers.RbdComponent.HasSynced); ready {
		return fmt.Errorf("wait sync component timeout")
	}
	return nil
}

// Stop to do
func (s *componentRuntimeStore) Stop() {
	close(s.stopch)
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
		if s.enableEtcdOperator { // hack etcd name
			if name, ok := pod.Labels["etcd_cluster"]; ok && name != "" {
				pod.Labels["name"] = name
			}
		}
		if s.enableMysqlOperator { // hack db name
			if name, ok := pod.Labels["v1alpha1.mysql.oracle.com/cluster"]; ok && name != "" {
				pod.Labels["name"] = name
			}
		}
		pods = append(pods, pod)
	}
	return pods
}

// ListEvent list pod and rbdcomponent events
func (s *componentRuntimeStore) ListEvent() []*corev1.Event {
	var events []*corev1.Event
	for _, obj := range s.listers.Event.List() {
		event := obj.(*corev1.Event)
		if event.Type == corev1.EventTypeWarning && (event.InvolvedObject.Kind == "Pod" || event.InvolvedObject.Kind == "RbdComponent") {
			events = append(events, event)
		}
	}
	return events
}
