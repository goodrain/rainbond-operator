package ginutil

import (
	"github.com/gin-gonic/gin"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"net/http"
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
	// http code from 100 to 599
	httpCode := http.StatusOK
	if bc.Code() >= 100 || bc.Code() <= 599 {
		httpCode = bc.Code()
	}
	result := Result{
		Code: bc.Code(),
		Msg:  bc.Msg(),
		Data: data,
	}
	c.JSON(httpCode, result)
}
