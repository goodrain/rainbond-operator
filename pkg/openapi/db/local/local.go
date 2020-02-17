package local

// Manager
type Manager struct {
	InitPath string
}

// CreateManager create local manager
func CreateManager(initPath string) (*Manager, error) {
	return &Manager{InitPath: initPath}, nil
}
