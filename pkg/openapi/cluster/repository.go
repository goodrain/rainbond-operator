package cluster

// Repository represent the cluster's repository contract
type Repository interface {
	EnterpriseID(eid string) string
}
