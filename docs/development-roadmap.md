# Development Roadmap

更新时间：2026-03-23

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-03-23：
- 已归档并稳定：A4-A37（含 A19 性能门禁、A20 全链路示例、A21 外部适配模板与迁移映射、A22 外部适配 conformance harness、A23 脚手架与 drift gate、A24 pre-1 轨道治理收口、A25 状态口径与模块 README 门禁、A26 manifest + runtime compatibility 契约、A27 capability negotiation + fallback 契约、A28 contract profile versioning + replay gate、A29 task board query contract、A30 mailbox 统一协调契约、A31 async-await lifecycle 收口、A32 async-await reconcile fallback 收口、A33 collaboration bounded retry 收口、A34 canonical invoke 入口收口、A35 mailbox runtime wiring 收口、A36 mailbox lifecycle worker 收口、A37 Windows gate fail-fast parity 收口）。
- 进行中：
  - `harden-mailbox-worker-lease-reclaim-and-panic-recovery-contract-a38`
  - `introduce-task-board-control-and-manual-recovery-contract-a39`

## 版本阶段口径（延续 0.x）

当前仓库**不做 `1.0.0` / prod-ready 承诺**，继续沿用 `0.x` 治理口径（见 `docs/versioning-and-compatibility.md`）。
在 `0.x` 阶段，版本号用于表达变更范围，不构成稳定兼容承诺；主线目标是“持续收敛、可回归迭代”。
`0.x` 阶段**允许新增能力型提案**，不采用“仅治理/仅修复”的限制；新增能力需满足准入字段与质量门禁要求。

1. 运行时主干稳定：
- Runner Run/Stream 统一语义与并发背压基线。
- Multi-provider（OpenAI/Anthropic/Gemini）统一 contract。
- Context Assembler CA1-CA4、Security S1-S4 已归档能力。

2. 多代理主链路稳定：
- A11-A18（同步/异步/延后、恢复边界、协作原语、统一诊断查询）语义收口。
- Shared contract gate 与 Run/Stream 等价约束保持阻断。

3. 质量与可回归稳定：
- A19 性能回归门禁（基线 + 相对阈值）。
- A20 全链路示例 smoke 阻断门禁。

4. 外部接入稳定：
- A21 模板与迁移映射（已归档）。
- A22 conformance harness（已归档）。
- A23 scaffold + conformance bootstrap（已归档）。

## 近期收口优先级（0.x）

### P0：A32 收口（已归档）

A32 依赖关系：
- A31 已提供 `awaiting_report + timeout + late_report_policy` 生命周期基线；
- A32 在此基础上补齐 callback 之外的 poll reconcile fallback 契约。

完成条件（A32）：
- 为 `awaiting_report` 任务增加可配置 reconcile poll fallback：`interval/batch_size/jitter_ratio`。
- 终态仲裁固定为 `first_terminal_wins + record_conflict`，后到冲突事件不覆写业务终态。
- `not_found_policy=keep_until_timeout`：poll `not_found` 不直接终态，保持等待至 `report_timeout`。
- 在 async accepted 路径持久化远端关联键（`remote_task_id`）并跨 snapshot/recovery 保持可对账。
- Task Board 查询扩展 async additive 观测字段：`resolution_source`、`remote_task_id`、`terminal_conflict_recorded`。
- `runtime/config` 新增 `scheduler.async_await.reconcile.*`（默认关闭）并纳入 fail-fast + 热更新回滚。
- `runtime/diagnostics` 增加 reconcile additive 字段并保持 `additive + nullable + default` 兼容窗口。
- shared multi-agent gate 纳入 async-await reconcile suites（callback-loss fallback、冲突仲裁、Run/Stream 等价、memory/file parity、replay idempotency）。

当前阶段非目标（A32 不做）：
- 引入外部 MQ（Kafka/NATS/RabbitMQ）适配。
- 提供平台化消息控制面（UI/RBAC/多租户运维面板）。
- 承诺 exactly-once 语义。

### P0：A34 收口（已归档）

A34 依赖关系：
- A30 已确立 mailbox 统一协调主契约。
- A33 已归档，协作原语重试语义可作为稳定基线。

完成条件（A34）：
- 退场 legacy direct invoke 公共入口（`InvokeSync` / `InvokeAsync`）并固定 mailbox 为 canonical 调用面。
- `MailboxBridge` 内部不再依赖 deprecated direct invoke 导出路径。
- shared multi-agent gate 与 quality gate 增加 canonical-only 阻断，防止 legacy 入口回流。
- README / roadmap / mainline index / orchestration 模块文档移除“deprecated 但仍主路径依赖”的中间态描述。

当前阶段非目标（A34 不做）：
- 不引入平台化控制面或外部消息总线。
- 不改 A32 async-await 收敛仲裁语义。

### P1：A35 接线（已归档）

A35 依赖关系：
- A34 收口 canonical 调用入口后，进一步把 mailbox 配置与运行时主链路接线闭环。

完成条件（A35）：
- managed 编排路径接入共享 mailbox runtime wiring，避免 per-call `NewInMemoryMailboxBridge()` 中间态。
- `mailbox.enabled=false` 时使用共享 memory mailbox；`mailbox.enabled=true` 按 resolved backend 初始化。
- `mailbox.backend=file` 初始化失败回退到 memory，并记录 deterministic fallback reason。
- mailbox publish 主路径接入 diagnostics 写入，使 `QueryMailbox` / `MailboxAggregates` 反映真实主链路数据。
- shared multi-agent gate 纳入 mailbox runtime wiring 套件（配置接线、fallback、Run/Stream 等价、memory/file parity）。

