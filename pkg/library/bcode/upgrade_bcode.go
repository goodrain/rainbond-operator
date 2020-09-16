package bcode

// business code for pkg/openapi/cluster
var (
	ErrInvalidVersion         = newCodeWithStatus(400, 1000, "invalid version")
	ErrCurrentVersionNotFound = newCodeWithStatus(404, 1001, "current version not found")
	ErrInvalidCurrentVersion  = newCodeWithStatus(400, 1002, "invalid current version")
	ErrLowerVersion           = newCodeWithStatus(400, 1003, "lower version, do not support downgrade")
	ErrVersionNotFound        = newCodeWithStatus(404, 1004, "version not found")
	ErrReadRbdComponent       = newCodeWithStatus(404, 1005, "can't read rbdcomponents")
)
