package handler

type ComponentHandler interface {
	// Before will do something before creating component, such as checking the prerequisites, etc.
	Before() error
	Resources() []interface{}
	After() error
}


