/*
precheck 包为 Operator 提供了预检查功能。

该包包含各种预检查器，如 `database` 检查器，它确保在进行进一步操作之前数据库连接是有效的。

`database` 检查器通过在尝试打开连接后执行 ping 操作，来验证数据库的连通性和可访问性。

`NewDatabasePrechecker` 函数创建一个新的数据库预检查器实例，负责执行这些检查，并将结果作为 `RainbondClusterCondition` 返回。

主要组件：
- `database`：一个结构体，包含被检查的条件类型以及执行检查所需的数据库配置。
- `NewDatabasePrechecker`：一个函数，用于创建新的 `database` 预检查器。
- `Check`：`database` 结构体上的一个方法，执行实际的检查，并返回表示检查状态的 `RainbondClusterCondition`。
- `check`：一个辅助方法，尝试打开数据库连接并执行 ping 操作，如果过程中遇到任何错误则返回这些错误。
*/

package precheck

import (
	"database/sql"
	"fmt"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
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
