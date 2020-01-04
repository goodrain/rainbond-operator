package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster"
	clusterCtrl "github.com/GLYASAI/rainbond-operator/pkg/openapi/cluster/controller"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/upload"
	uctrl "github.com/GLYASAI/rainbond-operator/pkg/openapi/user/controller"
	urepo "github.com/GLYASAI/rainbond-operator/pkg/openapi/user/repositry"
	uucase "github.com/GLYASAI/rainbond-operator/pkg/openapi/user/usecase"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
)

var log = logf.Log.WithName("openapi")

var (
	namespace       = "rbd-system"
	configName      = "rbd-globalconfig"
	etcdSecretName  = "rbd-etcd-secret"
	archiveFilePath = "/tmp/rainbond.tar"
)

func main() {
	db, _ := gorm.Open("sqlite3", "/tmp/gorm.db") // TODO hrh: data path and handle error
	defer db.Close()

	db.AutoMigrate(model.User{})
	db.Create(&model.User{Username: "admin", Password: "admin"})

	r := gin.Default()

	userRepo := urepo.NewSqlite3UserRepository(db)
	userRepo.CreateIfNotExist(&model.User{Username: "admin", Password: "admin"})
	userUcase := uucase.NewUserUsecase(userRepo, "my-secret-key")
	uctrl.NewUserController(r, userUcase)

	cfg := k8sutil.MustNewKubeConfig("/opt/rainbond/etc/kubernetes/kubecfg/172.20.0.11/admin.kubeconfig") // TODO: do not hardcode
	clientset := kubernetes.NewForConfigOrDie(cfg)
	rainbondClientset := versioned.NewForConfigOrDie(cfg)
	clusterUcase := cluster.NewClusterCase(namespace, configName, etcdSecretName, archiveFilePath, clientset, rainbondClientset)
	clusterCtrl.NewClusterController(r, clusterUcase)

	// TODO： move upload out of here
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
