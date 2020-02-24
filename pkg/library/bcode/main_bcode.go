package bcode

var (
	OK = newCode(200)

	NotFound  = newCode(404) // The server has not found anything matching the request.
	ServerErr = newCode(500) // The server encountered an unexpected condition which prevented it from fulfilling the request.
)
