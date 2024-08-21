/*
precheck 包为 Operator 提供了安装平台集群所需的环境和参数的预检查功能。

该包中的 `failCondition` 函数是一个辅助函数，用于在预检查失败时更新并返回 `RainbondClusterCondition`，以指示特定的失败原因和错误消息。

主要组件：
- `failCondition`：一个函数，用于将 `RainbondClusterCondition` 的状态设置为 `False`，并更新失败的原因 (`Reason`) 和错误消息 (`Message`)。该函数在各个预检查器中被调用，以统一处理检查失败的情况。
*/

package precheck

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, reason, msg string) rainbondv1alpha1.RainbondClusterCondition {
	condition.Status = corev1.ConditionFalse
	condition.Reason = reason
	condition.Message = msg
	return condition
}
