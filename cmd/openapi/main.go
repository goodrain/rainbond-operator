package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
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
	archiveFilePath = "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz"
)

// APIServer api server
var cfg *option.Config

func init() {
	cfg = &option.Config{}
	cfg.AddFlags(pflag.CommandLine)
	pflag.Parse()

	restConfig := k8sutil.MustNewKubeConfig(cfg.KubeconfigPath)
	cfg.KubeClient = kubernetes.NewForConfigOrDie(restConfig)
	cfg.RainbondKubeClient = versioned.NewForConfigOrDie(restConfig)
}

func main() {
	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	db, _ := gorm.Open("sqlite3", "/tmp/gorm.db") // TODO hrh: data path and handle error
	defer db.Close()

	db.AutoMigrate(model.User{})
	db.Create(&model.User{Username: "admin", Password: "admin"})

	r := gin.Default()

	userRepo := urepo.NewSqlite3UserRepository(db)
	userRepo.CreateIfNotExist(&model.User{Username: "admin", Password: "admin"})
	userUcase := uucase.NewUserUsecase(userRepo, "my-secret-key")
	uctrl.NewUserController(r, userUcase)
	clusterUcase := cluster.NewClusterCase(cfg)
	clusterCtrl.NewClusterController(r, clusterUcase)
	upload.NewUploadController(r, archiveFilePath)

	go r.Run() // listen and serve on 0.0.0.0:8080

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-term:
		log.Info("Received signal", s.String(), "exiting gracefully.")
	}
	log.Info("See you next time!")
}
