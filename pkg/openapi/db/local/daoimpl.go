package local

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/db/dao"
	localdao "github.com/goodrain/rainbond-operator/pkg/openapi/db/local/dao"
)

// EnterpriseDao enterprise dao
func (m *Manager) EnterpriseDao() dao.EnterpriseDao {
	return &localdao.EnterpriseDaoImpl{
		InitPath: m.InitPath,
	}
}

func (m *Manager) InstallDao() dao.InstallDao {
	return new(localdao.InstallDaoImpl)
}
