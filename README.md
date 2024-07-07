# rainbond-operator


| Rainbond-operator version | Rainbond version | Status      |
| ------------------------- | ---------------- | ----------- |
| release-1.0               | 5.1.11           | Release     |
| release-1.1               | 5.2.1            | Release     |
| 2.0.0                     | 5.3.0            | Release     |
| 2.0.1                     | 5.3.1            | Release     |
| 2.0.2                     | 5.3.2            | Release     |
| 2.0.3                     | 5.3.3            | Release     |
| 2.1.0                     | 5.4.0            | Release     |
| 2.1.1                     | 5.4.1            | Release     |
| 2.2.0                     | 5.5.0            | Release     |
| 2.3.0                     | 5.6.0            | Release     |


- api 定义资源的
- chart operator 自己的chart包
- config -config目录内有介绍
- controllers operator 的主要业务实现
  - rbdcomponent_controller.go rbdcomponent 资源创建后的处理逻辑代码
  - rainbondvolume_controller.go rainbondvolume 资源创建后的处理逻辑代码
  - rainbondcluster_controller.go rainbondcluster 资源创建后的处理逻辑代码
- util 存放了一些公共方法
  - check-sqllite 检查sqlite
  - downloadutil 下载文件相关代码
  - containerutil 获取容器运行时相关代码
  - initcontainerd containerd 运行时处理相关代码
  - logutil 日志客户端获取相关代码
  - k8sutil k8s 客户端获取相关代码