package bcode

// business code for pkg/openapi/cluster
var (
	// 50000~59999 for global configs.
	// ErrGenerateAdmin failed to generate administrator
	ErrGenerateAdmin = newCodeWithMsg(50001, "failed to generate administrator")
	// DoNotAllowGenerateAdmin do not allow generate administrator more than one
	DoNotAllowGenerateAdmin = newCodeWithMsg(50002, "have been generated")
	// UserPasswordInCorrect username or password not correct
	UserPasswordInCorrect = newCodeWithMsg(50003, "username or password not correct")
	// EmptyToken token is empty
	EmptyToken = newCodeWithMsg(50004, "token is empty")
	// InvalidToken token is invalid
	InvalidToken = newCodeWithMsg(50005, "token is invalid")
	// ExpiredToken token is expired
	ExpiredToken = newCodeWithMsg(50006, "token is expired")
	// UserNotFound user not found
	UserNotFound = newCodeWithMsg(50007, "user not found")
)
