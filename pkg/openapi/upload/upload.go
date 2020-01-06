package upload

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("openapi/uploads")
var archiveFilePath = "/tmp/rainbond.tar"

func init() {
	if os.Getenv("RBD_ARCHIVE") != "" {
		archiveFilePath = os.Getenv("RBD_ARCHIVE")
	}
}

// Controller upload controller
type Controller struct {
	archiveFilePath string
}

// NewUploadController creates a new k8s controller
func NewUploadController(g *gin.Engine, archiveFilePath string) {
	u := &Controller{archiveFilePath: archiveFilePath}
	g.POST("/uploads", u.Upload)
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

	// 上传文件至指定目录
	if err := c.SaveUploadedFile(file, archiveFilePath); err != nil {
		c.JSON(400, map[string]interface{}{"msg": err.Error})
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
}
