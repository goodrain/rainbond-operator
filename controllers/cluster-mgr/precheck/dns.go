/*
precheck 包为 Operator 提供了预检查功能。

该包中的 `dns` 结构体实现了 DNS 预检查器，用于验证集群配置中指定的 Rainbond 镜像仓库的 DNS 解析是否正确。

`NewDNSPrechecker` 函数创建一个新的 DNS 预检查器实例，该实例用于执行 DNS 解析检查，并将检查结果作为 `RainbondClusterCondition` 返回。

主要组件：
- `dns`：一个结构体，包含日志记录器和 RainbondCluster 的实例，用于执行 DNS 解析检查。
- `NewDNSPrechecker`：一个函数，用于创建新的 `dns` 预检查器实例。
- `Check`：`dns` 结构体上的一个方法，执行 DNS 解析检查，并返回表示检查状态的 `RainbondClusterCondition`。如果 DNS 解析失败，将返回失败的条件状态。
- `nslookup`：一个辅助函数，使用 `net.LookupIP` 方法执行 DNS 查询，并返回查询结果。
- `failCondition`：`dns` 结构体上的一个辅助方法，用于设置检查失败时的状态和错误消息，并返回失败的 `RainbondClusterCondition`。
*/

package precheck

import (
	"net"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dns struct {
	log     logr.Logger
	cluster *rainbondv1alpha1.RainbondCluster
}

// NewDNSPrechecker creates a new prechecker.
func NewDNSPrechecker(cluster *rainbondv1alpha1.RainbondCluster, log logr.Logger) PreChecker {
	return &dns{
		log:     log.WithName("DNSPreChecker"),
		cluster: cluster,
	}
}

func (d *dns) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeDNS,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	ref, err := reference.Parse(d.cluster.Spec.RainbondImageRepository)
	if err != nil {
		return d.failCondition(condition, err.Error())
	}
	domain := reference.Domain(ref.(reference.Named))

	// TODO: support offline installation
	if err := nslookup(domain); err != nil {
		return d.failCondition(condition, err.Error())
	}

	return condition
}

func nslookup(target string) error {
	_, err := net.LookupIP(target)
	if err != nil {
		return err
	}
	return nil
}

func (d *dns) failCondition(condition rainbondv1alpha1.RainbondClusterCondition, msg string) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "DNSFailed", msg)
}
