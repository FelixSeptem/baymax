# Development Roadmap

更新时间：2026-03-31

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-03-31：
- 已归档并稳定：A4-A53（完整清单以 `openspec/changes/archive/INDEX.md` 为准）。
- 进行中：
  - `introduce-observability-export-and-diagnostics-bundle-contract-a55`
  - `introduce-react-loop-and-tool-calling-parity-contract-a56`

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

### P1：A54 memory provider SPI + builtin filesystem engine（进行中，实施推进中）

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

### P1/P2：A56+ 借鉴整合提案池（全局视角）

前提约束（冻结）：
- 不调整 A54/A55 的既有范围、完成条件与验收口径；后续提案仅做增量扩展。
- 新提案必须复用既有治理主链路：`runtime/config`（`env > file > default` + fail-fast/回滚）+ `RuntimeRecorder` 单写 + diagnostics replay + quality gate。

与在研项目的先后顺序（强依赖）：
1. A54（进行中）：memory provider SPI + builtin filesystem engine。
2. A55（进行中）：observability export + diagnostics bundle contract。
3. A56（下一优先级，P1）：ReAct loop + tool-calling parity contract（Run/Stream 顺滑闭环）。
4. A57（次优先级，P1）：sandbox egress 治理与 adapter allowlist contract。
5. A58（中期，P1/P2）：memory scope 与注入预算治理 contract。
6. A59（后续，P2）：runtime 成本/时延预算与 admission contract。

备选项目说明（避免“单一路线”误解）：
- A56/A57/A58/A59 组成后续备选池，默认按上方顺序推进，但允许按风险信号前置切换，不要求机械串行实施。
- A55 已进入在研执行，不再作为备选项；A56 默认复用 A55 的观测与取证能力，减少 ReAct 上线盲区。
- 前置切换规则（示例）：
  - 若“Run 支持工具闭环但 Stream 无法工具分发/回灌”成为交付阻塞：A56 可前置为首个落地项。
  - 若合规审计或外部 adapter 引入加速：A57 可前置并优先实施。
  - 若 A54 落地后出现 memory 注入成本漂移：A58 可前置到 A57 之前。
- 无论是否前置切换，均不得改写 A54/A55 已冻结范围，只允许在其完成后做增量扩展。

备选 A56：`introduce-react-loop-and-tool-calling-parity-contract-a56`
- 目标：在保持 `library-first + contract-first` 边界下，一次性冻结 ReAct 运行闭环 contract，确保 Run/Stream 在工具调用场景语义等价并可回放可门禁。
- 范围：
  - runner 侧 ReAct 状态机收敛（`model -> tool dispatch -> tool result feedback -> next iteration`）；
  - Stream 路径工具分发与回灌能力补齐（消除 `stream_tool_dispatch_not_supported` 中间态）；
  - 统一 run-level `tool_call_limit` 与 iteration-level 上限协同；
  - provider adapter tool-calling 输入/输出归一（OpenAI/Anthropic/Gemini）；
  - ReAct additive diagnostics 字段、`react.v1` replay fixture、独立 `check-react-contract.*` gate。
- 依赖：复用 A44/A45/A49/A50/A55 的 admission/cardinality/explainability/version 与 observability 输出口径，不新增平行治理链路。
- 启动条件：主线交付需要“可稳定工具推理循环 + Stream 等价”能力，或出现 provider tool-calling 行为漂移。

备选 A57：`introduce-sandbox-egress-governance-and-adapter-allowlist-contract-a57`
- 目标：补齐 sandbox 网络外呼治理（egress policy）与 adapter 供应链 allowlist 契约，形成“执行隔离 + 出口治理 + 激活准入”闭环。
- 范围：`security.sandbox.egress.*`、`adapter.allowlist.*`、readiness/admission finding、taxonomy、replay drift 与 conformance matrix。
- 依赖：复用 A51/A52/A53 sandbox taxonomy 与 adapter manifest 激活边界，不新增平行安全语义。
- 启动条件：存在合规审计或外部 adapter 引入规模上升，需要可审计可阻断的 egress/allowlist 治理。

备选 A58：`introduce-memory-scope-and-injection-budget-governance-contract-a58`
- 目标：在 A54 SPI 基线之上补齐 `session|project|global` scope、注入预算与检索策略治理，抑制上下文膨胀与成本漂移。
- 范围：`runtime.memory.scope.*`、`runtime.memory.injection_budget.*`、QueryRuns additive 字段、mixed replay fixture、gate 阈值回归。
- 依赖：A54 memory facade/profile pack/readiness 字段稳定后再扩展，避免与 A54 实施交叉改动。
- 启动条件：A54 上线后出现上下文成本抖动、memory 注入不可解释或跨 provider 行为漂移。

备选 A59：`introduce-runtime-cost-latency-budget-and-admission-contract-a59`
- 目标：统一 token/tool/sandbox/memory 成本与时延预算，建立 admission 侧 fail-fast 与降级策略。
- 启动条件：成本或 P95 抖动成为主线瓶颈。

整合与重排说明：
- 原备选 `A57 incident-forensics-replay-package` 已并入 A55（diagnostics bundle + replay hint 一体化）。
- 新增 `A56 react-loop-and-tool-calling-parity` 作为“顺滑 ReAct 模式”专项提案，并置于备选池首位。
- 原备选 `A55 cross-channel-data-egress-and-secret-governance` 顺延并聚焦为 A57（sandbox egress + adapter allowlist 闭环）。
- 原备选 `A56 runtime-cost-latency-budget` 顺延为 A59，避免与 A56/A57/A58 的阶段目标重叠。

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
