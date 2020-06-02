package bcode

// business code for pkg/openapi/cluster
var (
	// 10000~19999 for global configs.
	// ErrGenHTTPDomain failed to generate http domain
	ErrGenHTTPDomain = newCodeWithMsg(10001, "failed to generate http domain")
	ErrInvalidNodes  = newCodeWithMsg(10002, "invalid nodes")
	// ErrClusterNotFound rainbondcluster not found
	ErrClusterNotFound          = newCodeWithStatus(404, 10003, "rainbondcluster not found")
	ErrInvalidKubernetesVersion = newCodeWithStatus(400, 10004, "The version of kubernetes cannot be less than v1.13.0")

	// 20000~29999 for rainbond package
	// ErrCreateRainbondPackage failed to create rainbond package
	ErrCreateRainbondPackage = newCodeWithMsg(20001, "failed to create rainbond package")

	// 30000~39999 for rainbond volume
	// ErrCreateRainbondVolume failed to create rainbond volume
	ErrCreateRainbondVolume = newCodeWithMsg(30001, "failed to create rainbond volume")

	// 40000~49999 for rainbond component
	// ErrCreateRainbondVolume failed to create rainbond component
	ErrCreateRbdComponent   = newCodeWithMsg(40001, "failed to create rainbond component")
	ErrRbdComponentNotFound = newCodeWithStatus(404, 40002, "rbdcomponent not found")
)
