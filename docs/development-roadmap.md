# Development Roadmap

更新时间：2026-04-02

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-04-02：
- 已归档并稳定：A4-A60（完整清单以 `openspec/changes/archive/INDEX.md` 为准）。
- 进行中：
  - `introduce-otel-tracing-and-agent-eval-interoperability-contract-a61`

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
- A42 diagnostics query 性能回归门禁（`BenchmarkDiagnosticsQueryRuns|QueryMailbox|MailboxAggregates`，默认阈值 `12/15/12%`，已归档）。
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

### P1：A39 task board control + manual recovery（已归档）

A39 依赖关系：
- A29 已交付 Task Board query 只读契约；
- A39 在保持 query 只读语义不变的前提下，补齐库级 control 路径与手工恢复契约。

完成条件（A39）：
- 新增 scheduler 控制入口，支持动作：
  - `cancel`：仅允许 `queued|awaiting_report`，`running` fail-fast（不做强杀）。
  - `retry_terminal`：仅允许 `failed|dead_letter -> queued`。
- 引入 `operation_id` 幂等键：重复提交 dedup，不重复膨胀 counters。
- 扩展 canonical reason taxonomy：
  - `scheduler.manual_cancel`
  - `scheduler.manual_retry`
- `runtime/config` 增加 `scheduler.task_board.control.enabled=false` 与 `scheduler.task_board.control.max_manual_retry_per_task=3`，并纳入启动/热更新 fail-fast + 原子回滚。
- `runtime/diagnostics` 增加 manual-control additive 字段（total/success/rejected/idempotent_dedup + action/reason breakdown）。
- shared multi-agent gate 与 quality gate 纳入 task-board-control contract suites（memory/file parity、Run/Stream 等价、replay idempotency）。

当前阶段非目标（A39 不做）：
- 不引入平台化任务控制面（RBAC/UI/多租户运维）。
- 不改变既有 enqueue/claim/heartbeat/requeue/commit 与 query 只读路径语义。

### P1：A40 runtime readiness preflight + degradation contract（已归档）

A40 依赖关系：
- A35/A36 已将 scheduler/mailbox/recovery fallback 状态统一回流到 runtime 诊断路径；
- A40 在保持 lib-first 边界下新增启动前 readiness 预检契约，不改变既有 Run/Stream 终态裁决。

完成条件（A40）：
- `runtime/config.Manager` 提供库级 `ReadinessPreflight()`，输出 `ready|degraded|blocked` 与 canonical findings（`code/domain/severity/message/metadata`）。
- 新增 `runtime.readiness.*` 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 预检覆盖本地配置有效性与 scheduler/mailbox/recovery backend/fallback 可见性。
- `strict=true` 时把 `degraded` 升级为 `blocked`，`strict=false` 保持可运行且可观测。
- run diagnostics 增量字段落地：`runtime_readiness_status`、计数字段、`runtime_readiness_primary_code`。
- composer 暴露 runtime readiness 透传入口，且查询路径保持只读，不引入新状态 taxonomy。
- quality gate 接入 readiness suites（classification、strict escalation、schema stability、diagnostics replay idempotency、composer parity）。

当前阶段非目标（A40 不做）：
- 不引入平台化控制面/远程运维探针系统。
- 不改变 scheduler/task lifecycle 语义，不引入额外终态。

### P1：A41 operation profile + timeout resolution contract（已归档）

A41 依赖关系：
- A40 readiness 契约已归档，运行时配置与诊断路径具备稳定扩展点；
- A41 在既有 scheduler/composer 多代理主链路上，补齐跨域 timeout 解析与父子预算收敛。

完成条件（A41）：
- `runtime.operation_profiles.*` 配置域落地，并保持 `env > file > default` 与 fail-fast/回滚语义。
- 共享 timeout resolver 固化 `profile -> domain -> request` 优先级，并输出来源标签与可追踪 trace。
- scheduler/composer 子任务路径统一接入 resolver；父子预算收敛固定为 `min(parent_remaining, child_resolved)`。
- timeout-resolution 元数据在 snapshot/recovery/replay 下保持稳定，且 replay 不膨胀逻辑聚合。
- diagnostics 与 QueryRuns/Task Board 补齐 A41 additive 字段，并保持 `additive + nullable + default` 兼容语义。
- shared contract gate 与 quality gate 纳入 A41 阻断套件（校验/优先级/夹紧与拒绝/Run-Stream 等价/memory-file parity/replay idempotency）。

当前阶段非目标（A41 不做）：
- 不引入平台化控制面与外部 MQ 依赖。
- 不改变既有 async-await/recovery 终态仲裁契约。

### P1：A42 diagnostics query performance baseline + regression gate（已归档）

A42 目标：
- 为 unified diagnostics query 建立可复现实验基线（延迟、分页、聚合开销）。
- 新增独立 gate 脚本：`scripts/check-diagnostics-query-performance-regression.sh` 与 `scripts/check-diagnostics-query-performance-regression.ps1`。
- 固化默认执行参数：`benchtime=200ms`、`count=5`。
- 在质量门禁接入回归阈值校验（默认：`ns/op 12%`、`p95-ns/op 15%`、`allocs/op 12%`），防止查询路径性能漂移。

### P1：A43 adapter runtime health probe + readiness integration（已归档）

A43 目标：
- 新增 `adapter/health` 运行期探测契约，固化 `healthy|degraded|unavailable` 三态与 canonical reason taxonomy。
- 新增 `adapter.health.*` 配置域（`enabled/strict/probe_timeout/cache_ttl`），并纳入 `env > file > default`、启动 fail-fast、热更新回滚。
- 将 adapter health 接入 `ReadinessPreflight()`：
  - required unavailable 在 strict 语义下阻断；
  - optional unavailable 在 non-strict 路径降级并保持可观测。
- 在 diagnostics 增加 adapter-health additive 字段（`status/probe_total/degraded_total/unavailable_total/primary_code`），保证 replay idempotency。
- 在 `integration/adapterconformance` 增加 adapter-health matrix，并接入 `check-adapter-conformance.*` 与 `check-quality-gate.*` 阻断步骤（shell/PowerShell parity）。

### P1：A44 readiness admission guard + degradation policy（已归档）

A44 目标：
- 在 managed Run/Stream 入口引入统一 readiness admission guard，形成执行前准入护栏。
- 新增 `runtime.readiness.admission.*` 配置域并保持 `env > file > default`、启动 fail-fast、热更新回滚语义。
- 固化 `blocked` 拒绝执行与 `degraded` 策略化处理（allow_and_record / fail_fast）规则。
- 增加 admission additive 诊断字段并纳入 replay idempotency 契约。
- 将 admission suites 纳入 quality gate 阻断路径并保持 shell/PowerShell parity。

### P1：A45 diagnostics cardinality budget + truncation governance（已归档）

A45 目标：
- 为新增 additive 字段建立高基数预算与截断治理，避免查询成本漂移。
- 固化 map/list/string 字段的 bounded-cardinality 与稳定序列化语义。
- 新增 `diagnostics.cardinality.*` 配置域，默认 `overflow_policy=truncate_and_record`，并支持 `fail_fast`。
- 将 cardinality drift 检查纳入质量门禁与回放契约验证。

### P1：A46 adapter health backoff + circuit governance（已归档）

A46 目标：
- 在 A43 健康探测语义上增加指数退避 + 抖动 + 半开探测治理。
- 防止外部 adapter 不可用时的探测风暴与瞬时抖动放大。
- 通过 conformance + quality gate 固化故障恢复和抖动抑制语义。

A46 当前落地（实现已完成）：
- `runtime/config` 新增 `adapter.health.backoff.*` 与 `adapter.health.circuit.*`（default/env/file、startup 校验、hot reload 非法更新回滚）。
- `adapter/health` 落地 `closed|open|half_open` 状态机、指数退避 + 抖动、half-open 探测预算与恢复判定。
- `runtime/config/readiness` 增加 circuit-open / half-open degraded / governance recovered 的 canonical `adapter.health.*` finding 映射，并保持 strict/non-strict 分类稳定。
- `runtime/diagnostics` 与 `RuntimeRecorder` 新增 A46 additive 字段：`adapter_health_backoff_applied_total`、`adapter_health_circuit_*`、`adapter_health_governance_primary_code`。
- `integration/adapterconformance` 新增 governance matrix suites（状态转移确定性、半开恢复、taxonomy drift guard、replay idempotency）。
- `scripts/check-adapter-conformance.*` 与 `scripts/check-quality-gate.*` 已纳入 A46 suites 并保持 shell/PowerShell parity。

### P1：A47 readiness-timeout-health replay fixture gate（已归档）

A47 目标：
- 固化 `readiness + timeout resolution + adapter health` 交叉语义回放夹具。
- 防止跨提案演进造成 finding taxonomy 与阻断策略漂移。
- 为后续 0.x 收敛阶段提供稳定的语义回归基线。

A47 当前落地（实现已完成）：
- `tool/diagnosticsreplay` 新增 A47 组合 fixture schema（`version=a47.v1`）、loader、canonical normalization 与 deterministic assertion pipeline。
- 错误分类补齐 `schema_mismatch|semantic_drift|ordering_drift`，并对 taxonomy/source/state 漂移执行 fail-fast。
- 新增 `integration/readiness_timeout_health_replay_contract_test.go` 与 `integration/testdata/diagnostics-replay/a47/v1/*`（success + taxonomy/source/state drift fixtures）。
- quality gate 接入 A47 阻断步骤：`go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReadinessTimeoutHealthReplayContract)' -count=1`，shell/PowerShell parity 保持一致。
- 主干索引与 diagnostics 文档已补齐 A47 fixture suite 与 gate 映射。

### P1：A48 cross-domain primary reason arbitration（已归档）

A48 目标：
- 固化 timeout/readiness/adapter-health 冲突场景下的 primary reason 裁决优先级与 tie-break 规则。
- 统一 `runtime_primary_domain|code|source` 解释链路，保持 Run/Stream/replay 语义一致。
- 将 arbitration drift 检测纳入 replay + quality gate 阻断，防止跨提案演进产生 reclassification drift。

