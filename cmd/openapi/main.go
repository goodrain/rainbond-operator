package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	clusterCtrl "github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/controller"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	uctrl "github.com/GLYASAI/rainbond-operator/pkg/openapi/user/controller"
	urepo "github.com/GLYASAI/rainbond-operator/pkg/openapi/user/repositry"
	uucase "github.com/GLYASAI/rainbond-operator/pkg/openapi/user/usecase"
)

var log = logf.Log.WithName("openapi")

func main() {
	db, _ := gorm.Open("sqlite3", "/tmp/gorm.db") // TODO hrh: data path and handle error
	defer db.Close()

	db.AutoMigrate(model.User{})
	db.Create(&model.User{Username: "admin", Password: "admin"})

	userRepo := urepo.NewSqlite3UserRepository(db)
	userRepo.CreateIfNotExist(&model.User{Username: "admin", Password: "admin"})
	userUcase := uucase.NewUserUsecase(userRepo, "my-secret-key")

	restConfig, err := clientcmd.BuildConfigFromFlags("", "/Users/fanyangyang/Documents/company/goodrain/remote/192.168.2.200/admin.kubeconfig") // TODO fanyangyang kubernetes correct?
	if err != nil {
		log.Error(err, fmt.Sprintf("create kubernetes rest config error: %s", err.Error()))
		return
	}
	normalClientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Error(err, fmt.Sprintf("create normal kubernetes clientset error: %s", err.Error()))
		return
	}
	rbdClientset, err := versioned.NewForConfig(restConfig)
	if err != nil {
		log.Error(err, fmt.Sprintf("create rbd kubernetes clientset error: %s", err.Error()))
		return
	}
	namespace := "rbd-system"
	configName := "rbd-globalconfig"
	etcdSecretName := "rbd-etcd-secret"
	archiveFilePath := "/tmp/rainbond.tar"
	clusterCase := cluster.NewClusterCase(namespace, configName, etcdSecretName, archiveFilePath, normalClientset, rbdClientset)

	r := gin.Default()
	uctrl.NewUserController(r, userUcase)
	clusterCtrl.NewClusterController(r, clusterCase)
	// upload.NewUploadController(r, archiveFilePath)
	r.POST("/uploads", func(c *gin.Context) { // upload can't work successfully in upload.NewUploadController(r, archiveFilePath)
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
	})

	go r.Run() // listen and serve on 0.0.0.0:8080

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-term:
		log.Info("Received signal", s.String(), "exiting gracefully.")
	}
	log.Info("See you next time!")
}
