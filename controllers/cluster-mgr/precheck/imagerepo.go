/*
precheck 包为 Operator 提供了预检查功能。

该包中的 `imagerepo` 结构体实现了镜像仓库的预检查器，用于验证集群配置中指定的镜像仓库的可用性，包括登录镜像仓库以确保访问权限。

`NewImageRepoPrechecker` 函数创建一个新的镜像仓库预检查器实例，该实例用于执行镜像仓库的检查，并将检查结果作为 `RainbondClusterCondition` 返回。

主要组件：
- `imagerepo`：一个结构体，包含上下文、日志记录器和 RainbondCluster 的实例，用于执行镜像仓库的预检查。
- `NewImageRepoPrechecker`：一个函数，用于创建新的 `imagerepo` 预检查器实例。
- `Check`：`imagerepo` 结构体上的一个方法，执行镜像仓库的预检查，并返回表示检查状态的 `RainbondClusterCondition`。如果镜像仓库无法访问，将返回失败的条件状态。
- `failCondition`：`imagerepo` 结构体上的一个辅助方法，用于设置检查失败时的状态和错误消息，并返回失败的 `RainbondClusterCondition`。
*/

package precheck

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/util/repositoryutil"
	"time"

	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type imagerepo struct {
	ctx     context.Context
	log     logr.Logger
	cluster *rainbondv1alpha1.RainbondCluster
}

// NewImageRepoPrechecker creates a new prechecker.
func NewImageRepoPrechecker(ctx context.Context, log logr.Logger, cluster *rainbondv1alpha1.RainbondCluster) PreChecker {
	l := log.WithName("ImageRepoPreChecker")
	return &imagerepo{
		ctx:     ctx,
		log:     l,
		cluster: cluster,
	}
}

func (d *imagerepo) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeImageRepository,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	imageRepo := rbdutil.GetImageRepository(d.cluster)

	if idx, cdt := d.cluster.Status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeImageRepository); (idx == -1 || cdt.Reason == "DefaultImageRepoFailed") && imageRepo != constants.DefImageRepository {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "InProgress"
		condition.Message =
			fmt.Sprintf("precheck for %s is in progress", rainbondv1alpha1.RainbondClusterConditionTypeImageRepository)
	}

	// Verify that the image repository is available
	d.log.V(6).Info("login repository", "repository", rbdutil.GetImageRepositoryDomain(d.cluster), "user", d.cluster.Spec.ImageHub.Username)

	if err := repositoryutil.LoginRepository(rbdutil.GetImageRepositoryDomain(d.cluster), d.cluster.Spec.ImageHub.Username, d.cluster.Spec.ImageHub.Password); err != nil {
		return d.failConditoin(condition, err)
	}
	return condition
}

func (d *imagerepo) failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, err error) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "ImageRepoFailed", err.Error())
}
