package v1alpha1

// GetCondition returns a rainbondpackage condition based on the given type.
func (r *RainbondPackageStatus) GetCondition(t PackageConditionType) (int, *PackageCondition) {
	for i, c := range r.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}
