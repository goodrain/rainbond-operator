package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewRainbondClusterCondition creates a new rianbondcluster condition.
func NewRainbondClusterCondition(condType RainbondClusterConditionType, status v1.ConditionStatus, reason, message string) *RainbondClusterCondition {
	return &RainbondClusterCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// SetCondition setups the given rainbondcluster condition.
func (r *RainbondClusterStatus) SetCondition(c RainbondClusterCondition) {
	pos, cp := r.GetCondition(c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		return
	}

	if cp != nil {
		r.Conditions[pos] = c
	} else {
		r.Conditions = append(r.Conditions, c)
	}
}

// GetCondition returns a rbdcomponent condition based on the given type.
func (r *RainbondClusterStatus) GetCondition(t RainbondClusterConditionType) (int, *RainbondClusterCondition) {
	for i, c := range r.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

// UpdateCondition updates existing rbdcomponent condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if rbdcomponent condition has changed or has been added.
func (r *RainbondClusterStatus) UpdateCondition(condition *RainbondClusterCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this RainbondVolume condition.
	conditionIndex, oldCondition := r.GetCondition(condition.Type)

	if oldCondition == nil {
		// We are adding new RainbondVolume condition.
		r.Conditions = append(r.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	r.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}
func (r *RainbondClusterStatus) DeleteCondition(typ3 RainbondClusterConditionType) {
	idx, _ := r.GetCondition(typ3)
	if idx == -1 {
		return
	}
	r.Conditions = append(r.Conditions[:idx], r.Conditions[idx+1:]...)
}
