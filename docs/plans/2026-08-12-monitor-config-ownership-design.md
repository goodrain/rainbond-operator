# Monitor configuration ownership design

## 一、项目背景

### 1.1 项目架构

`rainbond-operator` creates and reconciles `rbd-system/rbd-monitor` and its
`prometheus-config` ConfigMap. The VM plugin and AI Engine plugin previously
modified that same ConfigMap and deleted the monitor Pod to make Prometheus
reload it.

### 1.2 现有基础

The operator's embedded Prometheus configuration already uses Kubernetes
service discovery for platform and GPU metrics. KubeVirt VM metrics and the
AI Engine `gpu-observer` metrics can use the same discovery mechanism. A
missing Service simply produces no targets, so neither job needs a runtime
writer in a plugin.

### 1.3 核心需求

There must be exactly one writer and restart controller for monitor
configuration:

- The operator owns `rbd-system/prometheus-config` and `rbd-monitor`.
- VM and AI Engine must neither read nor write that ConfigMap, and must never
  delete monitor Pods.
- The operator must roll the StatefulSet once only when its rendered monitor
  configuration changes.
- Existing KubeVirt and GPU-observer metric collection must continue to work.

## 二、用户旅程（MUST — 禁止跳过）

### 2.1 用户操作流程

- An administrator installs or upgrades the VM or AI Engine plugin. No extra
  UI setting is required for monitoring.
- When KubeVirt exposes Services labelled
  `prometheus.kubevirt.io=true` with a `metrics` endpoint, monitor discovers
  them automatically.
- When AI Engine creates the `gpu-observer` Service with a `metrics` endpoint,
  monitor discovers it automatically.
- The administrator observes monitor availability normally; plugin startup,
  service changes, and unrelated ConfigMap events do not restart monitor.

### 2.2 页面原型

- No new page, dialog, form, or UI entry is needed.
- The existing Prometheus target UI/API reflects discovered endpoints.

### 2.3 外部系统交互

- Prometheus uses Kubernetes API service/endpoints discovery with its existing
  service account permissions.
- No new webhook, callback, or third-party integration is introduced.

## 三、整体架构设计

### 3.1 系统架构图

```text
KubeVirt Service / AI gpu-observer Service
                 │ labels + named metrics endpoint
                 ▼
operator-owned prometheus.yml ──► rbd-monitor StatefulSet
                 ▲                       │
                 │                       ▼
          rainbond-operator         Prometheus discovery

VM and AI Engine: no ConfigMap writes and no monitor Pod deletion
```

### 3.2 核心流程

1. The operator renders the base configuration including KubeVirt and
   `gpu-observer` scrape jobs.
2. The operator calculates a SHA-256 checksum over the rendered configuration
   and stores it on the monitor Pod template.
3. A changed rendered configuration updates the ConfigMap and Pod template,
   producing one normal StatefulSet rollout.
4. An unchanged configuration produces no ConfigMap update and no Pod rollout.
5. Plugins install their own workloads only; Prometheus discovers their
   endpoints through the static jobs.

## 四、数据模型设计

### 4.1 新增数据库表

None.

### 4.2 数据关系

No API or database schema changes. The monitor Pod template gains an internal
configuration checksum annotation.

## 五、API设计

### 5.1 接口列表

No public API changes.

### 5.2 请求/响应结构

Not applicable.

## 六、核心实现设计

### 6.1 关键逻辑

- Add static `kubevirt-vm` and `gpu-observer` jobs to the operator-owned
  `config/prom/prometheus.yml`.
- Make the monitor ConfigMap fully operator-owned again; remove the temporary
  external-data bypass so configuration upgrades are not ignored.
- Hash the exact ConfigMap data used by monitor and set the checksum in the
  StatefulSet Pod template annotation. API-defaulted fields remain ignored by
  the existing statefulset comparison, while checksum changes are detected.
- Delete the VM Prometheus reconciler and the AI Engine monitor controller,
  their startup wiring, and their regression tests. No replacement runtime
  controller is added.

