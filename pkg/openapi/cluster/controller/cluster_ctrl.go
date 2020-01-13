package controller

import (
	"net/http"
	"strings"

	"github.com/GLYASAI/rainbond-operator/pkg/util/corsutil"

	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/customerror"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	"github.com/gin-gonic/gin"
)

// ClusterController k8s controller
type ClusterController struct {
	clusterCase cluster.IClusterCase
}

var corsMidle = func(f gin.HandlerFunc) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		corsutil.SetCORS(ctx)
		f(ctx)
	})
}

// NewClusterController creates a new k8s controller
func NewClusterController(g *gin.Engine, clusterCase cluster.IClusterCase) {
	u := &ClusterController{clusterCase: clusterCase}

	clusterEngine := g.Group("/cluster")
	clusterEngine.GET("/configs", corsMidle(u.Configs))
	clusterEngine.PUT("/configs", corsMidle(u.UpdateConfig))

	// install
	clusterEngine.POST("/install", corsMidle(u.Install))
	clusterEngine.GET("/install/status", corsMidle(u.InstallStatus))

	// componse
	clusterEngine.GET("/components", corsMidle(u.Components))
	clusterEngine.GET("/components/:name", corsMidle(u.SingleComponent))
}

// Configs get cluster config info
func (cc *ClusterController) Configs(c *gin.Context) {
	configs, err := cc.clusterCase.GlobalConfigs().GlobalConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": configs})
}

// UpdateConfig update cluster config info
func (cc *ClusterController) UpdateConfig(c *gin.Context) {
	installStatus, err := cc.clusterCase.Install().InstallStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	if installStatus != nil {
		for _, status := range installStatus {
			if status.StepName == "step_setting" && status.Status != "status_finished" {
				c.JSON(http.StatusBadRequest, map[string]interface{}{"code": http.StatusBadRequest, "msg": "cluster is installing, can't update config"})
				return
			}
		}
	}
	var req *model.GlobalConfigs
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	if len(req.GatewayNodes) == 0 {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": http.StatusBadRequest, "msg": "please select gatenode"})
		return
	}
	if err := cc.clusterCase.GlobalConfigs().UpdateGlobalConfig(req); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success"})
}

// Install install
func (cc *ClusterController) Install(c *gin.Context) {
	if err := cc.clusterCase.Install().Install(); err != nil { // TODO fanyangyang can't download rainbond file filter and return download URL
		if downloadError, ok := err.(*customerror.DownLoadError); ok {
			c.JSON(http.StatusOK, map[string]interface{}{"code": downloadError.Code, "msg": downloadError.Msg})
		} else if downloadingError, ok := err.(*customerror.DownloadingError); ok {
			c.JSON(http.StatusOK, map[string]interface{}{"code": downloadingError.Code, "msg": downloadingError.Msg})
		} else {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success"})
}

// InstallStatus install status
func (cc *ClusterController) InstallStatus(c *gin.Context) {
	status, err := cc.clusterCase.Install().InstallStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": status})
}

// Components components status
func (cc *ClusterController) Components(c *gin.Context) {
	componseInfos, err := cc.clusterCase.Components().List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": componseInfos})
}

// SingleComponent single component
func (cc *ClusterController) SingleComponent(c *gin.Context) {
	name := c.Param("name")
	name = strings.TrimSpace(name)
	if name == "" {
		cc.Components(c) // TODO fanyangyang need for test TODO: WHY?
		return
	}
	componseInfos, err := cc.clusterCase.Components().Get(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": componseInfos})
}
