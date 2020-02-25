package bcode

var (
	OK = newCodeWithMsg(200, "ok")

	// BadRequest means the request could not be understood by the server due to malformed syntax.
	// The client SHOULD NOT repeat the request without modifications.
	BadRequest = newCode(400)
	// NotFound means the server has not found anything matching the request.
	NotFound = newCode(404)
	// ServerErr means  the server encountered an unexpected condition which prevented it from fulfilling the request.
	ServerErr = newCode(500)
)
