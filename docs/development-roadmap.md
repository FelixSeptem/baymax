# Development Roadmap

更新时间：2026-03-20

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-03-20：
- 已归档并稳定：A4-A32（含 A19 性能门禁、A20 全链路示例、A21 外部适配模板与迁移映射、A22 外部适配 conformance harness、A23 脚手架与 drift gate、A24 pre-1 轨道治理收口、A25 状态口径与模块 README 门禁、A26 manifest + runtime compatibility 契约、A27 capability negotiation + fallback 契约、A28 contract profile versioning + replay gate、A29 task board query contract、A30 mailbox 统一协调契约、A31 async-await lifecycle 收口、A32 async-await reconcile fallback 收口）。
- 进行中：
  - `enable-collaboration-primitive-bounded-retry-contract-a33`
  - `retire-legacy-direct-invoke-and-enforce-mailbox-canonical-entrypoints-a34`

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

### P0：A33 收口（当前阶段）

A33 依赖关系：
- A16 已收口协作原语（handoff/delegation/aggregation）基础语义；
- A33 在此基础上补齐“默认关闭、可显式开启”的 bounded retry 契约。

完成条件（A33）：
- 扩展 `composer.collab.retry.*` 配置域并冻结推荐默认值：
  - `enabled=false`
  - `max_attempts=3`
  - `backoff_initial=100ms`
  - `backoff_max=2s`
  - `multiplier=2.0`
  - `jitter_ratio=0.2`
  - `retry_on=transport_only`
- 固化 retry 分类和范围：默认仅 transport 重试；覆盖 sync delegation 与 async submit 阶段，不覆盖 accepted 后 async-await 收敛阶段。
- 固化 retry 所有权：scheduler 管理路径避免与 primitive retry 叠加，防止双重重试。
- 在 `runtime/diagnostics` 增加 collaboration retry additive 字段，并保持 `additive + nullable + default` 兼容窗口。
- shared multi-agent gate 纳入 collaboration retry suites（策略边界、Run/Stream 等价、replay idempotency）。

当前状态（A33）：
- 进入实施阶段：配置域、协作原语重试执行、诊断字段与 shared gate 接入按同一 change 收口。

当前阶段非目标（A33 不做）：
- 引入平台化重试编排控制面或外部消息总线依赖。
- 修改 A32 async-await 终态收敛主契约（callback/reconcile/timeout）。

### P1：0.x 质量与治理持续收敛

执行要求：
- 所有变更继续通过质量门禁（`check-quality-gate.*`）与契约索引追踪。
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
