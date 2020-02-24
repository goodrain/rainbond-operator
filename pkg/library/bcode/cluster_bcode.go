package bcode

// business code for pkg/openapi/cluster
var (
	// ErrGenHTTPDomain 10000~19999 for global configs.
	ErrGenHTTPDomain = newCodeWithMsg(10001, "failed to generate http domain")
)
