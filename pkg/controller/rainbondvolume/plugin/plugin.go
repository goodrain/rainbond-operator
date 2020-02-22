package plugin

type CSIPlugin interface {
	// TODO: rename IsPluginReady
	IsPluginReady() bool
	GetProvisioner() string
	GetClusterScopedResources() []interface{}
	GetSubResources() []interface{}
}
