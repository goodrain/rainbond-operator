package dao

// EnterpriseDao enterprise dao
type EnterpriseDao interface {
	EnterpriseID() (string, error)
}
