package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade"
	"github.com/goodrain/rainbond-operator/pkg/util/ginutil"
)

type UpgradeController struct {
	upgradeUcase upgrade.Usecase
}

// NewUpgradeController creates a new UpgradeController
func NewUpgradeController(g *gin.Engine, upgradeUcase upgrade.Usecase) {
	u := &UpgradeController{
		upgradeUcase: upgradeUcase,
	}

	e := g.Group("/upgrade")
	e.GET("/versions", u.versions)
}

func (u *UpgradeController) versions(c *gin.Context) {
	versions, err := u.upgradeUcase.Versions()
	if err != nil {
		ginutil.JSON(c, nil, err)
		return
	}
	ginutil.JSON(c, versions, nil)
}
