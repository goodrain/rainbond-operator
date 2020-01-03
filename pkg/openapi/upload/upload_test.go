package upload

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestController_Upload(t *testing.T) {
	c := gin.Default()
	c.POST("/uploads", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			fmt.Println(err)
			c.String(400, "formFile error : +s", err.Error())
		}
		fmt.Println(file.Filename)

		// 上传文件至指定目录
		c.SaveUploadedFile(file, "/tmp/"+file.Filename)

		c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})
	c.Run()
}
