package controller

import (
	"net/http"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/openapi/customerror"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/util/corsutil"

	"github.com/gin-gonic/gin"
)

var log = logf.Log.WithName("usecase_cluster")

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
	clusterEngine.GET("/status", corsMidle(u.ClusterStatus))
	clusterEngine.POST("/init", corsMidle(u.ClusterInit))

	clusterEngine.GET("/configs", corsMidle(u.Configs))
	clusterEngine.PUT("/configs", corsMidle(u.UpdateConfig))

	clusterEngine.GET("/address", corsMidle(u.Address))

	clusterEngine.DELETE("/uninstall", corsMidle(u.Uninstall))

	// install
	clusterEngine.POST("/install", corsMidle(u.Install))
	clusterEngine.GET("/install/status", corsMidle(u.InstallStatus))

	// componse
	clusterEngine.GET("/components", corsMidle(u.Components))
	clusterEngine.GET("/components/:name", corsMidle(u.SingleComponent))
}

// ClusterStatus cluster status
func (cc *ClusterController) ClusterStatus(c *gin.Context) {
	status, err := cc.clusterCase.Cluster().Status()
	if err != nil {
		c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "内部错误，请联系社区帮助"})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": status})
}

// ClusterInit cluster init
func (cc *ClusterController) ClusterInit(c *gin.Context) {
	err := cc.clusterCase.Cluster().Init()
	if err != nil {
		if _, ok := err.(*customerror.CRAlreadyExistsError); ok {
			c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "uninstalling"})
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "内部错误，请联系社区帮助"})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success"})
}

// Configs get cluster config info
func (cc *ClusterController) Configs(c *gin.Context) {
	configs, err := cc.clusterCase.GlobalConfigs().GlobalConfigs() // if can't get cluster config, return empty config and ignore error
	if err != nil {
		c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "internal error, please contact rainbond for help"})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": configs})
}

// UpdateConfig update cluster config info
func (cc *ClusterController) UpdateConfig(c *gin.Context) {
	data, err := cc.clusterCase.Cluster().Status()
	if err != nil { // if get cluster status failed, return custom error and redirect to index page TODO fanyangyang
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	if data.FinalStatus != model.Setting && data.FinalStatus != model.SettingCompleted {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": http.StatusBadRequest, "msg": "can't setting cluster now"})
		return
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

// Address address
func (cc *ClusterController) Address(c *gin.Context) {
	data, err := cc.clusterCase.GlobalConfigs().Address()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}

// Uninstall reset cluster
func (cc *ClusterController) Uninstall(c *gin.Context) {
	err := cc.clusterCase.Cluster().UnInstall()
	if err != nil {
		c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "卸载出错，请联系社区帮助"})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success"})
}

// Install install
func (cc *ClusterController) Install(c *gin.Context) {
	data, err := cc.clusterCase.Cluster().Status()
	if err != nil { // if get cluster status failed, return custom error and redirect to index page TODO fanyangyang
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	if data.FinalStatus != model.SettingCompleted {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": http.StatusBadRequest, "msg": "can't setting cluster now"})
		return
	}
	if err := cc.clusterCase.Install().Install(); err != nil {
		log.Error(err, "install error")
		c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "内部错误，请联系社区帮助"})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success"})
}

// InstallStatus install status
func (cc *ClusterController) InstallStatus(c *gin.Context) {
	data, err := cc.clusterCase.Install().InstallStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}

// Components components status
func (cc *ClusterController) Components(c *gin.Context) {
	data := c.DefaultQuery("isInit", "false")
	isInit := false
	if data == "true" {
		isInit = true
	}

	componseInfos, err := cc.clusterCase.Components().List(isInit)
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
