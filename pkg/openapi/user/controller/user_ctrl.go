package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user/usecase"
)

type UserController struct {
	userUcase user.Usecase
}

// NewUserController creates a new UserController
func NewUserController(g *gin.Engine, userUcase user.Usecase) {
	u := &UserController{
		userUcase: userUcase,
	}

	userEngine := g.Group("/user")
	userEngine.POST("/login", u.Login)
	userEngine.POST("/generate", u.Generate)
}

// Generate -
func (u *UserController) Generate(c *gin.Context) {
	userInfo, err := u.userUcase.GenerateUser()
	if err != nil {
		if err == bcode.DoNotAllowGenerateAdmin {
			c.JSON(http.StatusBadRequest, bcode.DoNotAllowGenerateAdmin)
			return
		}
		logrus.Debugf("generate administrator error: %s", err.Error())
		c.JSON(http.StatusInternalServerError, bcode.ErrGenerateAdmin)
		return
	}
	c.JSON(http.StatusOK, userInfo)
}

// Login -
func (u *UserController) Login(c *gin.Context) {
	var req model.User
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	token, err := u.userUcase.Login(req.Username, req.Password)
	if err != nil {
		if err == usecase.UserNotFound {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		if err == bcode.UserPasswordInCorrect {
			c.JSON(http.StatusBadRequest, bcode.UserPasswordInCorrect.Msg())
			return
		}
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(200, map[string]interface{}{"token": token})
}
