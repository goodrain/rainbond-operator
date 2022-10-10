package check_sqllite

import (
	"os"
)

func IsSQLLite() bool {
	if os.Getenv("IS_SQLLITE") != "" {
		return true
	}
	return false
}
