package upgrade

import v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"

// Usecase represent the upgrade's usecases
type Usecase interface {
	Versions() (*v1.UpgradeVersionsResp, error)
	Upgrade(req *v1.UpgradeReq) error
}
