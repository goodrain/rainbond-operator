/*
precheck 包为 Operator 提供了预检查功能。

该包中的 `k8sversion` 结构体实现了 Kubernetes 版本的预检查器，用于检查当前 Kubernetes 集群的版本是否符合平台所需的最低版本要求（v1.13.0 及以上）。

`NewK8sVersionPrechecker` 函数创建一个新的 Kubernetes 版本预检查器实例，该实例用于执行 Kubernetes 版本的检查，并将检查结果作为 `RainbondClusterCondition` 返回。

主要组件：
- `k8sversion`：一个结构体，包含上下文、日志记录器和 Kubernetes 客户端，用于执行 Kubernetes 版本的预检查。
- `NewK8sVersionPrechecker`：一个函数，用于创建新的 `k8sversion` 预检查器实例。
- `Check`：`k8sversion` 结构体上的一个方法，执行 Kubernetes 版本的检查，并返回表示检查状态的 `RainbondClusterCondition`。如果 Kubernetes 版本低于 v1.13.0，将返回失败的条件状态。
- `getKubernetesVersion`：一个辅助方法，用于获取当前 Kubernetes 集群的版本信息。如果获取失败或版本不符合要求，将返回相应的错误信息。
*/

package precheck

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sversion struct {
	ctx    context.Context
	log    logr.Logger
	client client.Client
}

// NewK8sVersionPrechecker creates a new kubernetes version prechecker.
func NewK8sVersionPrechecker(ctx context.Context, log logr.Logger, client client.Client) PreChecker {
	l := log.WithName("K8sVersionPreChecker")
	return &k8sversion{
		ctx:    ctx,
		log:    l,
		client: client,
	}
}

func (k *k8sversion) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeKubernetesVersion,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	version, err := k.getKubernetesVersion()
	if err != nil {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "KubernetesVersionFailed"
		condition.Message = err.Error()
		return condition
	}

	if version < "v1.13.0" {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "UnsupportedKubernetesVersion"
		condition.Message = "expect the version of k8s to be greater than or equal to 1.13.0, but got " + version
		return condition
	}

	return condition
}

func (k *k8sversion) getKubernetesVersion() (string, error) {
	nodeList := &corev1.NodeList{}
	var listOpts []client.ListOption
	if err := k.client.List(k.ctx, nodeList, listOpts...); err != nil {
		k.log.Error(err, "list nodes")
		return "", fmt.Errorf("list nodes: %v", err)
	}

	var version string
	for _, node := range nodeList.Items {
		if node.Status.NodeInfo.KubeletVersion == "" {
			continue
		}
		version = node.Status.NodeInfo.KubeletVersion
		break
	}

	if version == "" {
		return "", fmt.Errorf("failed to get kubernetes version")
	}

	return version, nil
}
