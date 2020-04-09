package bcode

// business code for pkg/openapi/cluster
var (
	// 50000~59999 for global configs.
	// ErrGenerateAdmin failed to generate administrator
	ErrGenerateAdmin = newCodeWithMsg(50001, "failed to generate administrator")
	// DoNotAllowGenerateAdmin do not allow generate administrator more than one
	DoNotAllowGenerateAdmin = newCodeWithMsg(50002, "have been generated")
)
