package check_sqllite

import (
	"github.com/sirupsen/logrus"
	"os"
)

func IsSQLLite() bool {
	if os.Getenv("IS_SQLLITE") != "" {
		logrus.Info("IS SQLLITE")
		return true
	}
	logrus.Info("IS MYSQL")
	return false
}
