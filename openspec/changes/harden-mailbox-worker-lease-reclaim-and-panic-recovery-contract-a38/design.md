## Context

A36 已提供 mailbox lifecycle worker 的消费闭环与 reason taxonomy 基线，但 worker 仍缺少与 scheduler 同级的恢复治理语义：

- handler panic 可能中断 worker 执行路径，当前缺少明确 recover 策略。
- mailbox 缺少 stale in-flight reclaim 机制，异常中断后消息可能长期停留在 `in_flight`。
- 对长执行任务缺少 heartbeat 续租语义，无法区分“正常长任务”与“卡死任务”。

项目当前处于 0.x 收敛阶段，目标是把 mailbox worker 从“可运行”升级为“可恢复”，并保持 `library-first + contract-first` 边界。

## Goals / Non-Goals

**Goals:**
- 为 mailbox worker 增加 lease/reclaim/recover 契约语义，并提供保守默认值。
- 增加 `mailbox.worker` 新配置字段并纳入启动/热更新 fail-fast 与原子回滚。
- 扩展 lifecycle diagnostics 和 quality gate，确保 reclaim/recover 语义可回归。
- 在不引入平台化能力前提下，提升 worker 异常恢复确定性。

**Non-Goals:**
- 不引入外部 MQ 或平台控制面。
- 不改变 scheduler async-await 仲裁规则与既有 task board 语义。
- 不承诺 exactly-once；仍保持 at-least-once + 幂等收敛模型。

## Decisions

### 1) 引入 worker lease + reclaim 语义，默认 consume 路径触发回收
- 决策：增加 `inflight_timeout` 与 `reclaim_on_consume`，每次 `Consume` 前执行 stale in-flight reclaim。
- 推荐默认值：
  - `mailbox.worker.inflight_timeout=30s`
  - `mailbox.worker.reclaim_on_consume=true`
- 原因：与 mailbox 当前 pull 模式兼容，避免引入额外后台调度线程，且能在自然消费节奏下收敛僵尸 in-flight。
- 备选：独立后台 reclaim loop。拒绝原因：复杂度与并发面上升，且与现有库级简洁模型不一致。

### 2) 引入 heartbeat 续租，默认 5s
- 决策：worker 对正在处理的 in-flight 消息周期性 heartbeat，刷新 lease。
- 推荐默认值：`mailbox.worker.heartbeat_interval=5s`，并要求 `< inflight_timeout`。
- 原因：兼顾长任务稳定性与超时检测及时性。
- 备选：无 heartbeat，仅依赖 timeout。拒绝原因：长任务场景会被误回收。

### 3) panic recover 策略固定为 follow_handler_error_policy
- 决策：新增 `mailbox.worker.panic_policy=follow_handler_error_policy`，worker 捕获 panic 后按 handler error policy 执行 `requeue|nack`。
- 原因：复用既有错误策略面，减少配置分叉与语义歧义。
- 备选：panic 一律 dead-letter 或直接中断。拒绝原因：过于激进或不可恢复。

### 4) reason taxonomy 增加 `lease_expired`，panic 先映射到 `handler_error`
- 决策：将 reclaim 场景 canonical reason 扩展为 `lease_expired`；panic 默认映射 `handler_error` 并通过 metadata 区分。
- 原因：对外最小增量地表达 reclaim 语义，同时避免一次引入过多 reason 扩展点。
- 备选：新增 `panic_recovered` reason。拒绝原因：当前阶段收益不足，可能放大 taxonomy 维护负担。

## Risks / Trade-offs

- [Risk] `inflight_timeout` 过小导致误回收  
  -> Mitigation: 默认 30s + heartbeat 续租 + 配置校验（heartbeat 必须小于 timeout）。

- [Risk] reclaim 与正常重试路径交织引入状态机复杂度  
  -> Mitigation: 固化状态迁移顺序并增加 memory/file parity + replay determinism 套件。

- [Risk] panic recover 可能掩盖业务 handler 代码质量问题  
  -> Mitigation: diagnostics 增加 panic-recovered 观测字段并在 gate 套件中强制覆盖。

## Migration Plan

1. 扩展 mailbox state/store：增加 in-flight lease 元数据、heartbeat、stale reclaim 路径。
2. 扩展 worker：增加 panic recover 与 heartbeat 驱动，接入 reclaim 语义。
3. 扩展 runtime/config：新增 `mailbox.worker.inflight_timeout|heartbeat_interval|reclaim_on_consume|panic_policy`，补齐校验与热更新回滚。
4. 扩展 diagnostics：记录 reclaim/heartbeat/panic-recovered 事件与聚合字段。
5. 增加 integration/contract suites：worker crash reclaim、run/stream 等价、memory/file parity、taxonomy drift。
6. 更新 gate 与文档索引：`check-multi-agent-shared-contract.*`、`check-quality-gate.*`、README/roadmap/mainline index。

回滚策略：
- 可通过关闭 `mailbox.worker.enabled` 或 `mailbox.worker.reclaim_on_consume` 快速回退。
- 新增字段均为 additive，不改变既有配置键语义。

## Open Questions

无阻塞项，按推荐值执行：
- `inflight_timeout=30s`
- `heartbeat_interval=5s`
- `reclaim_on_consume=true`
- `panic_policy=follow_handler_error_policy`
