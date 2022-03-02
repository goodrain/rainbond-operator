package precheck

import (
	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func failConditoin(condition wutongv1alpha1.WutongClusterCondition, reason, msg string) wutongv1alpha1.WutongClusterCondition {
	condition.Status = corev1.ConditionFalse
	condition.Reason = reason
	condition.Message = msg
	return condition
}
