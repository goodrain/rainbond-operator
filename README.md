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
  - 支持组件级别的资源配置、副本数、亲和性和污点容忍等设置

## 组件调度配置

`RbdComponent.spec.tolerations` 支持 Kubernetes 标准的污点容忍配置。非空列表会覆盖组件默认的 tolerations；未配置或设置为空列表时使用组件原有默认值。Operator 会将配置应用到组件生成的 Deployment、StatefulSet、DaemonSet 和 Job 的 Pod 模板中；已有 Job 仍遵循原有的仅创建逻辑。

升级时需先更新 `config/crd/bases/rainbond.io_rbdcomponents.yaml` 中的 CRD，再部署支持该字段的 Operator 镜像，否则字段可能被旧 CRD 丢弃或无法下发到工作负载。

例如，允许 `rbd-api` 调度到已 cordon 的节点：

```bash
kubectl -n rbd-system patch rbdcomponent rbd-api --type=merge -p '{"spec":{"tolerations":[{"key":"node.kubernetes.io/unschedulable","operator":"Exists","effect":"NoSchedule"}]}}'
```

仍需配合 `spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution` 限定目标节点；tolerations 本身不指定节点。`minio` 组件同样支持 `spec.affinity` 和 `spec.tolerations`。清除自定义 tolerations 可将该字段设为 `null`，恢复组件默认值。

`rainbond-operator` 自身不由 RbdComponent 管理，其调度配置需要在自身 Deployment 或安装配置中设置。
