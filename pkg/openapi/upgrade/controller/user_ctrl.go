package controller

import (
	"github.com/gin-gonic/gin"
)

type UpgradeController struct {
}

// NewUpgradeController creates a new UpgradeController
func NewUpgradeController(g *gin.Engine) {
	u := &UpgradeController{}

	e := g.Group("/upgrade")
	e.GET("/versions", u.versions)
}

func (u *UpgradeController) versions(c *gin.Context) {

}
