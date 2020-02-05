package upload

import (
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/util/corsutil"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path"
)

// Controller upload controller
type Controller struct {
	archiveFilePath string
}

var corsMidle = func(f gin.HandlerFunc) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		corsutil.SetCORS(ctx)
		f(ctx)
	})
}

// NewUploadController creates a new k8s controller
func NewUploadController(g *gin.Engine, archiveFilePath string) {
	u := &Controller{archiveFilePath: archiveFilePath}
	g.POST("/uploads", corsMidle(u.Upload))
}

// Upload upload file
func (u *Controller) Upload(c *gin.Context) {
	// 单文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, map[string]interface{}{"msg": err.Error})
		return
	}
	fmt.Println(file.Filename)

	// TODO: use mvc
	if err := os.MkdirAll(path.Dir(u.archiveFilePath), os.ModePerm); err != nil {
		c.JSON(400, map[string]interface{}{"msg": err.Error()})
		return
	}

	// 上传文件至指定目录
	if err := c.SaveUploadedFile(file, u.archiveFilePath); err != nil {
		c.JSON(400, map[string]interface{}{"msg": err.Error()})
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
}
