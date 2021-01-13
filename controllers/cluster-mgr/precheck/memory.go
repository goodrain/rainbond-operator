package precheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goodrain/rainbond-operator/util/k8sutil"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const memoryRequest = 2 * 1024 * 1024 * 1024

type memory struct {
	ctx    context.Context
	log    logr.Logger
	client client.Client
}

// NewMemory creates a new kubernetes version prechecker.
func NewMemory(ctx context.Context, log logr.Logger, client client.Client) PreChecker {
	l := log.WithName("MemoryPrechecker")
	return &memory{
		ctx:    ctx,
		log:    l,
		client: client,
	}
}

func (m *memory) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeMemory,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	nodes, err := k8sutil.ListNodes(m.ctx, m.client)
	if err != nil {
		return m.failCondition(condition, err.Error())
	}

	nodes = m.filterOut(nodes)
	totalMemory := totalMemory(nodes)
	if totalMemory < memoryRequest {
		return m.failCondition(condition, fmt.Sprintf("expected at least %d memory, but got %d", memoryRequest, totalMemory))
	}

	return condition
}

func (m *memory) failCondition(condition rainbondv1alpha1.RainbondClusterCondition, msg string) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "MemoryFailed", msg)
}

func (m *memory) filterOut(nodes []corev1.Node) []corev1.Node {
	var res []corev1.Node
	for i := range nodes {
		node := nodes[i]
		if node.Spec.Unschedulable {
			continue
		}
		if masterNode(&node) {
			continue
		}
		res = append(res, node)
	}
	return res
}

func masterNode(node *corev1.Node) bool {
	for key := range node.GetLabels() {
		if !strings.Contains(key, "master") {
			continue
		}
		for _, taint := range node.Spec.Taints {
			if strings.Contains(taint.Key, "master") && taint.Effect == corev1.TaintEffectNoSchedule {
				return true
			}
		}
	}
	return false
}

func totalMemory(nodes []corev1.Node) int64 {
	var total int64
	for _, node := range nodes {
		total += node.Status.Allocatable.Memory().Value()
	}
	return total
}
