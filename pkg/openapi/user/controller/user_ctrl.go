package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user/usecase"
	"github.com/goodrain/rainbond-operator/pkg/util/corsutil"
)

type UserController struct {
	userUcase user.Usecase
}

var corsMidle = func(f gin.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		corsutil.SetCORS(ctx)
		f(ctx)
	}
}

// NewUserController creates a new UserController
func NewUserController(g *gin.Engine, userUcase user.Usecase) {
	u := &UserController{
		userUcase: userUcase,
	}

	userEngine := g.Group("/user")
	userEngine.POST("/login", corsMidle(u.Login))
	userEngine.POST("/generate", corsMidle(u.Generate))
	userEngine.GET("/generate", corsMidle(u.IsGenerated))
}

// IsGenerated -
func (u *UserController) IsGenerated(c *gin.Context) {
	ok, err := u.userUcase.IsGenerated()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": bcode.ServerErr.Code(), "msg": bcode.ServerErr.Msg()})
		return
	}

	data := map[string]interface{}{"answer": ok}
	// just only the first time show admin username and password
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}

// Generate -
func (u *UserController) Generate(c *gin.Context) {
	userInfo, err := u.userUcase.GenerateUser()
	if err != nil {
		if err == bcode.DoNotAllowGenerateAdmin {
			c.JSON(http.StatusBadRequest, map[string]interface{}{"code": bcode.DoNotAllowGenerateAdmin.Code(), "msg": bcode.DoNotAllowGenerateAdmin.Msg()})
			return
		}
		logrus.Debugf("generate administrator error: %s", err.Error())
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": bcode.ErrGenerateAdmin.Code(), "msg": bcode.ErrGenerateAdmin.Msg()})
		return
	}
	data := map[string]interface{}{
		"username": userInfo.Username,
		"password": userInfo.Password,
	}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}

// Login -
func (u *UserController) Login(c *gin.Context) {
	var req model.User
	if err := c.ShouldBind(&req); err != nil {
		logrus.Errorf("parameter error: %s", err.Error())
		c.JSON(http.StatusBadRequest, map[string]interface{}{"code": http.StatusBadRequest, "msg": "用户名或密码错误"})
		return
	}

	token, err := u.userUcase.Login(req.Username, req.Password)
	if err != nil {
		if err == usecase.UserNotFound {
			logrus.Errorf("user not found: %s", err.Error())
			c.JSON(http.StatusNotFound, map[string]interface{}{"code": bcode.UserNotFound.Code(), "msg": "用户名或密码错误"})
			return
		}
		if err == bcode.UserPasswordInCorrect {
			c.JSON(http.StatusBadRequest, map[string]interface{}{"code": bcode.UserPasswordInCorrect.Code(), "msg": bcode.UserPasswordInCorrect.Msg()})
			return
		}
		logrus.Errorf("login failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"code": bcode.ServerErr.Code(), "msg": bcode.ServerErr.Msg()})
		return
	}

	data := map[string]interface{}{"token": token}
	c.JSON(http.StatusOK, map[string]interface{}{"code": http.StatusOK, "msg": "success", "data": data})
}
