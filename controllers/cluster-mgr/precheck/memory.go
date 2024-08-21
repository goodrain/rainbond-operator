/*
precheck 包为 Operator 提供了预检查功能。

该包中的 `memory` 结构体实现了内存资源的预检查器，用于检查 Kubernetes 集群中可用的节点内存是否满足 Rainbond 的最低内存要求（2GB 及以上）。

`NewMemory` 函数创建一个新的内存预检查器实例，该实例用于执行内存资源的检查，并将检查结果作为 `RainbondClusterCondition` 返回。

主要组件：
- `memory`：一个结构体，包含上下文、日志记录器和 Kubernetes 客户端，用于执行集群节点内存资源的预检查。
- `NewMemory`：一个函数，用于创建新的 `memory` 预检查器实例。
- `Check`：`memory` 结构体上的一个方法，执行内存资源的检查，并返回表示检查状态的 `RainbondClusterCondition`。如果可用内存不足，将返回失败的条件状态。
- `failCondition`：`memory` 结构体上的一个辅助方法，用于设置检查失败时的状态和错误消息，并返回失败的 `RainbondClusterCondition`。
- `filterOut`：一个辅助方法，用于过滤掉不可调度的节点和主节点，以便仅检查符合条件的工作节点。
- `masterNode`：一个辅助函数，用于判断节点是否为主节点。
- `totalMemory`：一个辅助函数，用于计算所有符合条件的节点的可用内存总量。
*/

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
