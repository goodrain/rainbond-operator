package checksqllite

import (
	"os"
)

// IS_SQLLITE is true if the database is a sqlite database
func IsSQLLite() bool {
	if os.Getenv("IS_SQLLITE") != "" {
		return true
	}
	return false
}
