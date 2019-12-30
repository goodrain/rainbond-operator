package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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

	r := gin.Default()
	uctrl.NewUserController(r, userUcase)

	go r.Run() // listen and serve on 0.0.0.0:8080

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-term:
		log.Info("Received signal", s.String(), "exiting gracefully.")
	}
	log.Info("See you next time!")
}
