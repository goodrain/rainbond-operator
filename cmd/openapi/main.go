package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/GLYASAI/rainbond-operator/pkg/openapi/upload"

	clusterCtrl "github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/controller"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
var (
	namespace       = "rbd-system"
	configName      = "rbd-globalconfig"
	etcdSecretName  = "rbd-etcd-secret"
	archiveFilePath = "/tmp/rainbond.tar"
	restConfig      *rest.Config
	k8sMasterURL    = ""
	k8sConfigPath   = "/root/.kube/config"
)

func init() {
	if os.Getenv("RBD_NAMESPACE") != "" {
		namespace = os.Getenv("RBD_NAMESPACE")
	}
	if os.Getenv("RBD_GLOBALCONFIG") != "" {
		configName = os.Getenv("RBD_GLOBALCONFIG")
	}
	if os.Getenv("RBD_ETCD_SECRET") != "" {
		etcdSecretName = os.Getenv("RBD_ETCD_SECRET")
	}
	if os.Getenv("RBD_ARCHIVE") != "" {
		archiveFilePath = os.Getenv("RBD_ARCHIVE")
	}

	if os.Getenv("K8S_MASTER_URL") != "" {
		var err error
		if restConfig, err = clientcmd.BuildConfigFromFlags(os.Getenv("K8S_MASTER_URL"), ""); err != nil {
			log.Error(err, fmt.Sprintf("create kubernetes rest config error: %s", err.Error()))
			return
		}
	} else {
		if os.Getenv("K8S_CONFIG_PATH") != "" {
			k8sConfigPath = os.Getenv("K8S_CONFIG_PATH")
		}
		var err error
		if restConfig, err = clientcmd.BuildConfigFromFlags("", k8sConfigPath); err != nil {
			log.Error(err, fmt.Sprintf("create kubernetes rest config error: %s", err.Error()))
			return
		}
	}

}

func main() {
	db, _ := gorm.Open("sqlite3", "/tmp/gorm.db") // TODO hrh: data path and handle error
	defer db.Close()

	db.AutoMigrate(model.User{})
	db.Create(&model.User{Username: "admin", Password: "admin"})

	userRepo := urepo.NewSqlite3UserRepository(db)
	userRepo.CreateIfNotExist(&model.User{Username: "admin", Password: "admin"})
	userUcase := uucase.NewUserUsecase(userRepo, "my-secret-key")

	if restConfig == nil {
		log.Error(nil, "create kubernetes client error")
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
	clusterCase := cluster.NewClusterCase(namespace, configName, etcdSecretName, archiveFilePath, normalClientset, rbdClientset)

	r := gin.Default()
	uctrl.NewUserController(r, userUcase)
	clusterCtrl.NewClusterController(r, clusterCase)
	r.POST("/uploads", upload.Upload)

	go r.Run() // listen and serve on 0.0.0.0:8080

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-term:
		log.Info("Received signal", s.String(), "exiting gracefully.")
	}
	log.Info("See you next time!")
}
