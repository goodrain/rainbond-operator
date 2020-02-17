package db

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/db/config"
	"github.com/goodrain/rainbond-operator/pkg/openapi/db/dao"
	"github.com/goodrain/rainbond-operator/pkg/openapi/db/local"
)

// Manager manage db
type Manager interface {
	EnterpriseDao() dao.EnterpriseDao
	InstallDao() dao.InstallDao
}

var defaultManager Manager

// CreateManager create db manager
func CreateManager(cfg config.Config) (err error) {
	defaultManager, err = local.CreateManager(cfg.InitPath)
	return err
}

// GetManager get db manager
func GetManager() Manager {
	return defaultManager
}
