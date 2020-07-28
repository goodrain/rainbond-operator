package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/goodrain/rainbond-operator/cmd/openapi/config"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	clusterCtrl "github.com/goodrain/rainbond-operator/pkg/openapi/cluster/controller"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/repository"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/store"
	cucase "github.com/goodrain/rainbond-operator/pkg/openapi/cluster/usecase"
	"github.com/goodrain/rainbond-operator/pkg/openapi/nodestore"
	upgradec "github.com/goodrain/rainbond-operator/pkg/openapi/upgrade/controller"
	upgradeu "github.com/goodrain/rainbond-operator/pkg/openapi/upgrade/usecase"
	uctrl "github.com/goodrain/rainbond-operator/pkg/openapi/user/controller"
	uucase "github.com/goodrain/rainbond-operator/pkg/openapi/user/usecase"
	"github.com/goodrain/rainbond-operator/pkg/util/corsutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("main")

// APIServer api server
var cfg *config.Config

func init() {
	cfg = &config.Config{}
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
	if err := storer.Start(); err != nil {
		log.Error(err, "start component storer")
		os.Exit(1)
	}
	defer storer.Stop()

	nodestorer := nodestore.New(ctx, clientset)
	if err := nodestorer.Start(); err != nil {
		log.Error(err, "start node storer")
		os.Exit(1)
	}
	defer nodestorer.Stop()

	// router
	r := gin.Default()
	r.OPTIONS("/*path", corsMidle(func(ctx *gin.Context) {}))
	r.Use(static.Serve("/", static.LocalFile("/app/ui", true)))

	// user
	userUcase := uucase.NewUserUsecase(nil, "my-secret-key")
	uctrl.NewUserController(r, userUcase)

	// cluster
	repo := repository.NewClusterRepo(cfg.InitPath)
	clusterUcase := cucase.NewClusterCase(cfg, repo, rainbondKubeClient, nodestorer, storer)
	clusterCtrl.NewClusterController(r, clusterUcase)

	// upgrade
	upgradeUcase := upgradeu.NewUpgradeUsecase(rainbondKubeClient, cfg.VersionDir, cfg.Namespace, cfg.ClusterName)
	upgradec.NewUpgradeController(r, upgradeUcase)

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
	return func(ctx *gin.Context) {
		corsutil.SetCORS(ctx)
		f(ctx)
	}
}
