package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/openapi/middleware"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
	"github.com/goodrain/rainbond-operator/pkg/util/corsutil"
	"github.com/goodrain/rainbond-operator/pkg/util/ginutil"

	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
)

var log = logf.Log.WithName("cluster_controller")

// ClusterController k8s controller
type ClusterController struct {
	clusterUcase cluster.IClusterUcase
}

var corsMidle = func(f gin.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		corsutil.SetCORS(ctx)
		f(ctx)
	}
}

// NewClusterController creates a new k8s controller
func NewClusterController(g *gin.Engine, cfg *option.Config, clusterCase cluster.IClusterUcase, userRepo user.Repository) {
	u := &ClusterController{clusterUcase: clusterCase}

	clusterEngine := g.Group("/cluster")
	clusterEngine.Use(middleware.Authenticate(cfg.JWTSecretKey, cfg.JWTExpTime, userRepo))

	clusterEngine.GET("/status", corsMidle(u.ClusterStatus))
	clusterEngine.GET("/status-info", corsMidle(u.ClusterStatusInfo))
	clusterEngine.POST("/init", corsMidle(u.ClusterInit))
	clusterEngine.GET("/nodes", corsMidle(u.ClusterNodes))

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
	status, err := cc.clusterUcase.Cluster().Status()
	if err != nil {
		ginutil.JSON(c, status, err)
		return
	}
	ginutil.JSON(c, status, nil)
}

// ClusterInit cluster init
func (cc *ClusterController) ClusterInit(c *gin.Context) {
	err := cc.clusterUcase.Cluster().Init()
	if err != nil {
		c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusInternalServerError, "msg": "内部错误，请联系社区帮助"})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success"})
}

// ClusterStatusInfo returns the cluster information from rainbondcluster.
func (cc *ClusterController) ClusterStatusInfo(c *gin.Context) {
	info, err := cc.clusterUcase.Cluster().StatusInfo()
	ginutil.JSON(c, info, err)
}

// ClusterNodes returns a list of v1.K8sNode
func (cc *ClusterController) ClusterNodes(c *gin.Context) {
	query := c.Query("query")
	runGateway, _ := strconv.ParseBool(c.Query("rungateway"))
	nodes := cc.clusterUcase.Cluster().ClusterNodes(query, runGateway)
	ginutil.JSON(c, nodes, nil)
}

// Configs get cluster config info
func (cc *ClusterController) Configs(c *gin.Context) {
	configs, err := cc.clusterUcase.GlobalConfigs().GlobalConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": configs})
}

// UpdateConfig update cluster config info
func (cc *ClusterController) UpdateConfig(c *gin.Context) {
	reqLogger := log.WithName("UpdateConfig")
	var req v1.GlobalConfigs
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.V(4).Info(fmt.Sprintf("bad request: %v", err))
		ginutil.JSON(c, err, bcode.BadRequest)
		return
	}

	// check if the given nodes are valid.
	{
		validNodes, invalidNodes := cc.clusterUcase.Cluster().CompleteNodes(req.NodesForGateways, true)
		if len(invalidNodes) > 0 {
			ginutil.JSON(c, invalidNodes, bcode.ErrInvalidNodes)
			return
		}
		req.NodesForGateways = validNodes
	}
	{
		validNodes, invalidNodes := cc.clusterUcase.Cluster().CompleteNodes(req.NodesForChaos, false)
		if len(invalidNodes) > 0 {
			ginutil.JSON(c, invalidNodes, bcode.ErrInvalidNodes)
			return
		}
		req.NodesForChaos = validNodes
	}

	data, err := cc.clusterUcase.Install().InstallStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	for _, status := range data.StatusList {
		if status.StepName == "step_setting" && status.Status != "status_finished" { // TODO fanyangyang
			c.JSON(http.StatusBadRequest, map[string]interface{}{"code": http.StatusBadRequest, "msg": "cluster is installing, can't update config"})
			return
		}
	}

	if err := cc.clusterUcase.GlobalConfigs().UpdateGlobalConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}

	ginutil.JSON(c, nil, nil)
}

// Address address
func (cc *ClusterController) Address(c *gin.Context) {
	data, err := cc.clusterUcase.GlobalConfigs().Address()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}

// Uninstall reset cluster
func (cc *ClusterController) Uninstall(c *gin.Context) {
	err := cc.clusterUcase.Cluster().UnInstall()
	if err != nil {
		log.Error(err, "uninstall cluster")
		ginutil.JSON(c, nil, err)
		return
	}
	ginutil.JSON(c, nil, nil)
}

// Install install
func (cc *ClusterController) Install(c *gin.Context) {
	reqLogger := log.WithName("UpdateConfig")
	var req v1.ClusterInstallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.V(4).Info(fmt.Sprintf("bad request: %v", err))
		ginutil.JSON(c, nil, bcode.BadRequest)
		return
	}

	if err := cc.clusterUcase.Install().Install(&req); err != nil {
		log.Error(err, "install error")
		ginutil.JSON(c, nil, err)
		return
	}
	ginutil.JSON(c, nil, nil)
}

// InstallStatus install status
func (cc *ClusterController) InstallStatus(c *gin.Context) {
	data, err := cc.clusterUcase.Install().InstallStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": http.StatusInternalServerError, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}

// Components components status
func (cc *ClusterController) Components(c *gin.Context) {
	isInit, _ := strconv.ParseBool(c.Query("isInit"))

	componseInfos, err := cc.clusterUcase.Components().List(isInit)
	if err != nil {
		log.Error(err, "list components")
		ginutil.JSON(c, nil, err)
		return
	}

	ginutil.JSON(c, componseInfos, nil)
}

// SingleComponent single component
func (cc *ClusterController) SingleComponent(c *gin.Context) {
	name := c.Param("name")
	cpn, err := cc.clusterUcase.Components().Get(name)
	if err != nil {
		log.Info(fmt.Sprintf("get rbdcomponent: %v", err))
		ginutil.JSON(c, nil, err)
		return
	}

	ginutil.JSON(c, cpn, nil)
}
