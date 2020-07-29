package ginutil

import (
	"github.com/gin-gonic/gin"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/sirupsen/logrus"
)

// Result represents a response for restful api.
type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// JSON -
func JSON(c *gin.Context, data interface{}, err error) {
	bc := bcode.Err2Coder(err)
	if bc == bcode.ServerErr {
		logrus.Error(err)
	}
	result := &Result{
		Code: bc.Code(),
		Msg:  bc.Msg(),
		Data: data,
	}
	c.JSON(bc.Status(), result)
}
