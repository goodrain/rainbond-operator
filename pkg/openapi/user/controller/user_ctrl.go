package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
}

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
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(200, token)
}
