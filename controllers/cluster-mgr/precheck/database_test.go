package precheck_test

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers/cluster-mgr/precheck"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestDatabasePreChecker(t *testing.T) {
	db := &rainbondv1alpha1.Database{
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "foo",
		Password: "bar",
		Name:     "foobar",
	}

	preChecker := precheck.NewDatabasePrechecker(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseRegion, db)

	condition := preChecker.Check()

	assert.Equal(t, rainbondv1alpha1.RainbondClusterConditionType(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseRegion), condition.Type)
	assert.Equal(t, corev1.ConditionFalse, condition.Status)
	assert.Equal(t, "DatabaseFailed", condition.Reason)
}