### 6.2 复用现有代码

- Reuse the operator's existing ConfigMap reconciliation and StatefulSet
  rolling-update behavior.
- Reuse existing Kubernetes service discovery patterns in Prometheus config.
- Reuse existing plugin install infrastructure; only remove monitor ownership.

## 七、实施计划

### 跨层覆盖检查（MUST）

- [x] Go (rainbond-operator): 需要 — own scrape config and trigger one rollout
  on a true config change.
- [x] Go (VM plugin): 需要 — remove shared ConfigMap/Pod management.
- [x] Go (AI Engine plugin): 需要 — remove shared ConfigMap watcher/writer and
  startup wiring.
- [x] Python (console): 不涉及 — no console API or orchestration changes.
- [x] React (rainbond-ui): 不涉及 — no user interaction changes.
- [x] Plugin frontend (enterprise-base): 不涉及 — backend-only behavior.
- [x] Plugin backend (plugin-template): 不涉及 — repository-local plugin
  backends change, not the shared template.

### Sprint 1: Remove competing owners

#### Task 1.1: Stop VM plugin monitor mutation

- 仓库：rainbond-plugins
- 文件：`rainbond-vm/backend/internal/reconciler/prometheus_config.go` and
  `rainbond-vm/backend/cmd/plugin/main.go`
- 实现内容：remove the reconciler and its startup; remove obsolete tests and
  unused configuration constants.
- 验收标准：VM backend has no access to `prometheus-config` or
  `rbd-monitor`; `make check` succeeds.

#### Task 1.2: Stop AI Engine monitor mutation

- 仓库：rainbond-ai-engine
- 文件：`backend/pkg/deployer/monitor.go`,
  `backend/pkg/deployer/monitor_controller.go`, tests, and
  `backend/cmd/plugin/main.go`
- 实现内容：remove ConfigMap mutation, monitor Pod deletion, informer, and
  startup calls; preserve unrelated user work in `main.go`.
- 验收标准：AI Engine has no monitor ConfigMap writer/controller; `make check`
  succeeds.

### Sprint 2: Centralize operator ownership

#### Task 2.1: Render plugin scrape jobs in operator configuration

- 仓库：rainbond-operator
- 文件：`config/prom/prometheus.yml`, `controllers/handler/monitor.go`,
  `controllers/handler/monitor_test.go`
- 实现内容：add static service-discovery jobs and a deterministic checksum
  annotation on the monitor Pod template.
- 验收标准：the generated ConfigMap contains both jobs; a config-content
  change changes only the Pod template checksum.

#### Task 2.2: Remove the temporary external ConfigMap bypass

- 仓库：rainbond-operator
- 文件：`controllers/component-mgr/component.go`,
  `controllers/component-mgr/component_test.go`
- 实现内容：remove the external-data annotation and data-copy bypass, retaining
  normal no-op update checks and ConfigMap additional-key preservation.
- 验收标准：operator updates a changed managed config key once and does not
  update an equivalent ConfigMap.

### Sprint 3: Cross-repository regression

#### Task 3.1: Verify source ownership and build artifacts

- 仓库：all three repositories
- 实现内容：run focused tests, complete test/vet/build gates, static searches
  for removed monitor write/delete paths, and diff review against pre-existing
  user changes.
- 验收标准：all commands pass and only intended files are staged per commit.

## 八、关键参考代码

| 功能 | 文件 | 说明 |
|---|---|---|
| Monitor resource creation | `controllers/handler/monitor.go` | ConfigMap and StatefulSet definitions |
| Resource no-op comparison | `controllers/component-mgr/component.go` | prevents needless API updates |
| VM competing writer | `rainbond-vm/backend/internal/reconciler/prometheus_config.go` | must be removed |
| AI competing writer | `rainbond-ai-engine/backend/pkg/deployer/monitor.go` | must be removed |
| AI event amplifier | `rainbond-ai-engine/backend/pkg/deployer/monitor_controller.go` | must be removed |
