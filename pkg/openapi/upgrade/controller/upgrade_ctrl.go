package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade"
	"github.com/goodrain/rainbond-operator/pkg/util/ginutil"
	"github.com/sirupsen/logrus"
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
	e.POST("/versions", u.upgrade)
}

func (u *UpgradeController) versions(c *gin.Context) {
	versions, err := u.upgradeUcase.Versions()
	if err != nil {
		ginutil.JSON(c, nil, err)
		return
	}
	ginutil.JSON(c, versions, nil)
}

func (u *UpgradeController) upgrade(c *gin.Context) {
	var req v1.UpgradeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Debugf("[upgrade] ShouldBindJSON: %v", err)
		ginutil.JSON(c, nil, bcode.BadRequest)
		return
	}

	err := u.upgradeUcase.Upgrade(&req)
	ginutil.JSON(c, nil, err)
}
