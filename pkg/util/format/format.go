package format

import (
	"fmt"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// Pod returns a string representing a pod in a consistent human readable format,
// with pod UID as part of the string.
func RainbondCluster(rc *rainbondv1alpha1.RainbondCluster) string {
	return RainbondClusterDesc(rc.Name, rc.Namespace, rc.UID)
}

// RainbondClusterDesc returns a string representing a RainbondCluster in a consistent human readable format,
// with RainbondCluster UID as part of the string.
func RainbondClusterDesc(rainbondClusterName, rainbondClusterNamespace string, rainbondClusterUID types.UID) string {
	// Use underscore as the delimiter because it is not allowed in RainbondCluster name
	// (DNS subdomain format), while allowed in the container name format.
	return fmt.Sprintf("%s_%s(%s)", rainbondClusterName, rainbondClusterNamespace, rainbondClusterUID)
}
