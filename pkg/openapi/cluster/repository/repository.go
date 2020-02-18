package repository

// Repository represent the cluster's repository contract
type Repository interface {
	EnterpriseID() (string, error)
	InstallID() string
}
