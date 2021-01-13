package precheck

import (
	"database/sql"
	"fmt"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type database struct {
	typ3 rainbondv1alpha1.RainbondClusterConditionType
	db   *rainbondv1alpha1.Database
}

// NewDatabasePrechecker creates a new prechecker.
func NewDatabasePrechecker(typ3 rainbondv1alpha1.RainbondClusterConditionType, db *rainbondv1alpha1.Database) PreChecker {
	return &database{
		typ3: typ3,
		db:   db,
	}
}

func (d *database) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              d.typ3,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}
	err := d.check(d.db)
	if err != nil {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "DatabaseFailed"
		condition.Message = err.Error()
	}
	return condition
}

func (d *database) check(db *rainbondv1alpha1.Database) error {
	db2, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", db.Username, db.Password, db.Host, db.Port, db.Name))
	if err != nil {
		return err
	}
	defer db2.Close()

	err = db2.Ping()
	if err != nil {
		return err
	}

	return nil
}
