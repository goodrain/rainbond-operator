package v1

// UpgradeReq
type UpgradeReq struct {
	Version string `json:"version" binding:"required"`
}

// UpgradeVersionsResp -
type UpgradeVersionsResp struct {
	CurrentVersion      string `json:"currentVersion"`
	UpgradeableVersions []string `json:"upgradeableVersions"`
}
