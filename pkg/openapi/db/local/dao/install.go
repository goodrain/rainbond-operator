package dao

import (
	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"
)

//InstallDaoImpl install dao impl
type InstallDaoImpl struct {
}

// InstallID return install id
func (impl *InstallDaoImpl) InstallID() (string, error) {
	return uuidutil.NewUUID(), nil
}
