# rainbond-operator

Rainbond-operator 是 [Rainbond](https://github.com/goodrain/rainbond) 的子项目，用于在 Kubernetes 集群中自动化安装、配置和管理 Rainbond 云原生应用管理平台。

## 项目简介

Rainbond-operator 基于 Kubernetes Operator 模式，提供声明式的方式来部署和管理 Rainbond 集群的各个组件，包括：

- **API 网关**：负责流量路由和负载均衡
- **应用构建**：支持源码构建和镜像构建
- **服务治理**：提供服务发现、配置管理等功能
- **监控告警**：集成 Prometheus 监控体系
- **存储管理**：支持多种存储后端

通过 CRD（Custom Resource Definitions）的方式，用户只需要定义期望的集群状态，operator 会自动处理复杂的安装和运维工作。

- **RainbondCluster** (`rainbondclusters.rainbond.io`)
  - 定义 Rainbond 集群的整体配置，包括版本、网关节点、构建节点分配等
  - 管理集群级别的全局设置和状态

- **RbdComponent** (`rbdcomponents.rainbond.io`)
  - 定义 Rainbond 各个组件的配置，如 API、Gateway、Worker、Chaos 等
  - 支持组件级别的资源配置、副本数、亲和性等设置
