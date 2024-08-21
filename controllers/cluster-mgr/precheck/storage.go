/*
precheck 包为 Operator 提供了安装平台集群所需的环境和参数的预检查功能。

该包中的 `storage` 结构体实现了存储资源的预检查器，用于检查 Kubernetes 集群中指定的 PersistentVolumeClaim (PVC) 是否已绑定，并确保存储类 (StorageClass) 配置正确。

`NewStorage` 函数创建一个新的存储预检查器实例，该实例用于执行存储资源的检查，并将检查结果作为 `RainbondClusterCondition` 返回。

主要组件：
- `storage`：一个结构体，包含上下文、Kubernetes 客户端、命名空间和 RainbondVolumeSpec，用于执行存储资源的预检查。
- `NewStorage`：一个函数，用于创建新的 `storage` 预检查器实例。
- `Check`：`storage` 结构体上的一个方法，执行存储资源的检查，并返回表示检查状态的 `RainbondClusterCondition`。如果 PVC 未绑定或存在其他存储问题，将返回失败的条件状态。
- `isPVCBound`：一个辅助方法，用于检查指定的 PVC 是否已绑定。
- `pvcForGrdata`：一个辅助方法，用于创建与 Rainbond 相关的 PVC 资源。
- `failCondition`：`storage` 结构体上的一个辅助方法，用于设置检查失败时的状态和错误消息，并返回失败的 `RainbondClusterCondition`。
- `eventListToString`：一个辅助函数，用于将与 PVC 相关的事件列表转换为字符串形式，以便在日志和消息中使用。
*/

package precheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type storage struct {
	ctx    context.Context
	client client.Client
	ns     string
	rwx    *rainbondv1alpha1.RainbondVolumeSpec
}

// NewStorage -
func NewStorage(ctx context.Context, client client.Client, ns string, rwx *rainbondv1alpha1.RainbondVolumeSpec) PreChecker {
	return &storage{
		ctx:    ctx,
		client: client,
		ns:     ns,
		rwx:    rwx,
	}
}

func (s *storage) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeStorage,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	if s.rwx != nil && s.rwx.StorageClassName != "" {
		if s.rwx.StorageClassName != "" {
			// check if pvc exists
			pvc, err := k8sutil.GetFoobarPVC(s.ctx, s.client, s.ns)
			if err != nil {
				return s.failConditoin(condition, err.Error())
			}

			if !s.isPVCBound(pvc) {
				// list Events
				eventList, err := k8sutil.EventsForPersistentVolumeClaim(pvc)
				if err != nil {
					return s.failConditoin(condition, err.Error())
				}
				return s.failConditoin(condition, eventListToString(eventList))
			}
		}
		return condition
	}

	if s.rwx == nil {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "InProgress"
		condition.Message =
			fmt.Sprintf("precheck for %s is in progress", rainbondv1alpha1.RainbondClusterConditionTypeStorage)
		return condition
	}

	return condition
}

func (s *storage) isPVCBound(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Status.Phase == corev1.ClaimBound {
		return true
	}
	return false
}

func (s *storage) pvcForGrdata(accessModes []corev1.PersistentVolumeAccessMode, storageClassName string) *corev1.PersistentVolumeClaim {
	labels := rbdutil.LabelsForRainbond(nil)
	return k8sutil.PersistentVolumeClaimForGrdata(s.ns, constants.GrDataPVC, accessModes, labels, storageClassName, 1)
}

func (s *storage) failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, msg string) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "StorageFailed", msg)
}

func eventListToString(eventList *corev1.EventList) string {
	var res []string
	for _, event := range eventList.Items {
		res = append(res, fmt.Sprintf("%s: %s", event.Reason, event.Message))
	}
	return strings.Join(res, ",")
}
