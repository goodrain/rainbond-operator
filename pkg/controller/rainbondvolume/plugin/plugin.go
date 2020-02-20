package plugin

type CSIPlugin interface {
	CheckIfCSIDriverExists() bool
	GetProvisioner() string
	GetResources() []interface{}
}
