package dao

// InstallDao install dao
type InstallDao interface {
	InstallID() (string, error)
}
