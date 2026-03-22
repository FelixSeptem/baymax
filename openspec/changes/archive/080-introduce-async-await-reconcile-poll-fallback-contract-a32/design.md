## Context

A31 已经引入 `awaiting_report` 生命周期、超时收敛和晚到回报治理，但终态收敛仍主要依赖 report callback 主链路。  
在 callback 丢失、抖动或暂时不可达时，任务会依赖 timeout 收敛，缺少“主动对账”后备通道，导致可恢复场景被误伤为超时失败。

当前代码已经具备：
- 调度域的 `awaiting_report` 状态与 timeout 语义；
- A2A status/result 查询能力；
- Task Board 读接口与诊断聚合扩展机制；
- `env > file > default` 配置优先级与热更新回滚框架。

A32 目标是在不引入平台化依赖的前提下，补齐 async-await 的自愈闭环：callback 之外增加 poll reconcile fallback，并保证确定性终态与可回放契约。

## Goals / Non-Goals

**Goals:**
- 为 `awaiting_report` 任务提供可配置的周期性 poll fallback 收敛能力。
- 固化双来源终态并发规则：`first_terminal_wins + record_conflict`。
- 持久化远端关联键并在 Task Board 暴露最小可观测字段（终态来源、远端关联）。
- 将 reconcile 配置和诊断纳入统一 runtime/config + runtime/diagnostics 契约，维持兼容窗口语义。
- 把 reconcile suites 纳入 shared multi-agent gate 阻断路径。

**Non-Goals:**
- 不引入 MQ、控制面、平台化任务编排能力。
- 不承诺 exactly-once，仅保持 at-least-once + idempotent convergence。
- 不改变既有 Run/Stream 对外接口形态，仅在语义层增强收敛路径。
- 不新增写操作型 Task Board API（仅扩展只读观测字段）。

## Decisions

### 1) 对账触发范围仅限 `awaiting_report`
- 决策：reconcile poll 仅扫描 `awaiting_report` 任务，不触碰 `queued/running/terminal` 任务。
- 原因：保持职责单一，避免对 scheduler 主执行路径引入额外扰动。
- 备选：全状态轮询。拒绝原因：成本高、收益低，且增加竞态面。

### 2) 配置默认关闭，按推荐值提供开关与节流
- 决策：新增 `scheduler.async_await.reconcile.*`：
  - `enabled=false`
  - `interval=5s`
  - `batch_size=64`
  - `jitter_ratio=0.2`
  - `not_found_policy=keep_until_timeout`
- 原因：默认无行为变更，便于灰度启用与回归对比。
- 备选：默认开启。拒绝原因：会在未评估环境直接放大远端查询负载。

### 3) 双来源终态采用 `first_terminal_wins + record_conflict`
- 决策：callback 与 poll 均可提交终态，但第一个终态写入获胜；后续冲突仅记录诊断，不翻转业务终态。
- 原因：保证可回放确定性，避免“终态来回翻转”。
- 备选：按最新事件覆盖。拒绝原因：时序不稳定，会破坏幂等和审计一致性。

### 4) `not_found` 采用 `keep_until_timeout`
- 决策：poll 返回 `not_found` 不立即终态，保持 `awaiting_report` 直到 `report_timeout`。
- 原因：兼容远端最终一致和延迟建档场景，减少误判失败。
- 备选：连续 N 次 `not_found` 即失败。拒绝原因：引入额外阈值复杂度且语义更脆弱。

### 5) 终态提交统一走既有 commit contract
- 决策：poll fallback 的终态提交复用现有 terminal commit/idempotency 路径。
- 原因：避免新建并行提交语义，减少分叉和回归面。
- 备选：新增 poll 专用提交入口。拒绝原因：会造成双写逻辑和规则漂移风险。

### 6) Task Board 只读扩展最小字段
- 决策：查询结果增加最小观测字段：
  - `resolution_source`（`callback|reconcile_poll|timeout`）
  - `remote_task_id`（或语义等价远端关联键）
  - `terminal_conflict_recorded`（可空）
- 原因：满足定位和审计，不扩大控制面。
- 备选：暴露完整内部对账事件流。拒绝原因：信息噪音高且耦合内部实现细节。

### 7) 诊断字段采用 additive 扩展并要求回放幂等
- 决策：新增 reconcile 相关聚合字段（命名可实现侧细化）并遵循 `additive + nullable + default`。
- 原因：兼容历史消费者，同时提供回归信号。
- 备选：复用现有 async 字段。拒绝原因：语义混叠，难以区分 callback 与 poll 收敛贡献。

### 8) Shared gate 增加 reconcile 阻断套件
- 决策：在已有 multi-agent shared gate 中纳入 reconcile 套件，不新增分裂式 gate。
- 原因：保持单一主干门禁入口，降低维护成本。
- 备选：独立 reconcile gate。拒绝原因：门禁碎片化，容易出现漏跑。

## Risks / Trade-offs

- [Risk] 开启 poll 可能增加远端负载与抖动放大  
  → Mitigation: 默认关闭 + `batch_size` + `interval` + `jitter_ratio` 组合限流，并支持动态调参回滚。

- [Risk] callback 与 poll 并发提交引入冲突记录激增  
  → Mitigation: 固化 `first_terminal_wins`，冲突只记录不覆写；增加冲突诊断计数用于治理。

- [Risk] `not_found` 长时间存在可能推迟故障暴露  
  → Mitigation: 与 `report_timeout` 绑定，超时仍按既有 timeout 终态收敛。

- [Risk] Task Board 新字段引发旧消费方解析差异  
  → Mitigation: 采用可空 additive 字段，不改变既有字段语义和分页游标契约。

## Migration Plan

1. 以前置 A31 生命周期能力为基线，在 scheduler/composer 接口中固化远端关联键持久化。
2. 实现 reconcile 扫描与 poll 执行器（受 `scheduler.async_await.reconcile.enabled` 控制）。
3. 将 poll 终态接入既有 terminal commit contract，并落地冲突记录语义。
4. 扩展 Task Board 查询模型与 diagnostics 聚合字段，保持 compatibility window。
5. 补齐契约测试：callback/poll 竞态、not_found、timeout、Run/Stream 等价、memory/file parity、replay 幂等。
6. 在 shared multi-agent gate 中接入 reconcile suites，并更新主干契约映射文档。

回滚策略：
- 若线上出现负载或语义风险，可先关闭 `scheduler.async_await.reconcile.enabled`，回退到 A31 callback+timeout 路径；
- 保留字段与测试资产，不回滚兼容性 additive 输出。

## Open Questions

当前无阻塞项，按推荐值冻结：
- `enabled=false`
- `interval=5s`
- `batch_size=64`
- `jitter_ratio=0.2`
- `not_found_policy=keep_until_timeout`

