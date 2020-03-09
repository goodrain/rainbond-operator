package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/store"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	clusterCtrl "github.com/goodrain/rainbond-operator/pkg/openapi/cluster/controller"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/repository"
	cucase "github.com/goodrain/rainbond-operator/pkg/openapi/cluster/usecase"
	"github.com/goodrain/rainbond-operator/pkg/openapi/nodestore"
	"github.com/goodrain/rainbond-operator/pkg/openapi/upload"
	uctrl "github.com/goodrain/rainbond-operator/pkg/openapi/user/controller"
	uucase "github.com/goodrain/rainbond-operator/pkg/openapi/user/usecase"
	"github.com/goodrain/rainbond-operator/pkg/util/corsutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
)

var log = logf.Log.WithName("main")

var (
	archiveFilePath = "/opt/rainbond/pkg/tgz/rainbond-pkg-V5.2-dev.tgz"
)

// APIServer api server
var cfg *option.Config

func init() {
	cfg = &option.Config{}
	cfg.AddFlags(pflag.CommandLine)
	pflag.Parse()
	cfg.SetLog()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	restConfig := k8sutil.MustNewKubeConfig(cfg.KubeconfigPath)
	cfg.RestConfig = restConfig
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	cfg.KubeClient = clientset
	rainbondKubeClient := versioned.NewForConfigOrDie(restConfig)
	cfg.RainbondKubeClient = rainbondKubeClient

	storer := store.NewStore(cfg.Namespace, rainbondKubeClient, clientset)
	storer.Start()
	defer storer.Stop()

	nodestorer := nodestore.New(ctx, clientset)
	if err := nodestorer.Start(); err != nil {
		log.Error(err, "start node storer")
		os.Exit(1)
	}
	defer nodestorer.Stop()

	repo := repository.NewClusterRepo(cfg.InitPath)

	r := gin.Default()
	r.OPTIONS("/*path", corsMidle(func(ctx *gin.Context) {}))
	r.Use(static.Serve("/", static.LocalFile("/app/ui", true)))

	userUcase := uucase.NewUserUsecase(nil, "my-secret-key")
	uctrl.NewUserController(r, userUcase)

	clusterUcase := cucase.NewClusterCase(cfg, repo, rainbondKubeClient, nodestorer, storer)
	clusterCtrl.NewClusterController(r, clusterUcase)

	upload.NewUploadController(r, archiveFilePath)

	logrus.Infof("api server listen %s", func() string {
		if port := os.Getenv("PORT"); port != "" {
			return ":" + port
		}
		return ":8080"
	}())
	go func() { _ = r.Run() }() // listen and serve on 0.0.0.0:8080

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	s := <-term
	logrus.Info("Received signal", s.String(), "exiting gracefully.")
	logrus.Info("See you next time!")
}

var corsMidle = func(f gin.HandlerFunc) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		corsutil.SetCORS(ctx)
		f(ctx)
	})
}
