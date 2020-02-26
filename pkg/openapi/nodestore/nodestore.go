package nodestore

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("nodestore")

// Interface defines methods for storing v1.K8sNode.
type Interface interface {
	Start() error
	Stop()
	SearchByIIP(iip string, rungateway bool) []*v1.K8sNode
}

// New creates a new nodestore Interface.
func New(ctx context.Context, clientset kubernetes.Interface) Interface {
	newCtx, cancel := context.WithCancel(ctx)
	s := &store{
		ctx:       newCtx,
		cancel:    cancel,
		clientset: clientset,
	}
	return s
}

type store struct {
	ctx    context.Context
	cancel context.CancelFunc

	clientset           kubernetes.Interface
	k8sNodes            []*v1.K8sNode
	k8sNodesForGateweay []*v1.K8sNode

	// lock for nodes running gateway
	gnlock sync.RWMutex
}

func (s *store) Start() error {
	log.Info("start node storer")
	k8sNodes, err := s.listNodes()
	if err != nil {
		return err
	}
	s.k8sNodes = k8sNodes

	// sync nodes for gateway every 3 second.
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			s.syncK8sNodesForGateway()
			time.Sleep(3 * time.Second)
		}
	}()

	return nil
}

func (s *store) Stop() {
	s.cancel()
}

func (s *store) listNodes() ([]*v1.K8sNode, error) {
	nodeList, err := s.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %v", err)
	}

	var k8sNodes []*v1.K8sNode
	for _, node := range nodeList.Items {
		var internalIP, externalIP string
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				internalIP = addr.Address
			}
			if addr.Type == corev1.NodeExternalIP {
				externalIP = addr.Address
			}
		}

		if internalIP == "" {
			log.Info("empty internal ip", "name", node.Name)
			continue
		}

		k8sNode := &v1.K8sNode{
			Name:       node.Name,
			InternalIP: internalIP,
			ExternalIP: externalIP,
		}
		k8sNodes = append(k8sNodes, k8sNode)
	}

	return k8sNodes, nil
}

func (s *store) SearchByIIP(iip string, rungateway bool) []*v1.K8sNode {
	if rungateway {
		s.gnlock.RLock()
		defer s.gnlock.RUnlock()
		return searchByInternalIP(s.k8sNodesForGateweay, iip)
	}
	return searchByInternalIP(s.k8sNodes, iip)
}

func searchByInternalIP(k8sNodes []*v1.K8sNode, iip string) []*v1.K8sNode {
	var nodes []*v1.K8sNode
	for idx := range k8sNodes {
		node := k8sNodes[idx]
		if !strings.Contains(node.InternalIP, iip) {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func (s *store) syncK8sNodesForGateway() {
	nodes := s.k8sNodesForGateway()
	s.setNodesForGateway(nodes)
}

func (s *store) setNodesForGateway(nodes []*v1.K8sNode) {
	s.gnlock.Lock()
	defer s.gnlock.Unlock()
	s.k8sNodesForGateweay = nodes
}

func (s *store) k8sNodesForGateway() []*v1.K8sNode {
	v1node2v1alpha1 := func(nodes []*v1.K8sNode) []*rainbondv1alpha1.K8sNode {
		var result []*rainbondv1alpha1.K8sNode
		for _, n := range nodes {
			res := &rainbondv1alpha1.K8sNode{
				Name:       n.Name,
				InternalIP: n.InternalIP,
				ExternalIP: n.ExternalIP,
			}
			result = append(result, res)
		}
		return result
	}
	v1alpha1node2v1 := func(nodes []*rainbondv1alpha1.K8sNode) []*v1.K8sNode {
		var result []*v1.K8sNode
		for _, n := range nodes {
			res := &v1.K8sNode{
				Name:       n.Name,
				InternalIP: n.InternalIP,
				ExternalIP: n.ExternalIP,
			}
			result = append(result, res)
		}
		return result
	}
	v1alphav1Nodes := v1node2v1alpha1(s.k8sNodes)
	nodes := rbdutil.FilterNodesWithPortConflicts(v1alphav1Nodes)

	return v1alpha1node2v1(nodes)
}
