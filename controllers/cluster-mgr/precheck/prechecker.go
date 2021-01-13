package precheck

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
)

// PreChecker checks the environment and parameters required to install the rainbond cluster
type PreChecker interface {
	Check() rainbondv1alpha1.RainbondClusterCondition
}
