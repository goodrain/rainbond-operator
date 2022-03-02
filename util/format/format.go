package format

import (
	"fmt"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

//WutongCluster Pod returns a string representing a pod in a consistent human readable format,
// with pod UID as part of the string.
func WutongCluster(rc *wutongv1alpha1.WutongCluster) string {
	return WutongClusterDesc(rc.Name, rc.Namespace, rc.UID)
}

// WutongClusterDesc returns a string representing a WutongCluster in a consistent human readable format,
// with WutongCluster UID as part of the string.
func WutongClusterDesc(WutongClusterName, WutongClusterNamespace string, WutongClusterUID types.UID) string {
	// Use underscore as the delimiter because it is not allowed in WutongCluster name
	// (DNS subdomain format), while allowed in the container name format.
	return fmt.Sprintf("%s_%s(%s)", WutongClusterName, WutongClusterNamespace, WutongClusterUID)
}