当前阶段非目标（A35 不做）：
- 不引入 MQ 平台化能力或控制平面。
- 不替代 A34 的 API 收口目标。

### P1：A36 lifecycle worker 与可观测性（已归档）

A36 依赖关系：
- A35 已完成 mailbox runtime wiring 与 publish 诊断闭环；
- A36 在此基础上补齐 mailbox lifecycle worker 原语与 reason taxonomy 治理。

完成条件（A36）：
- 新增库级 mailbox worker 原语（默认关闭）：`consume -> handler -> ack|nack|requeue`。
- 固化 worker 默认值：`enabled=false`、`poll_interval=100ms`、`handler_error_policy=requeue`。
- `runtime/config` 增加 `mailbox.worker.*` 配置域并纳入启动/热更新 fail-fast + 原子回滚。
- mailbox lifecycle diagnostics 覆盖 `consume/ack/nack/requeue/dead_letter/expired`。
- lifecycle reason taxonomy 冻结为 canonical 集合：
  `retry_exhausted`、`expired`、`consumer_mismatch`、`message_not_found`、`handler_error`。
- shared multi-agent gate 纳入 worker lifecycle 套件（enabled/disabled、Run/Stream 等价、memory/file parity、taxonomy drift guard）。

当前阶段非目标（A36 不做）：
- 不引入外部 MQ、平台化控制面或托管任务面板。
- 不改变 A32 async-await 终态仲裁语义。

### P1：A38 worker lease reclaim + panic recover（进行中）

A38 依赖关系：
- A36 已提供 mailbox lifecycle worker 基线（consume->handler->ack|nack|requeue）；
- A37 已完成 Windows 门禁 strict fail-fast parity，A38 在该基线上补齐恢复语义。

完成条件（A38）：
- mailbox worker 增加 lease/reclaim/recover 契约：
  - `inflight_timeout=30s`
  - `heartbeat_interval=5s`
  - `reclaim_on_consume=true`
  - `panic_policy=follow_handler_error_policy`
- stale `in_flight` 在 consume 路径支持 deterministic reclaim，canonical reason 使用 `lease_expired`。
- handler panic recover 路径按既有 handler-error policy 收敛（`requeue|nack`），并保留 `panic_recovered` 可观测标记。
- `runtime/config` 新增 `mailbox.worker.{inflight_timeout,heartbeat_interval,reclaim_on_consume,panic_policy}` 并纳入启动/热更新 fail-fast + 原子回滚。
- mailbox diagnostics 增加 reclaim/recover additive 字段（`reclaimed`、`panic_recovered`），并保持 query/aggregate 兼容。
- shared multi-agent gate 纳入 A38 recover/reclaim 套件并保持阻断。

当前阶段非目标（A38 不做）：
- 不引入独立后台 reclaim 控制线程或平台化任务控制面。
- 不改变 A30/A34 mailbox canonical 调用面与 A32 async-await 仲裁语义。

### P2：0.x 质量与治理持续收敛

执行要求：
- 所有变更继续通过质量门禁（`check-quality-gate.*`）与契约索引追踪。
- shell 与 PowerShell 门禁 required checks 维持语义等价：native command 非零即 fail-fast；仅 `govulncheck + warn` 允许告警放行。
- 继续按“小步提案 + 契约测试 + 文档同步”推进，不引入平台化控制面范围。
- 对外发布继续以 `0.x` 说明风险与兼容预期。

## 维护提示（状态快照更新）

每次归档或切换活跃 change 后，维护者应同步执行以下最小流程，避免触发 A25 口径漂移阻断：

1. 以 `openspec list --json` 与 `openspec/changes/archive/INDEX.md` 作为唯一状态权威源。
2. 更新 `README.md` 的里程碑快照和 `docs/development-roadmap.md` 的“当前状态”在研列表，确保 active/archived 语义一致。
3. 若状态变更涉及门禁映射，更新 `docs/mainline-contract-test-index.md` 的对应行。
4. 提交前执行 `pwsh -File scripts/check-docs-consistency.ps1`（或 shell 等价脚本）确认无漂移。

## 新增提案准入规则（0.x 阶段）

从本文件生效起，`0.x` 阶段新增提案需满足：

1. 允许新增能力型提案进入近期执行，但必须直接服务于以下至少一类目标：
- 契约一致性（Run/Stream、reason taxonomy、错误分层、兼容语义）。
- 可靠性与安全（fail-fast、回滚、幂等、恢复边界、安全治理）。
- 质量门禁回归治理（contract/perf/docs gate regression）。
- 外部接入 DX（模板、迁移、脚手架、conformance）且可被 gate 验证。
2. 必须保持 lib-first 边界，不引入平台化控制面能力。
3. 必须在提案内说明：`Why now`、风险、回滚点、文档影响、验证命令。
4. 不满足以上条件的需求，统一记录为长期方向，不进入近期执行。

## 长期方向（不进入近期主线）

以下方向明确延后：
- 平台化控制面（多租户、RBAC、审计与运营面板）。
- 跨租户全局调度与控制平面。
- 市场化/托管化 adapter registry 能力。

说明：上述方向在 `0.x` 阶段只登记，不作为当前迭代实施输入。

## 执行与验收规则

- 单变更优先；并行变更需显式依赖边界。
- 严格顺序：`proposal/design/spec/tasks -> code -> tests -> docs`。
- 合并前最少验证：
  - `go test ./...`
  - `go test -race ./...`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `pwsh -File scripts/check-quality-gate.ps1`