A48 当前落地（已归档）：
- `runtime/config` 新增 cross-domain arbitration helper，固定 precedence（timeout reject/exhausted > readiness blocked > adapter required unavailable > degraded/optional > warning/info）并支持 lexical tie-break 与 conflict_total。
- `runtime/config/readiness` 与 admission guard 统一消费 arbitration 输出，解释字段对齐 `primary domain/code/source`，Run/Stream 保持语义等价。
- `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 增加 A48 additive 字段：`runtime_primary_domain`、`runtime_primary_code`、`runtime_primary_source`、`runtime_primary_conflict_total`，并保持 replay idempotency。
- `tool/diagnosticsreplay` 新增 A48 fixture schema（`version=a48.v1`）与 drift 分类：`precedence_drift`、`tie_break_drift`、`taxonomy_drift`。
- 新增 `integration/primary_reason_arbitration_replay_contract_test.go` 与 `integration/testdata/diagnostics-replay/a48/v1/*`，覆盖 replay parity + drift guard。
- quality gate 阻断步骤扩展为：`go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReplayContractPrimaryReasonArbitrationFixture|ReadinessTimeoutHealthReplayContract|PrimaryReasonArbitrationReplayContract)' -count=1`（shell/PowerShell parity 保持一致）。

### P1：A49 arbitration explainability + secondary reason（已归档）

A49 目标：
- 固化 secondary reasons 的有界输出契约（上限、去重、稳定排序）并输出 rule version。
- 统一 remediation hint taxonomy，补齐 machine-readable explainability 字段。
- 将 explainability drift（secondary order/count、hint taxonomy、rule version）纳入 replay + quality gate 阻断。

A49 当前落地（已归档）：
- `runtime/config` 已扩展 arbitration explainability 输出：`runtime_secondary_reason_codes`、`runtime_secondary_reason_count`、`runtime_arbitration_rule_version`、`runtime_remediation_hint_code`、`runtime_remediation_hint_domain`，并固定 `max_secondary_reasons=3`。
- `runtime/config/readiness` 与 admission guard 已贯通 explainability 字段透传（primary + secondary + hint + rule version），deny details 保持 machine-readable 字段对齐。
- `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 已接入 A49 additive 字段并补齐 replay idempotency 断言。
- `tool/diagnosticsreplay` arbitration fixture 已升级支持 `version=a49.v1`，新增 drift 分类：`secondary_order_drift`、`secondary_count_drift`、`hint_taxonomy_drift`、`rule_version_drift`。
- quality gate readiness 套件已纳入 A49 parser-compatibility 回归（shell/PowerShell parity）。

### P1：A50 arbitration version governance + compatibility（已归档）

A50 目标：
- 固化 arbitration rule version 解析与 compatibility window 契约（requested/default/effective/source）。
- 统一 unsupported/mismatch 策略（默认 fail-fast），并贯通 readiness preflight 与 admission guard。
- 将 cross-version drift（`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`）纳入 replay + quality gate 阻断。

A50 当前落地（已归档）：
- `runtime/config` 已新增 `runtime.arbitration.version.*` 配置域（`enabled/default/compat_window/on_unsupported/on_mismatch`），并接入 `env > file > default`、启动 fail-fast 校验、热更新非法回滚。
- cross-domain arbitration/readiness/admission 已接入 version resolver，unsupported/mismatch 在 fail-fast 策略下保持 deterministic deny 与 explainability 透传（requested/effective/source/policy/counters）。
- `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 已接入 A50 additive 字段：`runtime_arbitration_rule_requested_version`、`runtime_arbitration_rule_effective_version`、`runtime_arbitration_rule_version_source`、`runtime_arbitration_rule_policy_action`、`runtime_arbitration_rule_unsupported_total`、`runtime_arbitration_rule_mismatch_total`。
- `tool/diagnosticsreplay` arbitration fixture 已升级支持 `version=a50.v1`，并新增 drift 分类：`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`，同时保持 `a48/a49` 向后兼容。
- 新增 A50 integration suites（Run/Stream parity、memory/file parity、replay parity），并已纳入 `check-quality-gate.sh/.ps1` 阻断步骤。

### P1：A51 sandbox execution isolation contract（已归档）

A51 Why now：
- 当前 S2-S4 已覆盖权限/限流/IO 过滤与 deny 告警投递，但本地工具执行仍以 in-process 为主，缺少“执行隔离”契约层。
- 对高风险工具（如 shell/file-system/process 访问）仅靠策略 deny/confirm 不足以满足更高隔离要求，需要可审计的 sandbox 运行面。
- 在保持 lib-first 边界前提下，需要提供“宿主可注入隔离执行器 + 运行时统一治理/诊断”的标准接缝，避免业务侧散装实现。

A51 依赖关系：
- 复用 S2/S3/S4 既有 taxonomy 与事件投递治理，不新增平行安全事件体系。
- 复用 A44 readiness admission；当 `sandbox.required=true` 且执行器不可用时，准入层可 fail-fast 阻断。
- 复用 A45 additive/cardinality 治理，保证 sandbox 诊断字段新增不破坏查询性能与兼容窗口。
- 复用 A49/A50 explainability 与 rule-version 口径，确保 sandbox deny 在 Run/Stream/replay 下可解释且稳定。

A51 完成条件（提案落地后）：
- 新增 `security.sandbox.*` 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚：
  - `enabled`（默认 `false`）、`mode`（`observe|enforce`）、`required`（默认 `false`）。
  - `default_action`（`host|sandbox|deny`）与 `by_tool`（`namespace+tool` 选择器，沿用 S2 格式）。
  - `fallback_action`（`allow_and_record|deny`）、`launch_timeout`、`exec_timeout`、`max_concurrency`。
  - `profile`（资源约束档位）与 `profiles.*`（如 cpu/memory/network/filesystem 限额，具体枚举由提案冻结）。
- 新增宿主注入式隔离执行 SPI（库不内置容器编排），并在以下路径统一接入：
  - `tool/local` 高风险工具调用路径支持按策略切换 `host`/`sandbox` 执行。
  - `mcp/stdio` 命令启动路径支持 sandbox 启动器接管（保持 transport 语义不变）。
- 固化 sandbox 决策与错误 taxonomy（示例）：`security.sandbox_policy_denied`、`security.sandbox_executor_missing`、`security.sandbox_launch_failed`、`security.sandbox_timeout`。
- `runtime/diagnostics` 增加 sandbox additive 字段并保持 `additive + nullable + default`：
  - 建议字段：`sandbox_mode`、`sandbox_profile`、`sandbox_decision`、`sandbox_reason_code`、`sandbox_fallback_used`、`sandbox_fallback_reason`、`sandbox_violation_total`、`sandbox_timeout_total`、`sandbox_launch_failed_total`。
- Run/Stream 语义等价：
  - 同输入同配置下，sandbox allow/deny/fallback 的终态、reason code、diagnostics 字段保持等价。
  - deny 路径保持 side-effect free（不触发调度/发布副作用），与 A44 admission 语义一致。
- 回放与门禁：
  - diagnostics replay 增加 sandbox fixture（建议 `a51.v1`）与 drift 分类（taxonomy/order/idempotency）。
  - quality gate 新增 `check-security-sandbox-contract.sh/.ps1` 并纳入 `check-quality-gate.*`。
  - 增加 offline deterministic `sandbox executor conformance harness`（`check-sandbox-executor-conformance.sh/.ps1`）并接入 sandbox gate。
  - CI 暴露独立 required-check 候选 `security-sandbox-gate`。

A51 当前落地（已归档）：
- `integration/sandbox_execution_isolation_contract_test.go` 已覆盖 Run/Stream parity、capability negotiation deny、backend compatibility matrix smoke（Linux + Windows job）。
- `integration/sandboxconformance` 已落地 offline deterministic conformance harness（canonical ExecSpec/ExecResult、capability negotiation drift、session lifecycle、fallback 语义）。
- `scripts/check-security-sandbox-contract.sh/.ps1` 已接入 conformance harness，并由 `scripts/check-quality-gate.sh/.ps1` 阻断执行。
- `.github/workflows/ci.yml` 已新增独立 job `security-sandbox-gate`（PR 触发）作为 required-check 候选。

A51 当前阶段非目标（不做）：
- 不内置 Docker/Kubernetes/VM 控制面，不引入平台化多租户治理能力。
- 不承诺跨主机/跨内核强隔离（隔离强度由宿主注入执行器能力决定）。
- 不改变 provider fallback、A2A/workflow/scheduler 既有主链路语义。

A51 风险与回滚点：
- 主要风险：策略误配导致误拒绝、sandbox 启动开销导致时延抖动、跨平台执行器行为漂移。
- 缓解策略：先 `mode=observe` 灰度，稳定后切换 `mode=enforce`；高风险工具先小范围 `by_tool` 启用。
- 回滚点：`security.sandbox.enabled=false` 或 `mode=observe`；非法热更新一律回滚到上一有效快照。

A51 验证命令（提案实施期最小集合）：
- `go test ./tool/local ./core/runner ./mcp/stdio -count=1`
- `go test ./integration -run '^TestSandboxExecutionIsolationContract' -count=1`
- `go test ./integration/sandboxconformance -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-sandbox-executor-conformance.ps1`
- `pwsh -File scripts/check-security-sandbox-contract.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

### P1：A52 sandbox runtime rollout + health/capacity governance（已归档）

A52 Why now：
- A51 已冻结 sandbox 接入与隔离语义，但“如何安全放量上线”仍缺统一 contract，当前容易落回业务侧脚本治理。
- rollout/freeze/capacity 若不统一到 readiness/admission/diagnostics/replay，将导致 Run/Stream 语义漂移与回滚不可验证。
- 需要把 sandbox 从“可用”提升到“可持续上线”，并保持主流后端接入路径在统一治理面下可替换。

A52 依赖关系：
- 复用 A51 的 sandbox execution isolation contract，不重新定义 ExecSpec/ExecResult 与 capability negotiation 基线。
- 复用 A44 readiness/admission fail-fast 与 deny side-effect-free 语义，作为 rollout/capacity 判定执行前置入口。
- 复用 A42/A45 的 diagnostics query/perf/cardinality 治理，确保 rollout 新字段不破坏查询与兼容窗口。
- 复用 A49/A50 的 explainability 与 version-governance 输出口径，保证冻结/节流动作可解释可回放。

A52 完成条件（提案落地后）：
- 新增 `security.sandbox.rollout.*` 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚：
  - phase 状态机：`observe|canary|baseline|full|frozen`（含合法迁移约束）。
  - 健康预算：启动失败率、超时率、违规率、P95 时延漂移、准入拒绝率。
  - 容量预算：`max_inflight`、`max_queue`、`throttle_threshold`、`deny_threshold`、`degraded_policy`。
  - 冻结治理：`freeze_on_breach`、`cooldown`、`manual_unfreeze_token`。
- readiness preflight + admission guard 接入 rollout/freeze/capacity canonical findings 与 deterministic 准入动作（`allow|throttle|deny`）。
- timeline/diagnostics/replay 一体化收敛：
  - timeline 新增 `sandbox.rollout.*` canonical reasons。
  - diagnostics 新增 rollout/capacity/freeze additive 字段并保持 single-writer idempotency。
  - replay 新增 `a52.v1` fixture 与 drift 分类（phase/health/capacity/freeze）。
- quality gate 收口：
  - 新增 `check-sandbox-rollout-governance-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
  - 作为独立 required-check 候选暴露，保持 shell/PowerShell parity。

A52 当前阶段非目标（不做）：
- 不引入平台化控制面（多租户运维面板、跨租户调度中心）。
- 不改变 A51 sandbox 执行 contract（ExecSpec/ExecResult/capability）。
- 不引入跨主机全局容量编排，仅定义单 runtime contract。

A52 风险与回滚点：
- 主要风险：预算阈值过紧导致误冻结、峰值期节流策略造成拒绝率抬升、后端抖动导致频繁冻结。
- 缓解策略：默认 `phase=observe`，先 `canary` 小流量；对冻结引入 cooldown + token 解冻；保留 `allow_and_record` 过渡策略。
- 回滚点：将 phase 回退到 `observe`，或暂时禁用 `freeze_on_breach`；非法热更新一律回滚到上一有效快照。

A52 验证命令（提案实施期最小集合）：
- `go test ./runtime/config ./runtime/config/readiness ./core/runner -count=1`
- `go test ./integration -run 'TestSandboxRollout|TestSandboxCapacityAdmission|TestRunStreamSandboxRolloutParity' -count=1`
- `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractSandboxA52Fixture' -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

### P1：A53 mainstream sandbox adapter conformance + migration pack（已归档）

A53 Why now：
- A52 已归档并冻结 sandbox rollout/health/capacity 治理基线，但主流后端（nsjail/bwrap/OCI/windows-job）接入仍依赖分散脚本与非标准 glue code。
- 若不统一 adapter manifest + conformance + migration mapping，后端切换成本高，且语义漂移很难被 gate 前置阻断。
- 需要在 A52 后续阶段收敛接入 contract，避免后续重复提出“同类 sandbox 接入治理”提案。

A53 依赖关系：
- 复用 A51 的 sandbox 执行隔离语义与 canonical backend/capability taxonomy，不重定义执行 contract。
- 复用 A52 rollout/health/capacity 治理语义，仅关注“外部接入 DX + conformance + migration”层。
- 复用 A21/A22/A26/A28 的 adapter template/conformance/manifest/profile-replay 治理链路，做 sandbox 维度扩展。

A53 完成条件（提案落地后）：
- 新增主流 sandbox profile pack（`linux_nsjail`、`linux_bwrap`、`oci_runtime`、`windows_job`）并冻结 profile 字段语义。
- 扩展 adapter manifest compatibility：
  - 新增 `sandbox_backend`、`sandbox_profile_id`、`host_os`、`host_arch`、`session_modes_supported` 字段契约。
  - 激活边界 fail-fast 校验 host/backend/session 兼容性。
- 扩展 conformance harness：
  - 新增 backend matrix suites（平台可用性条件化执行）。
  - 覆盖 capability negotiation + session lifecycle（crash/reconnect/close idempotent）。
  - 固化 drift 分类（backend/profile/manifest/session/taxonomy）。
- 扩展 template + migration mapping：
  - 提供四类 backend onboarding 模板与 legacy wrapper -> profile-pack adapter 迁移映射。
  - 模板与迁移条目必须绑定 conformance case id。
- 扩展 profile replay + quality gate：
  - 新增 `sandbox.v1` replay fixtures，保持既有 profile fixture 向后兼容。
  - 新增 `check-sandbox-adapter-conformance-contract.sh/.ps1` 并接入 `check-quality-gate.*`，暴露独立 required-check 候选。
- 扩展 readiness preflight：
  - 新增 `sandbox.adapter.*` finding（profile missing/backend unsupported/host mismatch/session unsupported）并保持 strict/non-strict 映射。

A53 当前阶段非目标（不做）：
- 不改 A51/A52 的 sandbox 执行与运行治理语义。
- 不引入平台化控制面或跨租户编排能力。
- 不承诺后端底层实现一致，仅要求 canonical 合同输出一致。

A53 已落地增量（归档记录）：
- `adapter/manifest` 已补齐 sandbox metadata/profile-pack 契约（`sandbox_backend`、`sandbox_profile_id`、`host_os`、`host_arch`、`session_modes_supported`）与 fail-fast 校验。
- `integration/adapterconformance` 已增加 mainstream backend matrix、capability negotiation、session lifecycle（`per_call|per_session`、crash/reconnect、close idempotent）与 canonical drift class 断言。
- `integration/adaptercontractreplay` 已增加 `sandbox.v1` 回放轨道与 mixed-track 回放，补齐 drift 分类断言（`sandbox_backend_profile_drift`、`sandbox_manifest_compat_drift`、`sandbox_session_mode_drift`）。
- `runtime/config/readiness` 已增加 `sandbox.adapter.*` finding（`profile_missing`、`backend_not_supported`、`host_mismatch`、`session_mode_unsupported`）与 strict/non-strict 测试映射。
- 已新增 `scripts/check-sandbox-adapter-conformance-contract.sh/.ps1` 并接入 `check-quality-gate.*`，CI 暴露独立 job `sandbox-adapter-conformance-gate`。

A53 风险与回滚点：
- 主要风险：profile pack 过重导致接入门槛上升、不同 runner 的 backend 可用性差异引发误报、模板与实现漂移。
- 缓解策略：最小必填 schema、平台条件化 matrix + skip reason 审计、模板绑定 conformance case 持续校验。
- 回滚点：暂时下线新增 sandbox adapter gate required-check；保留现有 adapter conformance 主路径与旧模板文档。

A53 验证命令（提案实施期最小集合）：
- `go test ./adapter/... ./tool/... -count=1`
- `go test ./integration -run 'TestSandboxAdapterConformance|TestSandboxAdapterManifestCompatibility|TestSandboxAdapterProfileReplay' -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

### P1：A54 memory provider SPI + builtin filesystem engine（已归档）

A54 Why now：
- 当前 memory 接入仍依赖 CA2 file/external retriever 分散路径，缺少统一 memory SPI 与 profile 契约。
- 主流 memory 框架（`mem0|zep|openviking`）接入成本高，且 provider-specific 分支容易渗透主流程并造成语义漂移。
- 需要一次性冻结 memory 的 config/readiness/diagnostics/replay/conformance/gate 契约，避免后续在 memory 主题上重复拆提案。

A54 依赖关系：
- 复用既有 runtime config 热更新治理（`env > file > default`、fail-fast、原子回滚）与 RuntimeRecorder single-writer 约束。
- 复用 A21/A22/A26/A28 的 template/conformance/manifest/profile-replay 治理链路，扩展到 memory 维度。
- 复用 A42/A45 的 diagnostics query/perf/cardinality 治理边界，确保 memory additive 字段不破坏查询与兼容窗口。
- 复用 A44 readiness strict/non-strict 映射语义，新增 `memory.*` findings 而不引入平行判定体系。

A54 完成条件（提案落地后）：
- 新增 `runtime-memory-engine-spi-and-filesystem-builtin` capability，冻结 canonical memory SPI（`Query/Upsert/Delete`）与错误 taxonomy。
- 新增 `runtime.memory.mode=external_spi|builtin_filesystem`，支持启动/热更新原子切换与失败回滚。
- 内置文件系统 memory 引擎契约收敛（append-only WAL + 原子 compaction/index + crash-safe recovery）。
- 新增主流 profile pack：`mem0`、`zep`、`openviking`、`generic`，并固定 required/optional capability 语义。
- CA2 Stage2 memory 路径统一经 memory facade，保持 Run/Stream 与 `fail_fast|best_effort` 语义等价。
- readiness/preflight 增加 `memory.*` findings；diagnostics 增加 memory additive 字段并保持 bounded-cardinality。
- replay 新增 `memory.v1` fixture 与 drift 分类；quality gate 新增 memory contract gate 并保持 shell/PowerShell parity。
- adapter manifest/template/migration/conformance 一体化扩展，覆盖 external SPI 与 builtin filesystem 双路径接入。

A54 当前阶段非目标（不做）：
- 不引入平台化 memory 控制面或跨租户调度系统。
- 不改 A51/A52 sandbox contract 语义，只复用其治理框架。
- 不承诺外部 provider 底层实现一致，仅要求 canonical 合同输出一致。

A54 风险与回滚点：
- 主要风险：外部 provider 能力差异导致 profile 语义不一致、模式切换误配导致运行抖动、文件系统 compaction 恢复窗口处理不当。
- 缓解策略：required/optional capability 分层、切换前 preflight 校验、WAL + 原子替换 + crash-recovery 合同测试。
- 回滚点：切换回 `builtin_filesystem` 或 `external_spi` 上一稳定配置快照；热更新失败一律原子回滚。

A54 验证命令（提案实施期最小集合）：
- `go test ./context/... ./runtime/config ./runtime/diagnostics -count=1`
- `go test ./integration -run 'TestMemoryProviderSPI|TestMemoryModeSwitch|TestMemoryRunStreamParity' -count=1`
- `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractMemoryFixture' -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `bash scripts/check-memory-contract-conformance.sh`
- `pwsh -File scripts/check-memory-contract-conformance.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

A54 gate 交付口径（当前实现）：
- memory contract suites 以 `smoke|full` 分层执行（主线 quality gate 默认 smoke，CI 独立 `memory-contract-gate` job 默认 full）。
- shell 与 PowerShell 脚本保持同一阻断语义（native command 非零即 fail-fast）。

### P1：A56 react loop + tool-calling parity contract（已归档）

A56 Why now：
- Run/Stream 在工具闭环路径长期存在语义偏移风险（step 边界、dispatch、feedback 与终止 reason 不完全同构）。
- provider tool-calling 映射与 readiness/admission/sandbox 语义在多提案叠加后需要统一收敛到单一 contract 口径。
- 需要把 ReAct 主题一次性接入 replay + gate，避免后续分散修补。

A56 当前落地（截至 2026-03-31）：
- loop 与 taxonomy：Run/Stream 共享 ReAct termination taxonomy（`react.completed`、预算耗尽、dispatch 失败、provider 错误、取消）。
- provider canonicalization：`model/openai|anthropic|gemini` 的 tool-call request/feedback 映射与 provider error taxonomy 已收敛。
- readiness/admission：新增 `react.*` finding（loop/stream dispatch/provider/tool registry/sandbox dependency）并贯通 strict/non-strict 与 admission deny/allow 语义。
- sandbox consistency：ReAct 多轮 host/sandbox/deny、fallback、capability mismatch 在 Run/Stream 下已具备 contract parity。
- replay/gate：`tool/diagnosticsreplay` 新增 `react.v1` fixture 与 drift 分类；`scripts/check-react-contract.sh/.ps1` 已接入 `check-quality-gate.*`；CI 已暴露 `react-contract-gate`。
- docs/examples：README、runtime-config-diagnostics、mainline index 与示例文档已补齐 ReAct 最小接入、字段与门禁映射。

A56 一次性闭环审查（10.4）：
- 审查范围：`loop -> provider -> readiness -> admission -> sandbox -> replay -> gate -> docs`。
- 审查结论：上述链路已形成同一 contract 语义闭环，当前没有必须再拆分的 ReAct 后续子提案。
- 剩余动作：执行全量回归验证（`go test`/`race`/`lint`/gate/docs consistency）并完成提案归档流程。

### P1/P2：A58+ 候选提案池（全局视角）

前提约束（冻结）：
- 不调整 A56/A57 的既有范围、完成条件与验收口径；后续提案仅做增量扩展。
- 新提案必须复用既有治理主链路：`runtime/config`（`env > file > default` + fail-fast/回滚）+ `RuntimeRecorder` 单写 + diagnostics replay + quality gate。
- 对齐主流框架时，优先补齐“可互操作 contract”缺口（guardrail precedence、memory 分层治理、OTel tracing/eval），避免散点功能堆叠。

补充参考（主流框架实现与设计查询，2026-03-31 对齐）：
- 本轮“无遗漏”对比项目（官方文档优先）：
  - Coding Agent Runtime：Claude Code、OpenAI Codex、DeerFlow 2.0（明确采用 2.0 口径，不混用 1.x 结论）。
  - Agent 编排框架：LangGraph、LlamaIndex Workflows、AutoGen、Semantic Kernel、CrewAI、Agno、AgentScope。
  - Memory 框架/引擎：Mem0、Zep、OpenViking、OpenClaw（并回看当前内置 filesystem memory 实现）。
- 对齐维度统一采用 7 项：`权限/审批`、`sandbox 边界`、`subagent/多 agent 编排`、`memory 分层与生命周期`、`tool/MCP 接入治理`、`HITL 中断恢复`、`observability/eval`。
- 关键实现信号（用于约束 A58+ 设计，不额外开新主线）：
  - Claude Code：managed/project/user 分层配置 + 权限规则、hook 事件化拦截、subagent 粒度权限与 MCP/memory 作用域；
  - Codex：`sandbox_mode` 与 `approval_policy` 分离治理、workspace-write 默认模型、cloud setup/agent 两阶段与 agent phase 默认断网、AGENTS.md 分层覆盖；
  - DeerFlow 2.0：local/docker/k8s sandbox 模式、host bash 默认关闭、subagent 并行与上下文隔离、local long-term memory、LangSmith tracing；
  - LangGraph/AutoGen/LlamaIndex/Semantic Kernel：强调持久化 checkpoint、HITL interrupt/resume、工作流级状态可回放；
  - CrewAI/Agno：强调角色编排、memory 与 tracing 结合、团队级任务分解与可观测；
  - AgentScope：强调 lifecycle hooks + middleware、统一 state/session 管理、plan notebook 与实时双向事件协议；
  - Mem0/Zep/OpenViking/OpenClaw：强调多层 memory（session/user/agent）、检索/重排、保留策略与 provider 互换能力。
- 一次性补齐项目归并（保持现有优先级，不再拆平行提案）：
  - A58：统一策略栈 precedence（action/sandbox/egress/allowlist/admission）+ 决策解释链，补齐跨入口判定一致性；
  - A59：一次性补齐 memory scope、写入模式、检索质量、生命周期（retention/ttl/forget）与 builtin filesystem v2 治理；
  - A60：统一 token/tool/sandbox/memory 成本与时延预算 admission 规则；
  - A61：补齐 OTel tracing 语义映射、agent eval contract，并合并 local/distributed evaluator 执行治理；
  - A65：补齐 agent lifecycle hooks + tool middleware 合同，统一横切扩展面；
  - A66：补齐统一 state/session snapshot 合同，打通跨模块恢复与迁移；
  - A67：补齐 ReAct plan notebook + plan-change hook 合同，增强动态计划可控性；
  - A68：实时双向事件协议专项（按业务触发），补齐 cancel/resume 与事件幂等合同；
  - A63：代码整合收敛专项（清理临时代码/文档、命名语义化、目录结构收敛），以“语义不变”为硬约束；
  - A64：工程优化&性能优化专项（goroutine pool、buffer/slice pool、批量导出、Context Assembler 热路径治理），以“语义不变”为硬约束；
  - A62：补齐“交付易用性”example pack（主要 agent 模式一站式示例与可回归冒烟）；
- 执行约束：A58-A61 负责核心 runtime contract 缺口，A65-A68 负责 agent runtime 基座能力补齐（A68 按实时交互需求触发），A63 负责代码整合收敛，A64 负责非语义性能工程化，A62 在前述能力相对稳定后承担交付易用性收口（example pack）；除非战略边界变化，不再新增同域提案，避免重复提案与重复改造。

参考链接（本轮核对，滚动更新）：
- Claude Code docs（permissions/hooks/subagents/memory/MCP）：<https://docs.anthropic.com/en/docs/claude-code>
- OpenAI Codex docs（sandboxing/approvals/environments/AGENTS.md/subagents）：<https://developers.openai.com/codex>
- DeerFlow 2.0（README + CONFIGURATION）：<https://github.com/bytedance/deer-flow>
- PocketFlow docs（design patterns / agent modes）：<https://the-pocket.github.io/PocketFlow/design_pattern/>
- LangGraph docs（persistence/HITL/interrupt）：<https://langchain-ai.github.io/langgraph/>
- CrewAI docs：<https://docs.crewai.com/>
- Agno docs：<https://docs.agno.com/>
- AgentScope docs：<https://doc.agentscope.io/>
- AgentScope GitHub：<https://github.com/modelscope/agentscope>
- AutoGen docs：<https://microsoft.github.io/autogen/>
- LlamaIndex docs：<https://docs.llamaindex.ai/>
- Semantic Kernel docs：<https://learn.microsoft.com/semantic-kernel/>
- Mem0 docs：<https://docs.mem0.ai/>
- Zep docs：<https://help.getzep.com/>
- OpenViking docs：<https://volcengine-openviking.mintlify.app/>
- OpenClaw docs：<https://docs.openclaw.ai/>

与在研项目的先后顺序（强依赖）：
1. A58（已归档，P1）：policy precedence + decision trace contract（优先承接跨层策略冲突风险）。
2. A59（已归档，P1）：memory scope + builtin filesystem memory v2 治理 contract（scope/write_mode/injection_budget/lifecycle/search 与 gate 已收口）。
3. A60（已归档，P2）：runtime 成本/时延预算与 admission contract（原 A59 顺延）。
4. A61（进行中，P2）：OTel tracing + agent eval 互操作 contract（含 local/distributed evaluator 执行治理）。
5. A65（新增，P2）：agent lifecycle hooks + tool middleware contract。
6. A66（新增，P2）：unified state/session snapshot contract。
7. A67（新增，P2）：react plan notebook + plan-change hook contract。
8. A68（新增，P2）：realtime event protocol + interrupt/resume contract（按业务触发）。
9. A63（新增，P2）：codebase consolidation and semantic labeling contract（代码收敛与语义化整顿）。
10. A64（新增，P2）：engineering/performance optimization contract（语义不变前提下性能收敛）。
11. A62（新增，P2）：delivery usability example pack contract（主要 agent 模式示例收口）。

备选项目说明（避免“单一路线”误解）：
- A61/A65/A66/A67/A68/A63/A64/A62 组成后续备选池，默认按上方顺序推进，但允许按风险信号前置切换，不要求机械串行实施。
- A61 正在实施，A58/A59/A60 已归档；A58 作为“跨策略层优先级治理”主提案，用于降低联调阶段语义冲突风险。
- 前置切换规则（示例）：
  - 若 A58 联调出现同一请求在 ActionGate/S2/sandbox/admission 判定不一致：优先在 A58 内增量吸收，不再拆平行提案。
  - 若 A58/A59 联调出现 memory 检索召回不足、注入不可解释、或本地文件引擎恢复/索引一致性风险：优先在 A59 内增量吸收。
  - 若成本或 P95 抖动在 A58/A59 联调窗口成为阻塞：优先在 A60 内增量吸收，不再拆平行预算 admission 提案。
  - 若 tracing 字段跨后端解释不一致，或评测执行耗时过长/不可续跑：A61 可前置实施（含 distributed eval 执行治理）。
  - 若业务扩展频繁出现横切逻辑重复接线（审计、限流、缓存、鉴权）：A65 可前置实施。
  - 若跨模块恢复/迁移需要统一状态导入导出：A66 可前置实施。
  - 若 ReAct 场景出现计划漂移不可解释或计划恢复不稳定：A67 可前置实施。
  - 若产品侧进入实时语音/实时协作阶段：A68 可前置实施。
  - 若仓库出现临时代码/文档积压、Axx 文案耦合扩散、模块命名与职责漂移：A63 可前置实施。
  - 若 CPU/GC 抖动、goroutine 峰值、allocs/op 退化成为主线风险：A64 可前置实施，按 `A64-S1 -> S2 -> S3 -> S4 -> S5 -> S6 -> S7 -> S8 -> S9 -> S10` 风险链路逐项吸收（允许按瓶颈信号调整顺序）。
  - 若外部团队接入/迁移周期过长、样例复用率低、或示例与主链路契约脱节：A62 可前置实施（但需明确冻结口径，避免反复返工）。
- 无论是否前置切换，均不得改写 A56 已归档与 A57 已冻结范围，只允许在其完成后做增量扩展。

提案 A58（已归档）：`introduce-policy-precedence-and-decision-trace-contract-a58`
- 目标：统一 ActionGate、Security S2、sandbox action/egress、adapter allowlist、readiness/admission 的策略判定优先级与解释链路，防止并行改造后出现判定冲突。
- 范围：
  - 固化跨策略层 precedence matrix 与 deterministic tie-break；
  - 统一 deny source taxonomy 与 explainability 字段；
  - 增加 `policy_stack.v1` replay fixture 与 drift 分类；
  - 增加独立 `check-policy-precedence-contract.*` gate。
- 当前落地（已完成）：
  - `check-policy-precedence-contract.sh/.ps1` 已接入 `check-quality-gate.sh/.ps1`；
  - CI 已暴露独立 required-check 候选 `policy-precedence-gate`；
  - replay 已覆盖 `policy_stack.v1` 与 mixed compatibility（`a50.v1` + `react.v1` + `sandbox_egress.v1` + `policy_stack.v1`）。
- Why now（紧急性）：A57 联调改动 runner/sandbox/readiness/admission，若缺少统一 precedence contract，极易产生“同请求不同入口判定不一致”的高风险回归。

提案 A57：`introduce-sandbox-egress-governance-and-adapter-allowlist-contract-a57`（已归档）
- 目标：补齐 sandbox 网络外呼治理（egress policy）与 adapter 供应链 allowlist 契约，形成“执行隔离 + 出口治理 + 激活准入”闭环。
- 范围：`security.sandbox.egress.*`、`adapter.allowlist.*`、readiness/admission finding、taxonomy、replay drift 与 conformance matrix。
- 门禁：`check-sandbox-egress-allowlist-contract.sh/.ps1`（已纳入 `check-quality-gate.sh/.ps1`），CI 独立 required-check 候选为 `sandbox-egress-allowlist-gate`。
- 依赖：复用 A51/A52/A53 sandbox taxonomy 与 adapter manifest 激活边界，不新增平行安全语义。
- 启动条件：存在合规审计或外部 adapter 引入规模上升，需要可审计可阻断的 egress/allowlist 治理。

备选 A59（合并版）：`introduce-memory-scope-and-builtin-filesystem-v2-governance-contract-a59`
- 目标：在 A54 SPI 基线之上合并推进两类能力：
  - memory scope 与注入预算治理（`session|project|global` + injection budget）；
  - builtin filesystem memory v2（本地检索与索引能力增强、恢复一致性与可观测治理）。
- 对标主流实现（参考 OpenClaw + Agno memory 路径）的补齐方向：
  - memory 语义分层治理：区分 `session_history`、`user_memory`、`session_summary` 三类语义及注入优先级；
  - memory 写入策略治理：支持 `automatic` 与 `agentic` 两类写入模式，并冻结回填窗口与幂等规则；
  - 本地索引检索增强：支持关键词检索与语义检索协同（hybrid retrieval）；
  - 索引生命周期治理：文件变更触发增量更新、provider/model 变化触发全量重建；
  - 结果后处理治理：去冗余重排（MMR 类）与时间衰减（recency boost）可配置；
  - 索引与存储一致性：WAL/snapshot 基线之上增加校验与恢复漂移检测；
  - 记忆生命周期治理：补齐 retention/ttl/forget 策略与 fail-fast 校验，避免 memory 无界增长；
  - 检索质量基线：新增 memory retrieval quality 回归套件（recall/top-k 命中率等）并纳入 gate；
  - 多源接入：memory 主文件 + 按 scope 的附加路径策略（保持 fail-fast + allowlist 边界）。
- 范围（合并后）：
  - `runtime.memory.mode.*`（`automatic|agentic` 写入策略与回填窗口）
  - `runtime.memory.scope.*`
  - `runtime.memory.injection_budget.*`
  - `runtime.memory.lifecycle.*`（retention/ttl/forget）
  - `runtime.memory.search.*`（hybrid/query/rerank/temporal_decay/index_update）
  - QueryRuns additive 字段 + `memory_scope.v1`、`memory_search.v1`、`memory_lifecycle.v1` mixed replay fixtures
  - 独立 gate：`check-memory-scope-and-search-contract.*`
- 依赖：A54 memory facade/profile/readiness 字段稳定后扩展，避免与 A56/A57 实施冲突。
- 启动条件：出现 memory 注入不可解释、检索召回不足、或本地文件 memory 在恢复/索引一致性上的风险信号。

备选 A60：`introduce-runtime-cost-latency-budget-and-admission-contract-a60`
- 目标：统一 token/tool/sandbox/memory 成本与时延预算，建立 admission 侧 fail-fast 与降级策略。
- 启动条件：成本或 P95 抖动成为主线瓶颈。

备选 A61（新增）：`introduce-otel-tracing-and-agent-eval-interoperability-contract-a61`
- 目标：补齐主流框架常见的“可观测 + 评测”互操作治理，降低跨平台对接成本并固定回归口径。
- 对标主流（OpenAI Agents / Agno / CrewAI / AgentScope）的补齐方向：
  - tracing 语义：对齐 OTel 场景下 run/model/tool/mcp/memory/hitl 关键 span/attribute 映射；
  - tracing 导出：保证不引入平台控制面的前提下，支持主流 OTel backend 稳定接入；
  - 评测基线：新增最小 agent eval contract（任务成功率、工具调用正确率、拒绝/拦截准确率、cost-latency 约束）；
  - 评测执行治理（合并项）：在 A61 内一次性支持 `local|distributed` evaluator execution、分片汇总、失败重试、断点续跑与结果幂等聚合；
  - 回放与门禁：增加 `otel_semconv.v1`、`agent_eval.v1`、`agent_eval_distributed.v1` fixtures，新增 `check-agent-eval-and-tracing-interop-contract.*`。
- 依赖：A55 observability export + diagnostics bundle 稳定后扩展；建议在 A58 decision trace 字段冻结后接入。
- 启动条件：出现 tracing 字段跨后端解释不一致、外部可观测平台接线成本高、或缺少稳定 agent 质量回归基线。
- 约束项（新增）：
  - Non-goals：不引入托管评测控制面、远程评测任务调度服务、平台化 UI/RBAC/多租户运维面板。
  - Gate 边界断言：`check-agent-eval-and-tracing-interop-contract.*` 必须包含 `control_plane_absent` 断言（distributed execution 仅作为库内执行策略，不新增服务化控制面依赖）。

备选 A65（新增）：`introduce-agent-lifecycle-hooks-and-tool-middleware-contract-a65`
- 目标：统一 agent 生命周期 hooks 与 tool middleware 合同，减少业务侧横切逻辑重复接线。
- 范围（简版）：
  - lifecycle hooks：`before_reasoning|after_reasoning|before_acting|after_acting|before_reply|after_reply`；
  - tool middleware：冻结 onion-chain 执行顺序、上下文透传、错误冒泡与超时隔离；
  - 配置治理：`runtime.hooks.*`、`runtime.tool_middleware.*` 统一 `env > file > default` 与 fail-fast/回滚；
  - skill discovery source（一次补齐）：在保持 `AGENTS.md` 兼容的前提下，新增按目录路径加载技能的统一配置口径；
  - discovery 配置（简版）：`runtime.skill.discovery.mode`（`agents_md|folder|hybrid`）与 `runtime.skill.discovery.roots`（目录列表）；
  - discovery 治理（简版）：多来源合并顺序 deterministic、重复技能去重规则固定、非法路径/不可读目录 fail-fast 并原子回滚；
  - 预处理接缝（一次补齐）：将 `Discover/Compile` 挂入 Run/Stream 前统一预处理阶段，支持开关控制与失败策略；
  - 预处理配置（简版）：`runtime.skill.preprocess.enabled`、`runtime.skill.preprocess.phase=before_run_stream`、`runtime.skill.preprocess.fail_mode=fail_fast|degrade`；
  - bundle 映射（一次补齐）：冻结 `SkillBundle -> prompt augmentation` 与 `SkillBundle -> tool whitelist` 的合同映射与冲突仲裁顺序；
  - 映射配置（简版）：`runtime.skill.bundle_mapping.prompt_mode`、`runtime.skill.bundle_mapping.whitelist_mode`、`runtime.skill.bundle_mapping.conflict_policy`；
  - 回放与门禁：新增 `hooks_middleware.v1` fixture 与 `check-hooks-middleware-contract.*`。
- 硬约束（简版）：
  - 不绕过 A58 precedence、A57 安全治理与 `RuntimeRecorder` 单写入口；
  - skill discovery source 切换不得绕过既有 trigger scoring/budget 与 skill 观测事件口径；
  - `Discover/Compile` 预处理与 `SkillBundle` 映射在 Run/Stream 下必须等价，不得引入“预处理只在单入口生效”的分叉；
  - whitelist 映射不得突破 A57 adapter allowlist 与 sandbox/egress 治理上界。
  - Hook/Middleware 失败语义必须 deterministic，不引入 Run/Stream 分叉。
- 当前状态：占位提案（简版）。

备选 A66（新增）：`introduce-unified-state-and-session-snapshot-contract-a66`
- 目标：统一 runtime state/session snapshot 导入导出合同，打通跨模块恢复、迁移与重放。
- 范围（简版）：
  - state surface：runner/session、memory、workflow/composer/scheduler 的统一 state descriptor；
  - snapshot contract：版本化 schema、部分恢复、字段兼容窗口、冲突 fail-fast；
  - 恢复治理：增量恢复、幂等重放、跨后端一致性检查；
  - 回放与门禁：新增 `state_session_snapshot.v1` fixture 与 `check-state-snapshot-contract.*`。
- 硬约束（简版）：
  - 不重写已有 checkpoint/snapshot 语义，仅做统一合同层；
  - 不引入平台控制面或远程状态服务依赖。
- 当前状态：占位提案（简版）。

备选 A67（新增）：`introduce-react-plan-notebook-and-plan-change-hook-contract-a67`
- 目标：补齐 ReAct 动态计划治理（Plan Notebook）与计划变更 hook，提升复杂任务可控性与可解释性。
- 范围（简版）：
  - plan notebook：`create|revise|complete|recover` 生命周期；
  - plan-change hook：计划变更前后回调、变更原因与上下文快照；
  - 观测与回放：新增计划漂移与恢复漂移字段、`react_plan_notebook.v1` fixture；
  - 门禁：新增 `check-react-plan-notebook-contract.*`。
- 硬约束（简版）：
  - 复用 A56 ReAct termination taxonomy，不新增平行 loop 语义；
  - 计划治理不得绕过 A58 决策链与 A57 安全链路。
- 当前状态：占位提案（简版）。

备选 A68（新增）：`introduce-realtime-event-protocol-and-interrupt-resume-contract-a68`
- 目标（简版）：补齐实时双向事件协议（server/client）与 interrupt/resume 合同，支撑实时交互场景。
- 范围（简版）：
  - 事件协议：请求、增量输出、取消、恢复、确认、错误的 canonical event taxonomy；
  - 会话治理：事件去重、顺序保证、重连恢复与幂等处理；
  - 回放与门禁：新增 `realtime_event_protocol.v1` fixture 与 `check-realtime-protocol-contract.*`。
- 硬约束（简版）：
  - 不引入平台化实时网关或托管控制面；
  - 协议语义必须与 A56/A58/A67 的主链路解释字段保持一致。
- 约束项（新增）：
  - Non-goals：不引入托管会话路由/连接管理控制面、实时 SaaS 运维面板或平台级常驻网关服务。
  - Gate 边界断言：`check-realtime-protocol-contract.*` 必须包含 `realtime_control_plane_absent` 断言（协议实现仅限库内 contract + adapter 接缝，不新增网关服务依赖）。
- 当前状态：占位提案（简版，按业务实时化需求触发）。

备选 A63（新增）：`introduce-codebase-consolidation-and-semantic-labeling-contract-a63`
- 目标（简版）：在不改变运行时语义前提下，完成仓库“代码与文档收敛整顿”，降低历史负担与命名歧义。
- 范围（简版）：
  - 临时文档/目录治理：清理或归档 `docs/drafts`、示例与脚手架生成物等临时资产，建立统一收口规则；
  - 离线生成物治理：收敛 `examples/adapters/_a23-offline-work/*` 这类离线 scaffold 产物，仅保留最小可复现样本与索引说明，其余转离线缓存或清理；
  - Context Assembler 命名收敛：将 `ca/ca2/ca3/ca4` 相关实现对外统一为语义化 `ca` 口径（内部可分层，但不再暴露编号式心智）；
  - Axx 文案语义化：仓库内面向用户/维护者的 Axx 编号描述迁移为语义化名称，Spec 编号映射集中在索引文档维护，不在模块 README/配置注释中散落耦合。
  - 阶段性工具命名治理：`cmd/*` 与 `scripts/*` 中编号化阶段命名（如 `ca3-threshold-*`、`ca4-benchmark-*`）统一补充语义别名与映射，避免新入口继续放大编号耦合；
  - 临时注释与占位清理：清理 `TODO/future milestone` 类临时注释并转化为 roadmap/index 可追踪事项，避免代码内长期悬挂。
- 硬约束（简版）：
  - 不改变 Run/Stream、readiness/admission、reason taxonomy、diagnostics/replay 契约语义；
  - 不删除仍被 gate/fixture 使用的兼容数据，仅允许“迁移+别名+映射”方式收敛；
  - 所有重命名或目录调整必须提供可回滚路径与兼容跳板。
  - 编号化保留边界：`openspec/changes` 与 `openspec/changes/archive` 作为历史索引允许保留编号，代码与用户向文档默认使用语义名称。
- 当前状态：占位提案（简版），待 A67/A68（若启用）主链路冻结后基于当时代码状态展开详细清单与实施步骤。

备选 A64（新增）：`introduce-engineering-and-performance-optimization-contract-a64`
- 目标（简版）：在“语义不变”前提下推进工程优化与性能优化（如 goroutine pool、buffer/slice pool、导出批处理等常规路径）。
- 子项目（性能治理，优先落地）：
  - A64-S1：`context-assembler-loop-hotpath-governance`
  - 目标：降低每轮 `Assemble` 固定开销与长跑内存累积风险，在不改变 CA1/CA2/CA3/CA4 语义前提下提升稳定吞吐。
  - 范围（第一批）：
    - 为 `prefixCache` / `ca3State` 增加 run-finished 清理与 TTL/LRU 上限治理，避免常驻进程无界增长；
    - 为 context journal 增加可开关批量写入路径（默认保持同步语义），并补齐 flush/异常中断边界测试；
    - 增加 CA3 stage2 “无增量跳过”优化开关（仅在 stage2 未追加有效上下文且输入签名不变时跳过第二次 CA3）；
    - 为 `stage2 provider=file` 增加索引化读取或分段扫描策略，降低大文件线性扫描成本；
    - 为 `stage2 provider=external(http/rag/db/elasticsearch)` 增加请求/响应编解码快路径与有界缓冲复用，降低 `json marshal/unmarshal + body read` 抖动；
    - 增加热点基准与回归门禁：`BenchmarkContextAssemblerLoop*`、`BenchmarkCA3Stage2Pass*`、`BenchmarkStage2FileProvider*`。
  - 非目标（第一批）：
    - 不修改 A60 成本/时延 budget admission 公式与降级动作；
    - 不调整 Run/Stream 行为、reason taxonomy、diagnostics 字段语义。
  - A64-S2：`runtime-recorder-and-diagnostics-hotpath-governance`
  - 目标：降低 `run.finished` 大负载映射、query 聚合与排序复制带来的 CPU/GC 抖动，保持 recorder/query 语义不变。
  - 范围（第一批）：
    - 为 `RuntimeRecorder` 增加可复用映射缓冲与按需字段投影，减少 `run.finished` 事件大对象重复分配；
    - 为 diagnostics store 查询路径增加可开关索引/分页游标策略，减少全量筛选 + 排序 + 复制；
    - 为 `MailboxAggregates`/P95 聚合引入有界统计优化（保持输出字段与解释口径不变）；
    - 增加 `BenchmarkRuntimeRecorderRunFinished*`、`BenchmarkDiagnosticsQueryRuns*`、`BenchmarkDiagnosticsMailboxAggregates*` 与回归 gate。
  - 非目标（第一批）：
    - 不改写 `RuntimeRecorder` 单写入口契约；
    - 不变更 QueryRuns/QueryMailbox 对外字段、排序解释与 replay 语义。
  - A64-S3：`scheduler-mailbox-file-backend-persistence-governance`
  - 目标：降低 file backend 高频全量持久化开销，控制 I/O 放大与锁竞争，同时维持 crash recovery 与幂等语义。
  - 范围（第一批）：
    - 为 scheduler/mailbox file store 增加可开关增量刷盘或批量合并持久化策略（默认保持现有强一致行为）；
    - 将 composer recovery file store 一并纳入持久化治理，控制全量 `marshal + atomic write` 放大；
    - 为 scheduler task-board query 增加可开关索引/缓存与增量分页策略，降低全量过滤+排序路径成本；
    - 引入持久化节流/批次参数与 fail-fast 校验，并补齐热更新回滚测试；
    - 增加 `BenchmarkSchedulerFileStorePersist*`、`BenchmarkMailboxFileStorePersist*` 与崩溃恢复一致性回归 gate。
  - 非目标（第一批）：
    - 不改变 task lifecycle、manual control、async-await 终态裁决；
    - 不引入外部 MQ 或服务化存储控制面。
  - A64-S4：`mcp-transport-invoke-and-event-emission-hotpath-governance`
  - 目标：降低 MCP stdio/http 调用链 goroutine 峰值与事件发射分配开销，保持 call contract 与诊断解释稳定。
  - 范围（第一批）：
    - 为 stdio/http client 的 invoke 路径增加有界 worker/复用策略开关，抑制每调用 goroutine 激增；
    - 为 MCP 事件发射 map 构建引入模板复用与延迟填充，减少短生命周期分配；
    - 增加 `BenchmarkMCPInvokePath*`、`BenchmarkMCPEventEmit*` 与 transport 回归 gate。
  - 非目标（第一批）：
    - 不调整 MCP 对外 API、重试/超时语义或错误 taxonomy；
    - 不改变已有 tracing/diagnostics 字段口径。
  - A64-S5：`skill-loader-discover-compile-io-and-scoring-governance`
  - 目标：降低 skill discover/compile 重复 I/O 与评分路径开销，保证 `agents.md|folder|hybrid` 解析结果一致。
  - 范围（第一批）：
    - 为 discover/compile 建立可开关元数据缓存与文件读取复用，避免同轮重复解析；
    - 为评分 tokenization/sort 路径引入有界缓存与短路策略（仅优化实现，不改分数语义）；
    - 增加 `BenchmarkSkillLoaderDiscover*`、`BenchmarkSkillLoaderCompile*`、`BenchmarkSkillSelectionScore*` 与回归 gate。
  - 非目标（第一批）：
    - 不改 discovery precedence、去重顺序、`SkillBundle -> prompt/tool whitelist` 映射语义；
    - 不替代 A65/A62 的 skill contract 主线治理职责。
  - A64-S6：`memory-filesystem-engine-write-query-index-governance`
  - 目标：降低 filesystem memory 引擎 WAL 写入、查询排序与索引重建成本，保持 A59 scope/lifecycle/search 契约不变。
  - 范围（第一批）：
    - 为 WAL 增加可开关批量 fsync/组提交策略（默认保留现有 durability 语义）；
    - 为 query 路径引入命名空间级索引/缓存与稳定排序复用，降低每次全量扫描；
    - 为索引 checksum/compaction 增加分段重建与后台节流治理；
    - 增加 `BenchmarkMemoryFilesystemWrite*`、`BenchmarkMemoryFilesystemQuery*`、`BenchmarkMemoryFilesystemCompaction*` 与回归 gate。
  - 非目标（第一批）：
    - 不修改 A59 的 scope resolution、retrieval quality 阈值、lifecycle policy 与可解释字段；
    - 不引入新 memory provider 协议面或第二套事实源。
  - A64-S7：`runner-loop-and-local-dispatch-hotpath-governance`
  - 目标：降低 Runner 每轮 timeline/run-finished 构造开销与 local tool dispatch 分类开销，稳定高迭代场景吞吐。
  - 范围（第一批）：
    - 为 Runner 引入 run-scope 配置快照/派生值复用，减少循环内重复 `EffectiveConfig()` 读取与大对象复制；
    - 为 `emitTimeline` / `runFinishedPayload` 增加可复用 payload 构建策略，降低 map 分配与拷贝；
    - 为 local dispatcher 的 `drop_low_priority` 分类链路增加关键字预编译与签名缓存，避免每调用重复排序/归一化；
    - 增加 `BenchmarkRunnerLoopHotpath*`、`BenchmarkRunnerTimelineEmit*`、`BenchmarkLocalDispatchPriorityClassify*` 与回归 gate。
  - 非目标（第一批）：
    - 不改变 action timeline 事件顺序、字段语义与 reason taxonomy；
    - 不改变 backpressure/retry/fail-fast 决策语义。
  - A64-S8：`provider-adapter-stream-and-decode-hotpath-governance`
  - 目标：降低 OpenAI/Anthropic/Gemini 适配器在流式事件映射与非流式解码阶段的分配与序列化开销。
  - 范围（第一批）：
    - 为 provider stream 事件映射引入 meta/payload 复用策略，减少每事件 map 临时分配；
    - 为 tool-call 参数解码增加快速路径与有界缓冲复用，降低高频 `json.Unmarshal` 抖动；
    - 为非流式响应解码优先使用 typed 字段读取，减少全量 `json.Marshal + gjson` 回退路径触发；
    - 增加 `BenchmarkProviderStreamEventMap*`、`BenchmarkProviderResponseDecode*` 与 provider parity gate。
  - 非目标（第一批）：
    - 不改变 provider capability 判定、tool_call 触发条件、事件顺序与 token usage 口径；
    - 不改写已有 provider 错误分类与重试语义。
  - A64-S9：`runtime-config-readpath-and-policy-resolve-hotpath-governance`
  - 目标：降低高并发场景下 runtime config 读取与 MCP policy 解析开销，保持配置治理与热更新语义稳定。
  - 范围（第一批）：
    - 为 runtime config 增加只读快照引用/派生缓存机制，减少频繁值拷贝；
    - 为 MCP runtime policy resolve 增加按 `profile + explicit override` 的可失效缓存（reload 后自动失效）；
    - 为关键热路径补齐 `BenchmarkRuntimeConfigReadPath*`、`BenchmarkMCPPolicyResolve*` 与回归 gate。
  - 非目标（第一批）：
    - 不改变 `env > file > default`、fail-fast 与热更新原子回滚语义；
    - 不改写 policy precedence、admission 与 sandbox rollout contract 字段。
  - A64-S10：`observability-event-pipeline-throughput-governance`
  - 目标：降低 observability 事件管线（dispatcher/logger/exporter）在高事件率场景下的分配与串行阻塞开销。
  - 范围（第一批）：
    - 为 runtime exporter 增加批量导出与批次聚合开关，替代逐事件 `ExportEvents([]event{...})` 热路径；
    - 为 dispatcher 增加可配置 fanout 策略与 handler 隔离治理，避免慢 handler 放大主链路延迟；
    - 为 JSON logger 增加编码器/缓冲复用与最小化字段构建路径，降低 per-event `map + json.Marshal` 开销；
    - 增加 `BenchmarkRuntimeExporterBatch*`、`BenchmarkEventDispatcherFanout*`、`BenchmarkJSONLoggerEmit*` 与回归 gate。
  - 非目标（第一批）：
    - 不改变事件 schema、timeline 序列、RuntimeRecorder 单写入口和 replay 字段语义；
    - 不引入平台化 observability 控制面或外置必选依赖。
- 强门禁（A64 子项共用，阻断合入）：
  - 必须新增并接入 `check-a64-semantic-stability-contract.sh/.ps1`，阻断“对外语义漂移”：
    - Run/Stream 行为等价；
    - diagnostics schema 与 reason taxonomy 不漂移；
    - replay fixture idempotency 稳定。
  - 必须新增并接入 `check-a64-performance-regression.sh/.ps1`，阻断关键基准退化（`ns/op`、`allocs/op`、`B/op`）；
  - 必须接入 `A64 impacted-contract suites` 校验（按改动模块选择主干 contract suites），要求主干 contract/replay suites 全绿且无漂移豁免；
  - A64 任一子项未通过 contract/replay/perf gate，不允许合入主干。
- `A64 impacted-contract suites` 模块映射（最低必跑，shell/PowerShell 必须语义等价）：
  - S1（context assembler + stage2 provider + journal）：
    - `go test ./context/assembler ./context/provider ./context/journal -count=1`
    - `scripts/check-diagnostics-replay-contract.sh/.ps1`
  - S2（runtime recorder + diagnostics）：
    - `scripts/check-diagnostics-replay-contract.sh/.ps1`
    - `scripts/check-diagnostics-query-performance-regression.sh/.ps1`
  - S3（scheduler/mailbox/composer recovery + task-board query）：
    - `scripts/check-multi-agent-shared-contract.sh/.ps1`
    - `go test ./orchestration/scheduler ./orchestration/composer -count=1`
  - S4（MCP transport invoke path）：
    - `go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1`
    - `scripts/check-multi-agent-shared-contract.sh/.ps1`
  - S5（skill loader discover/compile/scoring）：
    - `go test ./skill/loader ./runtime/config -count=1`
  - S6（memory filesystem engine）：
    - `scripts/check-memory-contract-conformance.sh/.ps1`
    - `scripts/check-memory-scope-and-search-contract.sh/.ps1`
  - S7（runner loop + local dispatch）：
    - `scripts/check-security-policy-contract.sh/.ps1`
    - `scripts/check-security-event-contract.sh/.ps1`
    - `scripts/check-security-delivery-contract.sh/.ps1`
    - `scripts/check-security-sandbox-contract.sh/.ps1`
  - S8（model provider adapters）：
    - `scripts/check-react-contract.sh/.ps1`
  - S9（runtime config read-path + policy resolve）：
    - `scripts/check-policy-precedence-contract.sh/.ps1`
    - `scripts/check-runtime-budget-admission-contract.sh/.ps1`
    - `scripts/check-sandbox-rollout-governance-contract.sh/.ps1`
  - S10（observability dispatcher/logger/exporter pipeline）：
    - `scripts/check-observability-export-and-bundle-contract.sh/.ps1`
    - `scripts/check-diagnostics-replay-contract.sh/.ps1`
  - 横切兜底（所有 A64 子项合并前必跑）：
    - `scripts/check-quality-gate.sh/.ps1`
- 硬约束（简版）：
  - 不改变 Run/Stream、backpressure、fail_fast、timeout/cancel、reason taxonomy、decision trace 语义；
  - 不绕过现有 contract gate 与 replay 约束；
  - 所有优化都必须可开关、可回滚。
- 当前状态：占位提案（含 A64-S1~S10 子项目），详细 contract/fixture/gate 清单待 A63 收敛后展开。

备选 A62（新增）：`introduce-delivery-usability-agent-mode-example-pack-contract-a62`
- 目标：将“主要 agent 模式”沉淀为可直接复用、可回归验证、与主线 contract 同步的 example pack，提升交付易用性与迁移效率。
- 模式覆盖（最低要求）：
  - `single agent`（最小 chat/任务执行主链路）；
  - `agent with skill`（skills 装载与触发评分、工具协同；同时覆盖 `AGENTS.md` 与目录路径配置两类加载入口）；
  - `react agent`（推理-行动-观察闭环，Run/Stream 等价）；
  - `multi agent`（至少包含协作链路与异步通道两类范式）；
  - `sandbox-governed agent`（sandbox/egress/allowlist 治理链路可演示）。
- 对标参考（示例组织方法）：
  - PocketFlow design patterns：Agent / Workflow / Multi-Agent 的模式化分层与最小可运行示例组织；
  - 本仓库 `examples/01-09` 现有主链路示例（避免重复造样例，优先改造为统一模式矩阵）。
- 范围：
  - 建立 `examples/agent-modes` 统一目录或等价索引（支持按模式检索）；
  - 每种模式提供 `minimal` + `production-ish` 两档示例（前者用于上手，后者用于治理链路演示）；
  - 示例统一注入 diagnostics/tracing 标记，确保可进入 replay 与 gate；
  - 提供模式级 `README` 与迁移指引（从旧示例到模式化示例的映射）。
- Gate：
  - `check-agent-mode-examples-smoke.sh/.ps1`（按模式矩阵执行最小冒烟）
  - required-check 候选：`agent-mode-examples-smoke-gate`
- 依赖：A57-A61 与 A65-A68（若启用）主链路冻结，且 A63/A64 收敛完成后实施。
- 启动条件：新增团队接入成本偏高、PoC 转生产迁移慢、或示例与 contract 漂移信号出现。

A58-A62 验收清单（contract / replay / gate，A62 为后置收口项）：

统一验收前提（当前已展开提案共用）：
- 配置治理：所有新增配置必须遵守 `env > file > default`，非法值 fail-fast，热更新失败原子回滚。
- 观测治理：运行态写入仅走 `RuntimeRecorder` 单写入口；QueryRuns 仅新增 additive 字段，禁止破坏既有字段语义。
- 回放治理：每个提案至少新增 1 个 replay fixture 与 drift 分类，并接入 docs index。
- 门禁治理：每个提案至少新增 1 个独立 contract gate（shell + PowerShell 语义等价），并接入 `check-quality-gate.*`。
- 兼容治理：Run/Stream 语义保持等价；未经提案显式声明，不变更公开 API 破坏性行为。

A58 验收清单：`introduce-policy-precedence-and-decision-trace-contract-a58`
- Contract 字段（最小集）：
  - `runtime.policy.precedence.version`
  - `runtime.policy.precedence.matrix.*`
  - `runtime.policy.tie_breaker.*`
  - `runtime.policy.explainability.enabled`
  - QueryRuns additive：`policy_decision_path`、`deny_source`、`winner_stage`、`tie_break_reason`
- Replay fixtures：
  - `policy_stack.v1`（覆盖 action/s2/sandbox/egress/allowlist/admission 冲突）
  - drift 分类至少包含：`precedence_conflict`、`tie_break_drift`、`deny_source_mismatch`
- Gate：
  - `check-policy-precedence-contract.sh/.ps1`
  - required-check 候选：`policy-precedence-gate`
- 最小测试矩阵：
  - 单测：precedence matrix 与 tie-break 决策稳定性；
  - 集成：同一请求在 Run/Stream + 不同入口路径下决策一致；
  - 负向：配置冲突/缺失时 fail-fast 与回滚。
- 文档同步：
  - `docs/runtime-config-diagnostics.md`（新增 policy 配置与 decision trace 字段）
  - `docs/mainline-contract-test-index.md`（新增 gate 与 replay 条目）
- 退出条件（DoD）：
  - 联调阶段“同请求不同入口判定不一致”归零；
  - replay 漂移被稳定归类且可复现。

A59 验收清单：`introduce-memory-scope-and-builtin-filesystem-v2-governance-contract-a59`
- Contract 字段（最小集）：
  - `runtime.memory.scope.*`（`session|project|global`）
  - `runtime.memory.write_mode.*`（`automatic|agentic`）
  - `runtime.memory.injection_budget.*`
  - `runtime.memory.lifecycle.*`（retention/ttl/forget）
  - `runtime.memory.search.*`（hybrid/query/rerank/temporal_decay/index_update）
  - QueryRuns additive：`memory_scope_selected`、`memory_budget_used`、`memory_hits`、`memory_rerank_stats`、`memory_lifecycle_action`
- Replay fixtures：
  - `memory_scope.v1`
  - `memory_search.v1`
  - `memory_lifecycle.v1`
  - drift 分类至少包含：`scope_resolution_drift`、`retrieval_quality_regression`、`lifecycle_policy_drift`、`recovery_consistency_drift`
- Gate：
  - `check-memory-scope-and-search-contract.sh/.ps1`
  - required-check 候选：`memory-scope-search-gate`
- 最小测试矩阵：
  - 单测：scope 解析、budget 裁剪、TTL/forget 语义、search/rerank 配置边界；
  - 集成：external SPI 与 builtin filesystem 双路径一致性；
  - 恢复：WAL + snapshot crash recovery、index drift detect；
  - 质量：recall@k / top-k 命中率 / 冗余率回归阈值。
- 文档同步：
  - `memory/README.md`（外部 SPI 与 builtin 模式选择、能力矩阵）
  - `docs/runtime-config-diagnostics.md`（memory 新字段与默认值）
  - `docs/mainline-contract-test-index.md`（memory fixtures + gate）
- 退出条件（DoD）：
  - memory 注入链路可解释（scope/budget/source 全可追踪）；
  - builtin filesystem 在恢复与检索一致性上通过 contract gate 全量回归。

A60 验收清单：`introduce-runtime-cost-latency-budget-and-admission-contract-a60`
- Contract 字段（最小集）：
  - `runtime.admission.budget.cost.*`
  - `runtime.admission.budget.latency.*`
  - `runtime.admission.degrade_policy.*`
  - QueryRuns additive：`budget_snapshot`、`budget_decision`、`degrade_action`
- Replay fixtures：
  - `budget_admission.v1`（不同负载、不同 provider、不同 sandbox 开销）
  - drift 分类至少包含：`budget_threshold_drift`、`admission_decision_drift`、`degrade_policy_drift`
- Gate：
  - `check-runtime-budget-admission-contract.sh/.ps1`
  - required-check 候选：`runtime-budget-admission-gate`
- 最小测试矩阵：
  - 单测：预算计算与夹紧逻辑；
  - 集成：token/tool/sandbox/memory 混合成本下 admission 判定一致；
  - 压测：P95/P99 触发阈值下 degrade 与 fail-fast 行为稳定。
- 文档同步：
  - `docs/runtime-config-diagnostics.md`（budget/admission 字段、阈值示例）
  - `docs/mainline-contract-test-index.md`（budget fixture + gate）
- 退出条件（DoD）：
  - 成本/时延抖动具备可解释 admission 决策；
  - budget 相关回归可被 replay + gate 稳定拦截。

A61 验收清单：`introduce-otel-tracing-and-agent-eval-interoperability-contract-a61`
- Contract 字段（最小集）：
  - `runtime.observability.tracing.otel.*`
  - `runtime.eval.agent.*`
  - `runtime.eval.execution.*`（`mode=local|distributed`、`shard`、`retry`、`resume`、`aggregation`）
  - QueryRuns additive：`trace_export_status`、`trace_schema_version`、`eval_suite_id`、`eval_summary`、`eval_execution_mode`、`eval_job_id`、`eval_shard_total`、`eval_resume_count`
- Replay fixtures：
  - `otel_semconv.v1`
  - `agent_eval.v1`
  - `agent_eval_distributed.v1`
  - drift 分类至少包含：`otel_attr_mapping_drift`、`span_topology_drift`、`eval_metric_drift`、`eval_aggregation_drift`、`eval_shard_resume_drift`
- Gate：
  - `check-agent-eval-and-tracing-interop-contract.sh/.ps1`
  - 边界断言：`control_plane_absent`（禁止托管评测控制面/服务化调度依赖）
  - required-check 候选：`agent-eval-tracing-interop-gate`
- 最小测试矩阵：
  - 单测：span/attribute 映射稳定性；
  - 集成：至少 2 类 OTel backend 兼容冒烟（本地 exporter + 远端 collector）；
  - 评测：任务成功率、工具调用正确率、拒绝/拦截准确率、cost-latency 约束；
  - 执行治理：local/distributed 评测结果聚合等价、分片失败重试、断点续跑幂等。
- 文档同步：
  - `docs/runtime-config-diagnostics.md`（OTel + eval + eval execution 配置）
  - `docs/mainline-contract-test-index.md`（OTel/eval fixtures + gate）
- 退出条件（DoD）：
  - tracing 字段跨后端解释一致；
  - agent 质量回归具备稳定、可复放、可阻断的最小口径；
  - distributed 评测执行具备稳定聚合与断点恢复，不另开平行提案；
  - 维持 `library-first` 形态：不引入托管评测控制面或服务化执行平面。

A62 验收清单：`introduce-delivery-usability-agent-mode-example-pack-contract-a62`
- Contract 字段（最小集）：
  - `runtime.examples.mode_index.version`
  - `runtime.examples.mode_index.required_modes`（`single|skill|react|multi_agent|sandbox_governed`）
  - `runtime.examples.smoke.enabled`
  - QueryRuns additive（示例运行）：`example_mode`、`example_profile`、`example_contract_version`
- Replay fixtures：
  - `example_modes.v1`（覆盖五类模式的最小运行快照）
  - drift 分类至少包含：`example_mode_contract_drift`、`example_readme_runtime_drift`
- Gate：
  - `check-agent-mode-examples-smoke.sh/.ps1`
  - required-check 候选：`agent-mode-examples-smoke-gate`
- 最小测试矩阵：
  - 模式冒烟：`single`、`skill`、`react`、`multi_agent`、`sandbox_governed` 全部可运行；
  - skill 入口一致：`skill` 模式需覆盖 `AGENTS.md`、`folder`、`hybrid` 三类 discovery 配置并验证触发评分与装载结果一致性；
  - skill 预处理接线：`Discover/Compile` 在 `Run` 与 `Stream` 前均可按开关启停，且失败策略与诊断输出保持一致；
  - skill bundle 映射一致：`SkillBundle` 对 prompt augmentation 与 tool whitelist 的映射在 `Run|Stream`、`discover-only|discover+compile`、`on|off` 组合下语义一致；
  - 语义一致：`react` 示例在 Run/Stream 下行为口径一致；
  - 治理一致：`sandbox_governed` 示例可稳定触发 egress/allowlist 判定与解释字段；
  - 文档一致：每个模式 README 都有“前置条件、运行命令、预期输出、失败排查”四段。
- 文档同步：
  - `examples/*/README.md`（模式化结构与索引）
  - `README.md`（Examples 快速入口更新）
  - `docs/mainline-contract-test-index.md`（新增 example gate）
- 退出条件（DoD）：
  - 新接入团队在不读源码前提下可按模式完成端到端跑通；
  - 示例与主线 contract 不再出现长期漂移（通过 smoke gate 持续阻断）。

A65-A68 占位验收口径（简版）：
- A65（hooks + middleware）：
  - 字段：`runtime.hooks.*`、`runtime.tool_middleware.*`、`runtime.skill.discovery.*`、`runtime.skill.preprocess.*`、`runtime.skill.bundle_mapping.*`
  - 回放：`hooks_middleware.v1`、`skill_discovery_sources.v1`（覆盖 `agents_md|folder|hybrid`）、`skill_preprocess_and_mapping.v1`
  - 门禁：`check-hooks-middleware-contract.*`
- A66（state/session snapshot）：
  - 字段：`runtime.state.snapshot.*`、`runtime.session.state.*`
  - 回放：`state_session_snapshot.v1`
  - 门禁：`check-state-snapshot-contract.*`
- A67（react plan notebook）：
  - 字段：`runtime.react.plan_notebook.*`、`runtime.react.plan_change_hook.*`
  - 回放：`react_plan_notebook.v1`
  - 门禁：`check-react-plan-notebook-contract.*`
- A68（realtime protocol）：
  - 字段：`runtime.realtime.protocol.*`、`runtime.realtime.interrupt_resume.*`
  - 回放：`realtime_event_protocol.v1`
  - 边界断言：`realtime_control_plane_absent`（禁止平台化实时网关/托管控制面）
  - 门禁：`check-realtime-protocol-contract.*`

跨提案联动收口（避免后续再开同域提案）：
- A58 冻结 `policy_decision_path` 与 `deny_source` 后，A60/A61 禁止重定义同义字段，仅允许引用。
- A59 冻结 memory 生命周期与检索质量阈值后，A60 预算计算必须复用该口径，不再另起成本定义。
- A60 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在 A60 内以增量任务吸收，不再新开平行提案。
- A61 的 eval 指标与 distributed 执行聚合必须复用 A58/A59/A60 的 contract 输出字段，禁止引入平行观测数据面。
- A61 distributed evaluator execution 仅允许库内嵌入式执行治理，不得演进为托管评测控制面或服务化调度平面。
- A65 不得绕过 A58 precedence 与 A57 安全治理链路；hook/middleware 输出仅走 `RuntimeRecorder` 单写入口。
- skill discovery source 同域需求（`AGENTS.md`/目录路径/混合加载、配置校验、去重顺序、回放与门禁）优先在 A65/A62 内增量吸收，不再新增平行提案。
- `Discover/Compile` 预处理接线与 `SkillBundle -> prompt/tool whitelist` 映射同域需求统一在 A65/A62 内增量吸收，不再新增平行提案。
- A66 必须复用现有 checkpoint/snapshot 语义与 A59 memory lifecycle，不得重写存储层事实源。
- A67 必须复用 A56 ReAct loop 终止 taxonomy 与 A65 hook 合同，不得新增平行 ReAct 主循环。
- A68 事件协议必须复用 A58/A67 决策与计划解释字段，不得引入第二套 interrupt/resume 语义。
- A68 realtime 合同仅定义协议与嵌入式接缝，不得新增平台化实时网关或托管连接控制面。
- A63 的命名与文档整合必须复用现有契约字段，不得改写 contract 语义；Axx->语义名映射集中维护于索引，不在多处重复定义。
- A64 的优化实现必须复用 A58-A68（若 A68 启用）既有契约字段与 reason taxonomy，禁止以性能优化引入语义分叉。
- Context Assembler 循环热路径同域需求（cache 回收、journal 批写、CA3 stage2 pass 优化、stage2 file 读取优化）统一在 A64-S1 内增量吸收，不再新增平行性能提案。
- RuntimeRecorder/diagnostics、scheduler-file/mailbox/composer recovery、MCP 调用链、skill loader、memory filesystem 引擎的同域性能需求统一在 A64-S2~S6 内增量吸收，不再新增平行性能提案。
- Runner 循环、local dispatch、provider adapter、runtime config/policy resolve 的同域性能需求统一在 A64-S7~S9 内增量吸收，不再新增平行性能提案。
- observability dispatcher/logger/exporter 事件管线的同域性能需求统一在 A64-S10 内增量吸收，不再新增平行性能提案。
- A64 所有子项必须通过 `semantic-stability + replay + perf-regression` 强门禁；任何 gate 漂移均按阻断处理，不得以“仅性能优化”为由豁免。
- A62 的示例字段与观测语义必须引用 A56-A68（若 A68 启用）既有 contract 输出，禁止在 examples 侧定义平行语义。
- 若出现新增需求，优先以 A58-A68 的“增量任务”吸收，默认不新增 A69+ 同域提案。

整合与重排说明：
- A56/A57 已从备选池转入进行中，不再计入备选编号。
- 新增紧急备选 A58（policy precedence + decision trace），用于承接 A56/A57 并行实施的跨策略冲突风险。
- 原备选 A58（memory scope）顺延为 A59，并与 builtin filesystem local-memory 增强方案合并，减少重复提案。
- 原备选 A59（runtime cost-latency budget）顺延为 A60。
- A61 合并“distributed evaluator execution”能力，不再另开平行评测执行提案。
- 新增 A65/A66/A67（hooks+middleware / unified state/session / react plan notebook），用于补齐 agent runtime 基座能力。
- 新增 A68（realtime protocol）前移至 A63/A64/A62 之前，补齐实时交互合同能力（仍按业务需求触发）。
- A63（codebase consolidation and semantic labeling）维持在 A68 之后，用于实施前收敛命名与结构。
- 原 A63（engineering/performance optimization）已顺延为 A64，保持在代码收敛提案之后，作为后置优化项。
- A62（delivery usability example pack）后移至 A64 之后，作为“功能相对完善后”的最终示例化与交付收口项。

### P2：0.x 质量与治理持续收敛

执行要求：
- 所有变更继续通过质量门禁（`check-quality-gate.*`）与契约索引追踪。
- shell 与 PowerShell 门禁 required checks 维持语义等价：native command 非零即 fail-fast；仅 `govulncheck + warn` 允许告警放行。
- 继续按“小步提案 + 契约测试 + 文档同步”推进，不引入平台化控制面范围。
- 对外发布继续以 `0.x` 说明风险与兼容预期。

### P2：Examples Backlog（从 examples TODO 收敛）

说明：
- 原 `examples/01-08/TODO.md` 已收敛到本节，避免分散维护。
- 示例运行态与使用方式以 `examples/*/README.md` 为准；增强项排期以本 roadmap 为准。
- A62 启动后，本节 backlog 统一并入“agent mode example pack”任务编排，按模式矩阵优先收口：
  - `single agent`
  - `agent with skill`
  - `react agent`
  - `multi agent`
  - `sandbox-governed agent`

当前 backlog（按示例编号）：
- `01-chat-minimal`：
  - OpenAI 实网变体（基于环境变量加载 key）。
  - 单轮延迟 benchmark 片段。
  - 进阶 prompt 状态交接示例。
- `02-tool-loop-basic`：
  - 可重试工具失败模拟（用于 backoff 调优）。
  - 背压模式对比（`reject` vs `block`）。
  - 多工具 fanout 变体与诊断输出。
- `03-mcp-mixed-call`：
  - fake MCP proxy 替换为真实 stdio/http runtime 接线。
  - reconnect/failover 演示与结构化指标。
  - async MCP 批量调用示例。
- `04-streaming-interrupt`：
  - 中断流 partial-result flush 策略。
  - 终端 UI delta 渲染。
  - cancel-storm 压测脚本。
- `05-parallel-tools-fanout`：
  - 可配置并发度与背压模式演示（`block`/`reject`）。
  - 慢工具+失败工具混合场景可视化输出。
  - 串行 vs 并行基准对比。
- `06-async-job-progress`：
  - 失败重试与 dead-letter 样例。
  - 取消传播与超时控制样例。
  - batch size 可配置与吞吐观测。
- `07-multi-agent-async-channel`：
  - 多 worker 并发与任务重分配策略。
  - 失败任务补偿与重试上限。
  - coordinator 背压与队列上限展示。
- `08-multi-agent-network-bridge`：
  - JSON-RPC batch 请求演示。
  - 标准错误码覆盖（invalid params/internal error）。
  - 客户端超时与重试策略样例。

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
