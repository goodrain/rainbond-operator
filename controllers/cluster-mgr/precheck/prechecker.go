/*
precheck 包为 Operator 提供了安装平台集群所需的环境和参数的预检查功能。

该包中的 `PreChecker` 接口定义了一个通用的预检查器，要求实现该接口的结构体必须提供 `Check` 方法，以检查特定的环境或参数，并返回一个表示检查结果的 `RainbondClusterCondition`。

主要组件：
- `PreChecker`：一个接口，定义了检查 Rainbond 集群安装所需环境和参数的通用方法 `Check`。该方法返回一个 `RainbondClusterCondition`，用于指示检查是否成功。
*/

package precheck

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
)

// PreChecker checks the environment and parameters required to install the rainbond cluster
type PreChecker interface {
	Check() rainbondv1alpha1.RainbondClusterCondition
}
