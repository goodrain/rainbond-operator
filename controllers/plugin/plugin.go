package plugin

import "sigs.k8s.io/controller-runtime/pkg/client"

//CSIPlugin csi plugin
type CSIPlugin interface {
	// TODO: rename IsPluginReady
	IsPluginReady() bool
	GetProvisioner() string
	GetClusterScopedResources() []client.Object
	GetSubResources() []client.Object
}
