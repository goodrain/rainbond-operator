package checkSqllite

import (
	"os"
)

//IsSQLLite -
func IsSQLLite() bool {
	if os.Getenv("IS_SQLLITE") != "" {
		return true
	}
	return false
}
