# Development Roadmap

更新时间：2026-04-09

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-04-09：
- 已归档并稳定：早期与主线归档提案（完整清单以 `openspec/changes/archive/INDEX.md` 为准）。
- 已归档：
  - `introduce-governance-automation-and-consistency-gate-contract-a70`
  - `introduce-context-compression-production-hardening-contract-a69`
  - `introduce-delivery-usability-agent-mode-example-pack-contract-a62`
  - `introduce-real-runtime-agent-mode-examples-contract-a71`（real runtime agent mode examples）
- 进行中：
  - `introduce-agent-mode-anti-template-doc-first-delivery-contract-a72`
- 候选：以 openspec list --json 为准（当前无独立候选快照条目）。

## 版本阶段口径（延续 0.x）

当前仓库**不做 `1.0.0` / prod-ready 承诺**，继续沿用 `0.x` 治理口径（见 `docs/versioning-and-compatibility.md`）。
在 `0.x` 阶段，版本号用于表达变更范围，不构成稳定兼容承诺；主线目标是“持续收敛、可回归迭代”。
`0.x` 阶段**允许新增能力型提案**，不采用“仅治理/仅修复”的限制；新增能力需满足准入字段与质量门禁要求。

1. 运行时主干稳定：
- Runner Run/Stream 统一语义与并发背压基线。
- Multi-provider（OpenAI/Anthropic/Gemini）统一 contract。
- Context Assembler CA1-CA4、Security S1-S4 已归档能力。

2. 多代理主链路稳定：
- multi-agent baseline contracts (sync/async/delayed/recovery/collab/unified query)（同步/异步/延后、恢复边界、协作原语、统一诊断查询）语义收口。
- Shared contract gate 与 Run/Stream 等价约束保持阻断。

3. 质量与可回归稳定：
- performance regression baseline gate 性能回归门禁（基线 + 相对阈值）。
- diagnostics query performance baseline + regression gate diagnostics query 性能回归门禁（`BenchmarkDiagnosticsQueryRuns|QueryMailbox|MailboxAggregates`，默认阈值 `12/15/12%`，已归档）。
- full-chain example smoke blocking gate 全链路示例 smoke 阻断门禁。

4. 外部接入稳定：
- adapter template + migration mapping contract 模板与迁移映射（已归档）。
- conformance harness contract conformance harness（已归档）。
- scaffold + conformance bootstrap contract scaffold + conformance bootstrap（已归档）。

## 近期收口优先级（0.x）

### P0：governance automation and consistency gate（A70，已归档）

治理自动化与一致性门禁（A70）目标：
- 固化 `openspec list --json`、`openspec/changes/archive/INDEX.md`、`docs/development-roadmap.md` 的状态一致性检查，阻断 roadmap 状态漂移。
- 固化后续提案 `Example Impact Assessment` 声明校验，输出稳定分类码：
  - `missing-example-impact-declaration`
  - `invalid-example-impact-value`
- 在 docs/quality 双门禁接线并保持 shell/PowerShell parity：
  - `scripts/check-openspec-roadmap-status-consistency.sh/.ps1`
  - `scripts/check-openspec-example-impact-declaration.sh/.ps1`
  - `scripts/check-docs-consistency.sh/.ps1`
  - `scripts/check-quality-gate.sh/.ps1`

A70 边界（不做）：
- 不修改 runtime/context/model/mcp 行为语义与公开 API。
- 不引入外部治理服务或平台化控制面。
- 不替代各能力提案原有 contract/replay/perf 验证职责。

A70 DoD：
- 新增治理脚本接线完成并在门禁中阻断漂移。
- CI 暴露 required-check 候选：
  - `.github/workflows/ci.yml::openspec-roadmap-status-consistency-gate`
  - `.github/workflows/ci.yml::openspec-example-impact-declaration-gate`
- 文档索引与协作规范同步到位（roadmap/mainline index/AGENTS）。

### P0：async-await poll reconcile fallback contract 收口（已归档）

async-await poll reconcile fallback contract 依赖关系：
- async-await lifecycle baseline contract 已提供 `awaiting_report + timeout + late_report_policy` 生命周期基线；
- async-await poll reconcile fallback contract 在此基础上补齐 callback 之外的 poll reconcile fallback 契约。

完成条件（async-await poll reconcile fallback contract）：
- 为 `awaiting_report` 任务增加可配置 reconcile poll fallback：`interval/batch_size/jitter_ratio`。
- 终态仲裁固定为 `first_terminal_wins + record_conflict`，后到冲突事件不覆写业务终态。
- `not_found_policy=keep_until_timeout`：poll `not_found` 不直接终态，保持等待至 `report_timeout`。
- 在 async accepted 路径持久化远端关联键（`remote_task_id`）并跨 snapshot/recovery 保持可对账。
- Task Board 查询扩展 async additive 观测字段：`resolution_source`、`remote_task_id`、`terminal_conflict_recorded`。
- `runtime/config` 新增 `scheduler.async_await.reconcile.*`（默认关闭）并纳入 fail-fast + 热更新回滚。
- `runtime/diagnostics` 增加 reconcile additive 字段并保持 `additive + nullable + default` 兼容窗口。
- shared multi-agent gate 纳入 async-await reconcile suites（callback-loss fallback、冲突仲裁、Run/Stream 等价、memory/file parity、replay idempotency）。

当前阶段非目标（async-await poll reconcile fallback contract 不做）：
- 引入外部 MQ（Kafka/NATS/RabbitMQ）适配。
- 提供平台化消息控制面（UI/RBAC/多租户运维面板）。
- 承诺 exactly-once 语义。

### P0：mailbox canonical entry consolidation 收口（已归档）

mailbox canonical entry consolidation 依赖关系：
- mailbox unified coordination contract 已确立 mailbox 统一协调主契约。
- collaboration primitive retry contract 已归档，协作原语重试语义可作为稳定基线。

完成条件（mailbox canonical entry consolidation）：
- 退场 legacy direct invoke 公共入口（`InvokeSync` / `InvokeAsync`）并固定 mailbox 为 canonical 调用面。
- `MailboxBridge` 内部不再依赖 deprecated direct invoke 导出路径。
- shared multi-agent gate 与 quality gate 增加 canonical-only 阻断，防止 legacy 入口回流。
- README / roadmap / mainline index / orchestration 模块文档移除“deprecated 但仍主路径依赖”的中间态描述。

当前阶段非目标（mailbox canonical entry consolidation 不做）：
- 不引入平台化控制面或外部消息总线。
- 不改 async-await poll reconcile fallback contract async-await 收敛仲裁语义。

### P1：mailbox runtime wiring contract 接线（已归档）

mailbox runtime wiring contract 依赖关系：
- mailbox canonical entry consolidation 收口 canonical 调用入口后，进一步把 mailbox 配置与运行时主链路接线闭环。

完成条件（mailbox runtime wiring contract）：
- managed 编排路径接入共享 mailbox runtime wiring，避免 per-call `NewInMemoryMailboxBridge()` 中间态。
- `mailbox.enabled=false` 时使用共享 memory mailbox；`mailbox.enabled=true` 按 resolved backend 初始化。
- `mailbox.backend=file` 初始化失败回退到 memory，并记录 deterministic fallback reason。
- mailbox publish 主路径接入 diagnostics 写入，使 `QueryMailbox` / `MailboxAggregates` 反映真实主链路数据。
- shared multi-agent gate 纳入 mailbox runtime wiring 套件（配置接线、fallback、Run/Stream 等价、memory/file parity）。

当前阶段非目标（mailbox runtime wiring contract 不做）：
- 不引入 MQ 平台化能力或控制平面。
- 不替代 mailbox canonical entry consolidation 的 API 收口目标。

### P1：mailbox lifecycle worker + observability contract lifecycle worker 与可观测性（已归档）

mailbox lifecycle worker + observability contract 依赖关系：
- mailbox runtime wiring contract 已完成 mailbox runtime wiring 与 publish 诊断闭环；
- mailbox lifecycle worker + observability contract 在此基础上补齐 mailbox lifecycle worker 原语与 reason taxonomy 治理。

完成条件（mailbox lifecycle worker + observability contract）：
- 新增库级 mailbox worker 原语（默认关闭）：`consume -> handler -> ack|nack|requeue`。
- 固化 worker 默认值：`enabled=false`、`poll_interval=100ms`、`handler_error_policy=requeue`。
- `runtime/config` 增加 `mailbox.worker.*` 配置域并纳入启动/热更新 fail-fast + 原子回滚。
- mailbox lifecycle diagnostics 覆盖 `consume/ack/nack/requeue/dead_letter/expired`。
- lifecycle reason taxonomy 冻结为 canonical 集合：
  `retry_exhausted`、`expired`、`consumer_mismatch`、`message_not_found`、`handler_error`。
- shared multi-agent gate 纳入 worker lifecycle 套件（enabled/disabled、Run/Stream 等价、memory/file parity、taxonomy drift guard）。

当前阶段非目标（mailbox lifecycle worker + observability contract 不做）：
- 不引入外部 MQ、平台化控制面或托管任务面板。
- 不改变 async-await poll reconcile fallback contract async-await 终态仲裁语义。

### P1：task board control + manual recovery contract task board control + manual recovery（已归档）

task board control + manual recovery contract 依赖关系：
- task board query read-only contract 已交付 Task Board query 只读契约；
- task board control + manual recovery contract 在保持 query 只读语义不变的前提下，补齐库级 control 路径与手工恢复契约。

完成条件（task board control + manual recovery contract）：
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

当前阶段非目标（task board control + manual recovery contract 不做）：
- 不引入平台化任务控制面（RBAC/UI/多租户运维）。
- 不改变既有 enqueue/claim/heartbeat/requeue/commit 与 query 只读路径语义。

### P1：runtime readiness preflight + degradation contract runtime readiness preflight + degradation contract（已归档）

runtime readiness preflight + degradation contract 依赖关系：
- mailbox runtime wiring contract/mailbox lifecycle worker + observability contract 已将 scheduler/mailbox/recovery fallback 状态统一回流到 runtime 诊断路径；
- runtime readiness preflight + degradation contract 在保持 lib-first 边界下新增启动前 readiness 预检契约，不改变既有 Run/Stream 终态裁决。

完成条件（runtime readiness preflight + degradation contract）：
- `runtime/config.Manager` 提供库级 `ReadinessPreflight()`，输出 `ready|degraded|blocked` 与 canonical findings（`code/domain/severity/message/metadata`）。
- 新增 `runtime.readiness.*` 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 预检覆盖本地配置有效性与 scheduler/mailbox/recovery backend/fallback 可见性。
- `strict=true` 时把 `degraded` 升级为 `blocked`，`strict=false` 保持可运行且可观测。
- run diagnostics 增量字段落地：`runtime_readiness_status`、计数字段、`runtime_readiness_primary_code`。
- composer 暴露 runtime readiness 透传入口，且查询路径保持只读，不引入新状态 taxonomy。
- quality gate 接入 readiness suites（classification、strict escalation、schema stability、diagnostics replay idempotency、composer parity）。

当前阶段非目标（runtime readiness preflight + degradation contract 不做）：
- 不引入平台化控制面/远程运维探针系统。
- 不改变 scheduler/task lifecycle 语义，不引入额外终态。

### P1：operation profile + timeout resolution contract operation profile + timeout resolution contract（已归档）

operation profile + timeout resolution contract 依赖关系：
- runtime readiness preflight + degradation contract readiness 契约已归档，运行时配置与诊断路径具备稳定扩展点；
- operation profile + timeout resolution contract 在既有 scheduler/composer 多代理主链路上，补齐跨域 timeout 解析与父子预算收敛。

完成条件（operation profile + timeout resolution contract）：
- `runtime.operation_profiles.*` 配置域落地，并保持 `env > file > default` 与 fail-fast/回滚语义。
- 共享 timeout resolver 固化 `profile -> domain -> request` 优先级，并输出来源标签与可追踪 trace。
- scheduler/composer 子任务路径统一接入 resolver；父子预算收敛固定为 `min(parent_remaining, child_resolved)`。
- timeout-resolution 元数据在 snapshot/recovery/replay 下保持稳定，且 replay 不膨胀逻辑聚合。
- diagnostics 与 QueryRuns/Task Board 补齐 operation profile + timeout resolution contract additive 字段，并保持 `additive + nullable + default` 兼容语义。
- shared contract gate 与 quality gate 纳入 operation profile + timeout resolution contract 阻断套件（校验/优先级/夹紧与拒绝/Run-Stream 等价/memory-file parity/replay idempotency）。

当前阶段非目标（operation profile + timeout resolution contract 不做）：
- 不引入平台化控制面与外部 MQ 依赖。
- 不改变既有 async-await/recovery 终态仲裁契约。

### P1：diagnostics query performance baseline + regression gate diagnostics query performance baseline + regression gate（已归档）

diagnostics query performance baseline + regression gate 目标：
- 为 unified diagnostics query 建立可复现实验基线（延迟、分页、聚合开销）。
- 新增独立 gate 脚本：`scripts/check-diagnostics-query-performance-regression.sh` 与 `scripts/check-diagnostics-query-performance-regression.ps1`。
- 固化默认执行参数：`benchtime=200ms`、`count=5`。
- 在质量门禁接入回归阈值校验（默认：`ns/op 12%`、`p95-ns/op 15%`、`allocs/op 12%`），防止查询路径性能漂移。

### P1：adapter runtime health probe + readiness integration contract adapter runtime health probe + readiness integration（已归档）

adapter runtime health probe + readiness integration contract 目标：
- 新增 `adapter/health` 运行期探测契约，固化 `healthy|degraded|unavailable` 三态与 canonical reason taxonomy。
- 新增 `adapter.health.*` 配置域（`enabled/strict/probe_timeout/cache_ttl`），并纳入 `env > file > default`、启动 fail-fast、热更新回滚。
- 将 adapter health 接入 `ReadinessPreflight()`：
  - required unavailable 在 strict 语义下阻断；
  - optional unavailable 在 non-strict 路径降级并保持可观测。
- 在 diagnostics 增加 adapter-health additive 字段（`status/probe_total/degraded_total/unavailable_total/primary_code`），保证 replay idempotency。
- 在 `integration/adapterconformance` 增加 adapter-health matrix，并接入 `check-adapter-conformance.*` 与 `check-quality-gate.*` 阻断步骤（shell/PowerShell parity）。

### P1：readiness admission guard + degradation policy contract readiness admission guard + degradation policy（已归档）

readiness admission guard + degradation policy contract 目标：
- 在 managed Run/Stream 入口引入统一 readiness admission guard，形成执行前准入护栏。
- 新增 `runtime.readiness.admission.*` 配置域并保持 `env > file > default`、启动 fail-fast、热更新回滚语义。
- 固化 `blocked` 拒绝执行与 `degraded` 策略化处理（allow_and_record / fail_fast）规则。
- 增加 admission additive 诊断字段并纳入 replay idempotency 契约。
- 将 admission suites 纳入 quality gate 阻断路径并保持 shell/PowerShell parity。

### P1：diagnostics cardinality budget + truncation governance contract diagnostics cardinality budget + truncation governance（已归档）

diagnostics cardinality budget + truncation governance contract 目标：
- 为新增 additive 字段建立高基数预算与截断治理，避免查询成本漂移。
- 固化 map/list/string 字段的 bounded-cardinality 与稳定序列化语义。
- 新增 `diagnostics.cardinality.*` 配置域，默认 `overflow_policy=truncate_and_record`，并支持 `fail_fast`。
- 将 cardinality drift 检查纳入质量门禁与回放契约验证。

### P1：adapter health backoff + circuit governance contract adapter health backoff + circuit governance（已归档）

adapter health backoff + circuit governance contract 目标：
- 在 adapter runtime health probe + readiness integration contract 健康探测语义上增加指数退避 + 抖动 + 半开探测治理。
- 防止外部 adapter 不可用时的探测风暴与瞬时抖动放大。
- 通过 conformance + quality gate 固化故障恢复和抖动抑制语义。

adapter health backoff + circuit governance contract 当前落地（实现已完成）：
- `runtime/config` 新增 `adapter.health.backoff.*` 与 `adapter.health.circuit.*`（default/env/file、startup 校验、hot reload 非法更新回滚）。
- `adapter/health` 落地 `closed|open|half_open` 状态机、指数退避 + 抖动、half-open 探测预算与恢复判定。
- `runtime/config/readiness` 增加 circuit-open / half-open degraded / governance recovered 的 canonical `adapter.health.*` finding 映射，并保持 strict/non-strict 分类稳定。
- `runtime/diagnostics` 与 `RuntimeRecorder` 新增 adapter health backoff + circuit governance contract additive 字段：`adapter_health_backoff_applied_total`、`adapter_health_circuit_*`、`adapter_health_governance_primary_code`。
- `integration/adapterconformance` 新增 governance matrix suites（状态转移确定性、半开恢复、taxonomy drift guard、replay idempotency）。
- `scripts/check-adapter-conformance.*` 与 `scripts/check-quality-gate.*` 已纳入 adapter health backoff + circuit governance contract suites 并保持 shell/PowerShell parity。

### P1：readiness-timeout-health replay fixture gate readiness-timeout-health replay fixture gate（已归档）

readiness-timeout-health replay fixture gate 目标：
- 固化 `readiness + timeout resolution + adapter health` 交叉语义回放夹具。
- 防止跨提案演进造成 finding taxonomy 与阻断策略漂移。
- 为后续 0.x 收敛阶段提供稳定的语义回归基线。

readiness-timeout-health replay fixture gate 当前落地（实现已完成）：
- `tool/diagnosticsreplay` 新增 readiness-timeout-health replay fixture gate 组合 fixture schema（`version=readiness-timeout-health replay fixture gate.v1`）、loader、canonical normalization 与 deterministic assertion pipeline。
- 错误分类补齐 `schema_mismatch|semantic_drift|ordering_drift`，并对 taxonomy/source/state 漂移执行 fail-fast。
- 新增 `integration/readiness_timeout_health_replay_contract_test.go` 与 `integration/testdata/diagnostics-replay/readiness-timeout-health replay fixture gate/v1/*`（success + taxonomy/source/state drift fixtures）。
- quality gate 接入 readiness-timeout-health replay fixture gate 阻断步骤：`go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReadinessTimeoutHealthReplayContract)' -count=1`，shell/PowerShell parity 保持一致。
- 主干索引与 diagnostics 文档已补齐 readiness-timeout-health replay fixture gate fixture suite 与 gate 映射。

### P1：cross-domain primary reason arbitration contract cross-domain primary reason arbitration（已归档）

cross-domain primary reason arbitration contract 目标：
- 固化 timeout/readiness/adapter-health 冲突场景下的 primary reason 裁决优先级与 tie-break 规则。
- 统一 `runtime_primary_domain|code|source` 解释链路，保持 Run/Stream/replay 语义一致。
- 将 arbitration drift 检测纳入 replay + quality gate 阻断，防止跨提案演进产生 reclassification drift。

cross-domain primary reason arbitration contract 当前落地（已归档）：
- `runtime/config` 新增 cross-domain arbitration helper，固定 precedence（timeout reject/exhausted > readiness blocked > adapter required unavailable > degraded/optional > warning/info）并支持 lexical tie-break 与 conflict_total。
- `runtime/config/readiness` 与 admission guard 统一消费 arbitration 输出，解释字段对齐 `primary domain/code/source`，Run/Stream 保持语义等价。
- `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 增加 cross-domain primary reason arbitration contract additive 字段：`runtime_primary_domain`、`runtime_primary_code`、`runtime_primary_source`、`runtime_primary_conflict_total`，并保持 replay idempotency。
- `tool/diagnosticsreplay` 新增 cross-domain primary reason arbitration contract fixture schema（`version=cross-domain primary reason arbitration contract.v1`）与 drift 分类：`precedence_drift`、`tie_break_drift`、`taxonomy_drift`。
- 新增 `integration/primary_reason_arbitration_replay_contract_test.go` 与 `integration/testdata/diagnostics-replay/cross-domain primary reason arbitration contract/v1/*`，覆盖 replay parity + drift guard。
- quality gate 阻断步骤扩展为：`go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReplayContractPrimaryReasonArbitrationFixture|ReadinessTimeoutHealthReplayContract|PrimaryReasonArbitrationReplayContract)' -count=1`（shell/PowerShell parity 保持一致）。

### P1：arbitration explainability + secondary reason contract arbitration explainability + secondary reason（已归档）

arbitration explainability + secondary reason contract 目标：
- 固化 secondary reasons 的有界输出契约（上限、去重、稳定排序）并输出 rule version。
- 统一 remediation hint taxonomy，补齐 machine-readable explainability 字段。
- 将 explainability drift（secondary order/count、hint taxonomy、rule version）纳入 replay + quality gate 阻断。

arbitration explainability + secondary reason contract 当前落地（已归档）：
- `runtime/config` 已扩展 arbitration explainability 输出：`runtime_secondary_reason_codes`、`runtime_secondary_reason_count`、`runtime_arbitration_rule_version`、`runtime_remediation_hint_code`、`runtime_remediation_hint_domain`，并固定 `max_secondary_reasons=3`。
- `runtime/config/readiness` 与 admission guard 已贯通 explainability 字段透传（primary + secondary + hint + rule version），deny details 保持 machine-readable 字段对齐。
- `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 已接入 arbitration explainability + secondary reason contract additive 字段并补齐 replay idempotency 断言。
- `tool/diagnosticsreplay` arbitration fixture 已升级支持 `version=arbitration explainability + secondary reason contract.v1`，新增 drift 分类：`secondary_order_drift`、`secondary_count_drift`、`hint_taxonomy_drift`、`rule_version_drift`。
- quality gate readiness 套件已纳入 arbitration explainability + secondary reason contract parser-compatibility 回归（shell/PowerShell parity）。

### P1：arbitration version governance + compatibility contract arbitration version governance + compatibility（已归档）

arbitration version governance + compatibility contract 目标：
- 固化 arbitration rule version 解析与 compatibility window 契约（requested/default/effective/source）。
- 统一 unsupported/mismatch 策略（默认 fail-fast），并贯通 readiness preflight 与 admission guard。
- 将 cross-version drift（`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`）纳入 replay + quality gate 阻断。

arbitration version governance + compatibility contract 当前落地（已归档）：
- `runtime/config` 已新增 `runtime.arbitration.version.*` 配置域（`enabled/default/compat_window/on_unsupported/on_mismatch`），并接入 `env > file > default`、启动 fail-fast 校验、热更新非法回滚。
- cross-domain arbitration/readiness/admission 已接入 version resolver，unsupported/mismatch 在 fail-fast 策略下保持 deterministic deny 与 explainability 透传（requested/effective/source/policy/counters）。
- `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 已接入 arbitration version governance + compatibility contract additive 字段：`runtime_arbitration_rule_requested_version`、`runtime_arbitration_rule_effective_version`、`runtime_arbitration_rule_version_source`、`runtime_arbitration_rule_policy_action`、`runtime_arbitration_rule_unsupported_total`、`runtime_arbitration_rule_mismatch_total`。
- `tool/diagnosticsreplay` arbitration fixture 已升级支持 `version=arbitration version governance + compatibility contract.v1`，并新增 drift 分类：`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`，同时保持 `cross-domain primary reason arbitration contract/arbitration explainability + secondary reason contract` 向后兼容。
- 新增 arbitration version governance + compatibility contract integration suites（Run/Stream parity、memory/file parity、replay parity），并已纳入 `check-quality-gate.sh/.ps1` 阻断步骤。

### P1：sandbox execution isolation contract sandbox execution isolation contract（已归档）

sandbox execution isolation contract Why now：
- 当前 S2-S4 已覆盖权限/限流/IO 过滤与 deny 告警投递，但本地工具执行仍以 in-process 为主，缺少“执行隔离”契约层。
- 对高风险工具（如 shell/file-system/process 访问）仅靠策略 deny/confirm 不足以满足更高隔离要求，需要可审计的 sandbox 运行面。
- 在保持 lib-first 边界前提下，需要提供“宿主可注入隔离执行器 + 运行时统一治理/诊断”的标准接缝，避免业务侧散装实现。

sandbox execution isolation contract 依赖关系：
- 复用 S2/S3/S4 既有 taxonomy 与事件投递治理，不新增平行安全事件体系。
- 复用 readiness admission guard + degradation policy contract readiness admission；当 `sandbox.required=true` 且执行器不可用时，准入层可 fail-fast 阻断。
- 复用 diagnostics cardinality budget + truncation governance contract additive/cardinality 治理，保证 sandbox 诊断字段新增不破坏查询性能与兼容窗口。
- 复用 arbitration explainability + secondary reason contract/arbitration version governance + compatibility contract explainability 与 rule-version 口径，确保 sandbox deny 在 Run/Stream/replay 下可解释且稳定。

sandbox execution isolation contract 完成条件（提案落地后）：
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
  - deny 路径保持 side-effect free（不触发调度/发布副作用），与 readiness admission guard + degradation policy contract admission 语义一致。
- 回放与门禁：
  - diagnostics replay 增加 sandbox fixture（建议 `sandbox execution isolation contract.v1`）与 drift 分类（taxonomy/order/idempotency）。
  - quality gate 新增 `check-security-sandbox-contract.sh/.ps1` 并纳入 `check-quality-gate.*`。
  - 增加 offline deterministic `sandbox executor conformance harness`（`check-sandbox-executor-conformance.sh/.ps1`）并接入 sandbox gate。
  - CI 暴露独立 required-check 候选 `security-sandbox-gate`。

sandbox execution isolation contract 当前落地（已归档）：
- `integration/sandbox_execution_isolation_contract_test.go` 已覆盖 Run/Stream parity、capability negotiation deny、backend compatibility matrix smoke（Linux + Windows job）。
- `integration/sandboxconformance` 已落地 offline deterministic conformance harness（canonical ExecSpec/ExecResult、capability negotiation drift、session lifecycle、fallback 语义）。
- `scripts/check-security-sandbox-contract.sh/.ps1` 已接入 conformance harness，并由 `scripts/check-quality-gate.sh/.ps1` 阻断执行。
- `.github/workflows/ci.yml` 已新增独立 job `security-sandbox-gate`（PR 触发）作为 required-check 候选。

sandbox execution isolation contract 当前阶段非目标（不做）：
- 不内置 Docker/Kubernetes/VM 控制面，不引入平台化多租户治理能力。
- 不承诺跨主机/跨内核强隔离（隔离强度由宿主注入执行器能力决定）。
- 不改变 provider fallback、A2A/workflow/scheduler 既有主链路语义。

sandbox execution isolation contract 风险与回滚点：
- 主要风险：策略误配导致误拒绝、sandbox 启动开销导致时延抖动、跨平台执行器行为漂移。
- 缓解策略：先 `mode=observe` 灰度，稳定后切换 `mode=enforce`；高风险工具先小范围 `by_tool` 启用。
- 回滚点：`security.sandbox.enabled=false` 或 `mode=observe`；非法热更新一律回滚到上一有效快照。

sandbox execution isolation contract 验证命令（提案实施期最小集合）：
- `go test ./tool/local ./core/runner ./mcp/stdio -count=1`
- `go test ./integration -run '^TestSandboxExecutionIsolationContract' -count=1`
- `go test ./integration/sandboxconformance -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-sandbox-executor-conformance.ps1`
- `pwsh -File scripts/check-security-sandbox-contract.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

### P1：sandbox runtime rollout + health/capacity governance contract sandbox runtime rollout + health/capacity governance（已归档）

sandbox runtime rollout + health/capacity governance contract Why now：
- sandbox execution isolation contract 已冻结 sandbox 接入与隔离语义，但“如何安全放量上线”仍缺统一 contract，当前容易落回业务侧脚本治理。
- rollout/freeze/capacity 若不统一到 readiness/admission/diagnostics/replay，将导致 Run/Stream 语义漂移与回滚不可验证。
- 需要把 sandbox 从“可用”提升到“可持续上线”，并保持主流后端接入路径在统一治理面下可替换。

sandbox runtime rollout + health/capacity governance contract 依赖关系：
- 复用 sandbox execution isolation contract 的 sandbox execution isolation contract，不重新定义 ExecSpec/ExecResult 与 capability negotiation 基线。
- 复用 readiness admission guard + degradation policy contract readiness/admission fail-fast 与 deny side-effect-free 语义，作为 rollout/capacity 判定执行前置入口。
- 复用 diagnostics query performance baseline + regression gate/diagnostics cardinality budget + truncation governance contract 的 diagnostics query/perf/cardinality 治理，确保 rollout 新字段不破坏查询与兼容窗口。
- 复用 arbitration explainability + secondary reason contract/arbitration version governance + compatibility contract 的 explainability 与 version-governance 输出口径，保证冻结/节流动作可解释可回放。

sandbox runtime rollout + health/capacity governance contract 完成条件（提案落地后）：
- 新增 `security.sandbox.rollout.*` 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚：
  - phase 状态机：`observe|canary|baseline|full|frozen`（含合法迁移约束）。
  - 健康预算：启动失败率、超时率、违规率、P95 时延漂移、准入拒绝率。
  - 容量预算：`max_inflight`、`max_queue`、`throttle_threshold`、`deny_threshold`、`degraded_policy`。
  - 冻结治理：`freeze_on_breach`、`cooldown`、`manual_unfreeze_token`。
- readiness preflight + admission guard 接入 rollout/freeze/capacity canonical findings 与 deterministic 准入动作（`allow|throttle|deny`）。
- timeline/diagnostics/replay 一体化收敛：
  - timeline 新增 `sandbox.rollout.*` canonical reasons。
  - diagnostics 新增 rollout/capacity/freeze additive 字段并保持 single-writer idempotency。
  - replay 新增 `sandbox runtime rollout + health/capacity governance contract.v1` fixture 与 drift 分类（phase/health/capacity/freeze）。
- quality gate 收口：
  - 新增 `check-sandbox-rollout-governance-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
  - 作为独立 required-check 候选暴露，保持 shell/PowerShell parity。

sandbox runtime rollout + health/capacity governance contract 当前阶段非目标（不做）：
- 不引入平台化控制面（多租户运维面板、跨租户调度中心）。
- 不改变 sandbox execution isolation contract sandbox 执行 contract（ExecSpec/ExecResult/capability）。
- 不引入跨主机全局容量编排，仅定义单 runtime contract。

sandbox runtime rollout + health/capacity governance contract 风险与回滚点：
- 主要风险：预算阈值过紧导致误冻结、峰值期节流策略造成拒绝率抬升、后端抖动导致频繁冻结。
- 缓解策略：默认 `phase=observe`，先 `canary` 小流量；对冻结引入 cooldown + token 解冻；保留 `allow_and_record` 过渡策略。
- 回滚点：将 phase 回退到 `observe`，或暂时禁用 `freeze_on_breach`；非法热更新一律回滚到上一有效快照。

sandbox runtime rollout + health/capacity governance contract 验证命令（提案实施期最小集合）：
- `go test ./runtime/config ./runtime/config/readiness ./core/runner -count=1`
- `go test ./integration -run 'TestSandboxRollout|TestSandboxCapacityAdmission|TestRunStreamSandboxRolloutParity' -count=1`
- `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractSandboxA52Fixture' -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

### P1：mainstream sandbox adapter conformance + migration pack contract mainstream sandbox adapter conformance + migration pack（已归档）

mainstream sandbox adapter conformance + migration pack contract Why now：
- sandbox runtime rollout + health/capacity governance contract 已归档并冻结 sandbox rollout/health/capacity 治理基线，但主流后端（nsjail/bwrap/OCI/windows-job）接入仍依赖分散脚本与非标准 glue code。
- 若不统一 adapter manifest + conformance + migration mapping，后端切换成本高，且语义漂移很难被 gate 前置阻断。
- 需要在 sandbox runtime rollout + health/capacity governance contract 后续阶段收敛接入 contract，避免后续重复提出“同类 sandbox 接入治理”提案。

mainstream sandbox adapter conformance + migration pack contract 依赖关系：
- 复用 sandbox execution isolation contract 的 sandbox 执行隔离语义与 canonical backend/capability taxonomy，不重定义执行 contract。
- 复用 sandbox runtime rollout + health/capacity governance contract rollout/health/capacity 治理语义，仅关注“外部接入 DX + conformance + migration”层。
- 复用 adapter template + migration mapping contract/conformance harness contract/manifest template contract/profile versioning + replay contract 的 adapter template/conformance/manifest/profile-replay 治理链路，做 sandbox 维度扩展。

mainstream sandbox adapter conformance + migration pack contract 完成条件（提案落地后）：
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

mainstream sandbox adapter conformance + migration pack contract 当前阶段非目标（不做）：
- 不改 sandbox execution isolation contract/sandbox runtime rollout + health/capacity governance contract 的 sandbox 执行与运行治理语义。
- 不引入平台化控制面或跨租户编排能力。
- 不承诺后端底层实现一致，仅要求 canonical 合同输出一致。

mainstream sandbox adapter conformance + migration pack contract 已落地增量（归档记录）：
- `adapter/manifest` 已补齐 sandbox metadata/profile-pack 契约（`sandbox_backend`、`sandbox_profile_id`、`host_os`、`host_arch`、`session_modes_supported`）与 fail-fast 校验。
- `integration/adapterconformance` 已增加 mainstream backend matrix、capability negotiation、session lifecycle（`per_call|per_session`、crash/reconnect、close idempotent）与 canonical drift class 断言。
- `integration/adaptercontractreplay` 已增加 `sandbox.v1` 回放轨道与 mixed-track 回放，补齐 drift 分类断言（`sandbox_backend_profile_drift`、`sandbox_manifest_compat_drift`、`sandbox_session_mode_drift`）。
- `runtime/config/readiness` 已增加 `sandbox.adapter.*` finding（`profile_missing`、`backend_not_supported`、`host_mismatch`、`session_mode_unsupported`）与 strict/non-strict 测试映射。
- 已新增 `scripts/check-sandbox-adapter-conformance-contract.sh/.ps1` 并接入 `check-quality-gate.*`，CI 暴露独立 job `sandbox-adapter-conformance-gate`。

mainstream sandbox adapter conformance + migration pack contract 风险与回滚点：
- 主要风险：profile pack 过重导致接入门槛上升、不同 runner 的 backend 可用性差异引发误报、模板与实现漂移。
- 缓解策略：最小必填 schema、平台条件化 matrix + skip reason 审计、模板绑定 conformance case 持续校验。
- 回滚点：暂时下线新增 sandbox adapter gate required-check；保留现有 adapter conformance 主路径与旧模板文档。

mainstream sandbox adapter conformance + migration pack contract 验证命令（提案实施期最小集合）：
- `go test ./adapter/... ./tool/... -count=1`
- `go test ./integration -run 'TestSandboxAdapterConformance|TestSandboxAdapterManifestCompatibility|TestSandboxAdapterProfileReplay' -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

### P1：memory provider SPI + builtin filesystem engine contract memory provider SPI + builtin filesystem engine（已归档）

memory provider SPI + builtin filesystem engine contract Why now：
- 当前 memory 接入仍依赖 CA2 file/external retriever 分散路径，缺少统一 memory SPI 与 profile 契约。
- 主流 memory 框架（`mem0|zep|openviking`）接入成本高，且 provider-specific 分支容易渗透主流程并造成语义漂移。
- 需要一次性冻结 memory 的 config/readiness/diagnostics/replay/conformance/gate 契约，避免后续在 memory 主题上重复拆提案。

memory provider SPI + builtin filesystem engine contract 依赖关系：
- 复用既有 runtime config 热更新治理（`env > file > default`、fail-fast、原子回滚）与 RuntimeRecorder single-writer 约束。
- 复用 adapter template + migration mapping contract/conformance harness contract/manifest template contract/profile versioning + replay contract 的 template/conformance/manifest/profile-replay 治理链路，扩展到 memory 维度。
- 复用 diagnostics query performance baseline + regression gate/diagnostics cardinality budget + truncation governance contract 的 diagnostics query/perf/cardinality 治理边界，确保 memory additive 字段不破坏查询与兼容窗口。
- 复用 readiness admission guard + degradation policy contract readiness strict/non-strict 映射语义，新增 `memory.*` findings 而不引入平行判定体系。

memory provider SPI + builtin filesystem engine contract 完成条件（提案落地后）：
- 新增 `runtime-memory-engine-spi-and-filesystem-builtin` capability，冻结 canonical memory SPI（`Query/Upsert/Delete`）与错误 taxonomy。
- 新增 `runtime.memory.mode=external_spi|builtin_filesystem`，支持启动/热更新原子切换与失败回滚。
- 内置文件系统 memory 引擎契约收敛（append-only WAL + 原子 compaction/index + crash-safe recovery）。
- 新增主流 profile pack：`mem0`、`zep`、`openviking`、`generic`，并固定 required/optional capability 语义。
- CA2 Stage2 memory 路径统一经 memory facade，保持 Run/Stream 与 `fail_fast|best_effort` 语义等价。
- readiness/preflight 增加 `memory.*` findings；diagnostics 增加 memory additive 字段并保持 bounded-cardinality。
- replay 新增 `memory.v1` fixture 与 drift 分类；quality gate 新增 memory contract gate 并保持 shell/PowerShell parity。
- adapter manifest/template/migration/conformance 一体化扩展，覆盖 external SPI 与 builtin filesystem 双路径接入。

memory provider SPI + builtin filesystem engine contract 当前阶段非目标（不做）：
- 不引入平台化 memory 控制面或跨租户调度系统。
- 不改 sandbox execution isolation contract/sandbox runtime rollout + health/capacity governance contract sandbox contract 语义，只复用其治理框架。
- 不承诺外部 provider 底层实现一致，仅要求 canonical 合同输出一致。

memory provider SPI + builtin filesystem engine contract 风险与回滚点：
- 主要风险：外部 provider 能力差异导致 profile 语义不一致、模式切换误配导致运行抖动、文件系统 compaction 恢复窗口处理不当。
- 缓解策略：required/optional capability 分层、切换前 preflight 校验、WAL + 原子替换 + crash-recovery 合同测试。
- 回滚点：切换回 `builtin_filesystem` 或 `external_spi` 上一稳定配置快照；热更新失败一律原子回滚。

memory provider SPI + builtin filesystem engine contract 验证命令（提案实施期最小集合）：
- `go test ./context/... ./runtime/config ./runtime/diagnostics -count=1`
- `go test ./integration -run 'TestMemoryProviderSPI|TestMemoryModeSwitch|TestMemoryRunStreamParity' -count=1`
- `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractMemoryFixture' -count=1`
- `go test -race ./...`
- `golangci-lint run --config .golangci.yml`
- `bash scripts/check-memory-contract-conformance.sh`
- `pwsh -File scripts/check-memory-contract-conformance.ps1`
- `pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-docs-consistency.ps1`

memory provider SPI + builtin filesystem engine contract gate 交付口径（当前实现）：
- memory contract suites 以 `smoke|full` 分层执行（主线 quality gate 默认 smoke，CI 独立 `memory-contract-gate` job 默认 full）。
- shell 与 PowerShell 脚本保持同一阻断语义（native command 非零即 fail-fast）。

### P1：react loop + tool-calling parity contract react loop + tool-calling parity contract（已归档）

react loop + tool-calling parity contract Why now：
- Run/Stream 在工具闭环路径长期存在语义偏移风险（step 边界、dispatch、feedback 与终止 reason 不完全同构）。
- provider tool-calling 映射与 readiness/admission/sandbox 语义在多提案叠加后需要统一收敛到单一 contract 口径。
- 需要把 ReAct 主题一次性接入 replay + gate，避免后续分散修补。

react loop + tool-calling parity contract 当前落地（截至 2026-03-31）：
- loop 与 taxonomy：Run/Stream 共享 ReAct termination taxonomy（`react.completed`、预算耗尽、dispatch 失败、provider 错误、取消）。
- provider canonicalization：`model/openai|anthropic|gemini` 的 tool-call request/feedback 映射与 provider error taxonomy 已收敛。
- readiness/admission：新增 `react.*` finding（loop/stream dispatch/provider/tool registry/sandbox dependency）并贯通 strict/non-strict 与 admission deny/allow 语义。
- sandbox consistency：ReAct 多轮 host/sandbox/deny、fallback、capability mismatch 在 Run/Stream 下已具备 contract parity。
- replay/gate：`tool/diagnosticsreplay` 新增 `react.v1` fixture 与 drift 分类；`scripts/check-react-contract.sh/.ps1` 已接入 `check-quality-gate.*`；CI 已暴露 `react-contract-gate`。
- docs/examples：README、runtime-config-diagnostics、mainline index 与示例文档已补齐 ReAct 最小接入、字段与门禁映射。

react loop + tool-calling parity contract 一次性闭环审查（10.4）：
- 审查范围：`loop -> provider -> readiness -> admission -> sandbox -> replay -> gate -> docs`。
- 审查结论：上述链路已形成同一 contract 语义闭环，当前没有必须再拆分的 ReAct 后续子提案。
- 剩余动作：执行全量回归验证（`go test`/`race`/`lint`/gate/docs consistency）并完成提案归档流程。

### P1/P2：post-policy baseline proposal pool 候选提案池（全局视角）

前提约束（冻结）：
- 不调整 react loop + tool-calling parity contract/sandbox egress governance + adapter allowlist contract 的既有范围、完成条件与验收口径；后续提案仅做增量扩展。
- 新提案必须复用既有治理主链路：`runtime/config`（`env > file > default` + fail-fast/回滚）+ `RuntimeRecorder` 单写 + diagnostics replay + quality gate。
- 对齐主流框架时，优先补齐“可互操作 contract”缺口（guardrail precedence、memory 分层治理、OTel tracing/eval），避免散点功能堆叠。

补充参考（主流框架实现与设计查询，2026-03-31 对齐）：
- 本轮“无遗漏”对比项目（官方文档优先）：
  - Coding Agent Runtime：Claude Code、OpenAI Codex、DeerFlow 2.0（明确采用 2.0 口径，不混用 1.x 结论）。
  - Agent 编排框架：LangGraph、LlamaIndex Workflows、AutoGen、Semantic Kernel、CrewAI、Agno、AgentScope。
  - Memory 框架/引擎：Mem0、Zep、OpenViking、OpenClaw（并回看当前内置 filesystem memory 实现）。
- 对齐维度统一采用 7 项：`权限/审批`、`sandbox 边界`、`subagent/多 agent 编排`、`memory 分层与生命周期`、`tool/MCP 接入治理`、`HITL 中断恢复`、`observability/eval`。
- 关键实现信号（用于约束 post-policy baseline proposal pool 设计，不额外开新主线）：
  - Claude Code：managed/project/user 分层配置 + 权限规则、hook 事件化拦截、subagent 粒度权限与 MCP/memory 作用域；
  - Codex：`sandbox_mode` 与 `approval_policy` 分离治理、workspace-write 默认模型、cloud setup/agent 两阶段与 agent phase 默认断网、AGENTS.md 分层覆盖；
  - DeerFlow 2.0：local/docker/k8s sandbox 模式、host bash 默认关闭、subagent 并行与上下文隔离、local long-term memory、LangSmith tracing；
  - LangGraph/AutoGen/LlamaIndex/Semantic Kernel：强调持久化 checkpoint、HITL interrupt/resume、工作流级状态可回放；
  - CrewAI/Agno：强调角色编排、memory 与 tracing 结合、团队级任务分解与可观测；
  - AgentScope：强调 lifecycle hooks + middleware、统一 state/session 管理、plan notebook 与实时双向事件协议；
  - Mem0/Zep/OpenViking/OpenClaw：强调多层 memory（session/user/agent）、检索/重排、保留策略与 provider 互换能力。
- 一次性补齐项目归并（保持现有优先级，不再拆平行提案）：
  - policy precedence + decision trace contract：统一策略栈 precedence（action/sandbox/egress/allowlist/admission）+ 决策解释链，补齐跨入口判定一致性；
  - memory scope + builtin filesystem governance contract：一次性补齐 memory scope、写入模式、检索质量、生命周期（retention/ttl/forget）与 builtin filesystem v2 治理；
  - runtime cost/latency budget admission contract：统一 token/tool/sandbox/memory 成本与时延预算 admission 规则；
  - OTel tracing + agent eval interoperability contract：补齐 OTel tracing 语义映射、agent eval contract，并合并 local/distributed evaluator 执行治理；
  - agent lifecycle hooks + tool middleware contract：补齐 agent lifecycle hooks + tool middleware 合同，统一横切扩展面；
  - unified state/session snapshot contract：补齐统一 state/session snapshot 合同，打通跨模块恢复与迁移；
  - react plan notebook + plan-change hook contract：补齐 ReAct plan notebook + plan-change hook 合同，增强动态计划可控性；
  - realtime event protocol + interrupt/resume contract：实时双向事件协议专项（按业务触发），补齐 cancel/resume 与事件幂等合同；
  - Context JIT Organization：补齐 JIT context organization 合同（reference-first/progressive disclosure/write-compress-isolate），提升 ReAct 模式可用性与上下文效率；
  - codebase consolidation and semantic labeling contract：代码整合收敛专项（清理临时代码/文档、命名语义化、目录结构收敛），以“语义不变”为硬约束；
  - a64：工程优化&性能优化专项（goroutine pool、buffer/slice pool、批量导出、Context Assembler 热路径治理），以“语义不变”为硬约束；
  - a69：context compression production hardening（语义压缩 + 冷热分层 + 冷存治理的生产可用收口）；
  - a62：补齐“交付易用性”example pack（主要 agent 模式一站式示例与可回归冒烟）；
- 执行约束：policy + memory + budget + tracing baseline contracts 负责核心 runtime contract 缺口，hooks/snapshot/plan/realtime baseline contracts 负责 agent runtime 基座能力补齐（realtime event protocol + interrupt/resume contract 按实时交互需求触发），Context JIT Organization 负责 ReAct/JIT context 组织合同收敛，codebase consolidation and semantic labeling contract 负责代码整合收敛，a64 负责非语义性能工程化，a69 负责 context 压缩与冷存生产可用治理，a62 在前述能力相对稳定后承担交付易用性收口（example pack）；除非战略边界变化，不再新增同域提案，避免重复提案与重复改造。

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
1. policy precedence + decision trace contract（已归档，P1）：policy precedence + decision trace contract（优先承接跨层策略冲突风险）。
2. memory scope + builtin filesystem governance contract（已归档，P1）：memory scope + builtin filesystem memory v2 治理 contract（scope/write_mode/injection_budget/lifecycle/search 与 gate 已收口）。
3. runtime cost/latency budget admission contract（已归档，P2）：runtime 成本/时延预算与 admission contract（原 memory scope + builtin filesystem governance contract 顺延）。
4. OTel tracing + agent eval interoperability contract（已归档，P2）：OTel tracing + agent eval 互操作 contract（含 local/distributed evaluator 执行治理）。
5. agent lifecycle hooks + tool middleware contract（已归档，P2）：agent lifecycle hooks + tool middleware contract。
6. unified state/session snapshot contract（已归档，P2）：unified state/session snapshot contract。
7. react plan notebook + plan-change hook contract（已归档，P2）：react plan notebook + plan-change hook contract。
8. realtime event protocol + interrupt/resume contract（已归档，P2）：realtime event protocol + interrupt/resume contract。
9. Context JIT Organization（已归档，P2）：jit context organization + reference-first assembly contract（ReAct 场景上下文组织专项）。
10. codebase consolidation and semantic labeling contract（进行中，P2）：codebase consolidation and semantic labeling contract（代码收敛与语义化整顿）。
11. a64（进行中，P2）：engineering/performance optimization contract（语义不变前提下性能收敛）。
12. a69（候选，P2）：context compression production hardening contract（语义压缩 + 冷热分层 + 冷存治理生产化）。
13. a71（进行中，P2）：real runtime agent mode examples contract（真实示例全量替换与收口）。

后续项目说明（避免“单一路线”误解）：
- codebase consolidation and semantic labeling contract（进行中）与 Context JIT Organization（已归档）、realtime event protocol + interrupt/resume contract（已归档）、a64/a69/a71（a64/a71 进行中，a69 候选）构成后续提案池，默认按上方顺序推进，但允许按风险信号前置切换，不要求机械串行实施。
- policy + memory + budget + tracing baseline contracts 已归档，用作稳定基线，不再作为当前推进主路径。
- 前置切换仅在以下风险信号出现时触发：实时交互压力（realtime event protocol + interrupt/resume contract）、上下文组织漂移（Context JIT Organization）、context 压缩生产可用风险（a69）、命名/文档收敛压力（codebase consolidation and semantic labeling contract）、性能回归压力（a64）、交付易用性压力（a62）。
- a64 前置时仍按 `a64-S1 -> ... -> a64-S10` 风险链路吸收，允许按瓶颈调整顺序。
- 无论是否前置切换，均不得改写 react loop + tool-calling parity contract 已归档与 sandbox egress governance + adapter allowlist contract 已冻结范围，只允许在其完成后做增量扩展。

提案 policy precedence + decision trace contract（已归档）：`introduce-policy-precedence-and-decision-trace-contract-policy precedence + decision trace contract`
- 目标：统一 ActionGate、Security S2、sandbox action/egress、adapter allowlist、readiness/admission 的策略判定优先级与解释链路，防止并行改造后出现判定冲突。
  - 范围：
  - 固化跨策略层 precedence matrix 与 deterministic tie-break；
  - 统一 deny source taxonomy 与 explainability 字段；
  - 增加 `policy_stack.v1` replay fixture 与 drift 分类；
  - 增加独立 `check-policy-precedence-contract.*` gate。
- 当前落地（已完成）：
  - `check-policy-precedence-contract.sh/.ps1` 已接入 `check-quality-gate.sh/.ps1`；
  - CI 已暴露独立 required-check 候选 `policy-precedence-gate`；
  - replay 已覆盖 `policy_stack.v1` 与 mixed compatibility（`arbitration version governance + compatibility contract.v1` + `react.v1` + `sandbox_egress.v1` + `policy_stack.v1`）。
- Why now（紧急性）：sandbox egress governance + adapter allowlist contract 联调改动 runner/sandbox/readiness/admission，若缺少统一 precedence contract，极易产生“同请求不同入口判定不一致”的高风险回归。

提案 sandbox egress governance + adapter allowlist contract：`introduce-sandbox-egress-governance-and-adapter-allowlist-contract-sandbox egress governance + adapter allowlist contract`（已归档）
- 目标：补齐 sandbox 网络外呼治理（egress policy）与 adapter 供应链 allowlist 契约，形成“执行隔离 + 出口治理 + 激活准入”闭环。
- 范围：`security.sandbox.egress.*`、`adapter.allowlist.*`、readiness/admission finding、taxonomy、replay drift 与 conformance matrix。
- 门禁：`check-sandbox-egress-allowlist-contract.sh/.ps1`（已纳入 `check-quality-gate.sh/.ps1`），CI 独立 required-check 候选为 `sandbox-egress-allowlist-gate`。
- 依赖：复用 sandbox execution isolation contract/sandbox runtime rollout + health/capacity governance contract/mainstream sandbox adapter conformance + migration pack contract sandbox taxonomy 与 adapter manifest 激活边界，不新增平行安全语义。
- 启动条件：存在合规审计或外部 adapter 引入规模上升，需要可审计可阻断的 egress/allowlist 治理。

备选 memory scope + builtin filesystem governance contract（合并版）：`introduce-memory-scope-and-builtin-filesystem-v2-governance-contract-memory scope + builtin filesystem governance contract`
- 目标：在 memory provider SPI + builtin filesystem engine contract SPI 基线之上合并推进两类能力：
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
- 依赖：memory provider SPI + builtin filesystem engine contract memory facade/profile/readiness 字段稳定后扩展，避免与 react loop + tool-calling parity contract/sandbox egress governance + adapter allowlist contract 实施冲突。
- 启动条件：出现 memory 注入不可解释、检索召回不足、或本地文件 memory 在恢复/索引一致性上的风险信号。

备选 runtime cost/latency budget admission contract：`introduce-runtime-cost-latency-budget-and-admission-contract-runtime cost/latency budget admission contract`
- 目标：统一 token/tool/sandbox/memory 成本与时延预算，建立 admission 侧 fail-fast 与降级策略。
- 启动条件：成本或 P95 抖动成为主线瓶颈。

备选 OTel tracing + agent eval interoperability contract（新增）：`introduce-otel-tracing-and-agent-eval-interoperability-contract-OTel tracing + agent eval interoperability contract`
- 目标：补齐主流框架常见的“可观测 + 评测”互操作治理，降低跨平台对接成本并固定回归口径。
- 对标主流（OpenAI Agents / Agno / CrewAI / AgentScope）的补齐方向：
  - tracing 语义：对齐 OTel 场景下 run/model/tool/mcp/memory/hitl 关键 span/attribute 映射；
  - tracing 导出：保证不引入平台控制面的前提下，支持主流 OTel backend 稳定接入；
  - 评测基线：新增最小 agent eval contract（任务成功率、工具调用正确率、拒绝/拦截准确率、cost-latency 约束）；
  - 评测执行治理（合并项）：在 OTel tracing + agent eval interoperability contract 内一次性支持 `local|distributed` evaluator execution、分片汇总、失败重试、断点续跑与结果幂等聚合；
  - 回放与门禁：增加 `otel_semconv.v1`、`agent_eval.v1`、`agent_eval_distributed.v1` fixtures，新增 `check-agent-eval-and-tracing-interop-contract.*`。
- 依赖：observability export + diagnostics bundle contract observability export + diagnostics bundle 稳定后扩展；建议在 policy precedence + decision trace contract decision trace 字段冻结后接入。
- 启动条件：出现 tracing 字段跨后端解释不一致、外部可观测平台接线成本高、或缺少稳定 agent 质量回归基线。
- 约束项（新增）：
  - Non-goals：不引入托管评测控制面、远程评测任务调度服务、平台化 UI/RBAC/多租户运维面板。
  - Gate 边界断言：`check-agent-eval-and-tracing-interop-contract.*` 必须包含 `control_plane_absent` 断言（distributed execution 仅作为库内执行策略，不新增服务化控制面依赖）。

提案 agent lifecycle hooks + tool middleware contract（已实施，待归档）：`introduce-agent-lifecycle-hooks-and-tool-middleware-contract-agent lifecycle hooks + tool middleware contract`
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
  - required-check 候选：`hooks-middleware-contract-gate`。
- 硬约束（简版）：
  - 不绕过 policy precedence + decision trace contract precedence、sandbox egress governance + adapter allowlist contract 安全治理与 `RuntimeRecorder` 单写入口；
  - skill discovery source 切换不得绕过既有 trigger scoring/budget 与 skill 观测事件口径；
  - `Discover/Compile` 预处理与 `SkillBundle` 映射在 Run/Stream 下必须等价，不得引入“预处理只在单入口生效”的分叉；
  - whitelist 映射不得突破 sandbox egress governance + adapter allowlist contract adapter allowlist 与 sandbox/egress 治理上界。
  - Hook/Middleware 失败语义必须 deterministic，不引入 Run/Stream 分叉。
- 当前状态：已实施（OpenSpec tasks 完成，待归档）。

提案 unified state/session snapshot contract（已归档）：`introduce-unified-state-and-session-snapshot-contract-unified state/session snapshot contract`
- 目标：统一 runtime state/session snapshot 导入导出合同，打通跨模块恢复、迁移与重放。
- 范围（简版）：
  - state surface：runner/session、memory、workflow/composer/scheduler 的统一 state descriptor；
  - snapshot contract：版本化 schema、部分恢复、字段兼容窗口、冲突 fail-fast；
  - 恢复治理：增量恢复、幂等重放、跨后端一致性检查；
  - 回放与门禁：新增 `state_session_snapshot.v1` fixture 与 `check-state-snapshot-contract.*`；
  - CI 候选：新增 `state-snapshot-contract-gate` required-check 候选。
- 硬约束（简版）：
  - 不重写已有 checkpoint/snapshot 语义，仅做统一合同层；
  - 不引入平台控制面或远程状态服务依赖。
- 当前状态：已归档（详见 `openspec/changes/archive/111-introduce-unified-state-and-session-snapshot-contract-unified state/session snapshot contract`）。

提案 react plan notebook + plan-change hook contract（已归档）：`introduce-react-plan-notebook-and-plan-change-hook-contract-react plan notebook + plan-change hook contract`
- 目标：补齐 ReAct 动态计划治理（Plan Notebook）与计划变更 hook，提升复杂任务可控性与可解释性。
- 范围（简版）：
  - plan notebook：`create|revise|complete|recover` 生命周期；
  - plan-change hook：计划变更前后回调、变更原因与上下文快照；
  - 配置域：`runtime.react.plan_notebook.*`、`runtime.react.plan_change_hook.*`（默认关闭，`env > file > default`，非法配置 fail-fast + 热更新原子回滚）；
  - 观测与回放：新增 `react_plan_id`、`react_plan_version`、`react_plan_change_total`、`react_plan_last_action`、`react_plan_change_reason`、`react_plan_recover_count`、`react_plan_hook_status` 与 `react_plan_notebook.v1` fixture；
  - 门禁：新增 `check-react-plan-notebook-contract.*`，并暴露 CI required-check 候选 `react-plan-notebook-gate`。
- 硬约束（简版）：
  - 复用 react loop + tool-calling parity contract ReAct termination taxonomy，不新增平行 loop 语义；
  - 计划治理不得绕过 policy precedence + decision trace contract 决策链与 sandbox egress governance + adapter allowlist contract 安全链路。
- 当前状态：已归档（详见 `openspec/changes/archive/112-introduce-react-plan-notebook-and-plan-change-hook-contract-react plan notebook + plan-change hook contract`）。

提案 realtime event protocol + interrupt/resume contract（进行中）：`introduce-realtime-event-protocol-and-interrupt-resume-contract-realtime event protocol + interrupt/resume contract`
- 目标（简版）：补齐实时双向事件协议（server/client）与 interrupt/resume 合同，支撑实时交互场景。
- 范围（简版）：
  - 事件协议：请求、增量输出、取消、恢复、确认、错误的 canonical event taxonomy；
  - 会话治理：事件去重、顺序保证、重连恢复与幂等处理；
  - 回放与门禁：新增 `realtime_event_protocol.v1` fixture 与 `check-realtime-protocol-contract.*`。
- 一次性补齐边界（realtime event protocol + interrupt/resume contract 内闭环）：
  - 本提案内一次性冻结 realtime 事件信封、序列推进、resume 游标、ack/nack、去重键与错误分层合同；
  - realtime 同域新增需求（事件类型扩展、恢复语义、去重/顺序治理、回放分类、门禁断言）仅允许在 realtime event protocol + interrupt/resume contract 内增量吸收，不再新增平行提案。
- 硬约束（简版）：
  - 不引入平台化实时网关或托管控制面；
  - 协议语义必须与 react loop + tool-calling parity contract/policy precedence + decision trace contract/react plan notebook + plan-change hook contract 的主链路解释字段保持一致。
- 约束项（新增）：
  - Non-goals：不引入托管会话路由/连接管理控制面、实时 SaaS 运维面板或平台级常驻网关服务。
  - Gate 边界断言：`check-realtime-protocol-contract.*` 必须包含 `realtime_control_plane_absent` 断言（协议实现仅限库内 contract + adapter 接缝，不新增网关服务依赖）。
- 退出条件（DoD）：
  - interrupt/resume 在 Run/Stream 与 replay 下语义等价，事件顺序/幂等断言稳定通过；
  - realtime contract 的增量需求可在 realtime event protocol + interrupt/resume contract tasks 内吸收，不再拆分 realtime event protocol + interrupt/resume contract 平行子提案。
- 当前状态：进行中（OpenSpec `in-progress`）。

提案 Context JIT Organization（已归档）：`introduce-jit-context-organization-and-reference-first-assembly-contract-react plan notebook + plan-change hook contract-ctx`
- 目标：以“顺滑支撑 ReAct 模式”为导向，在不破坏既有 CA 合同语义前提下，一次性补齐 JIT context organization 的核心契约，降低上下文噪声与膨胀风险。
- 范围（聚焦 6 项，避免后续重复拆提案）：
  - reference-first stage2：新增 `discover_refs -> resolve_selected_refs` 两段式注入路径，优先注入引用（path/id/type/locator），按需再展开正文；
  - isolate handoff：定义子代理回传合同（`summary`、`artifacts[]`、`evidence_refs[]`、`confidence`、`ttl`），主代理默认只消费摘要与引用；
  - context edit gate：引入 `clear_at_least` 类收益阈值（预计释放 token 与上下文稳定性成本比对），仅在收益达标时触发激进编辑；
  - relevance swap-back：将 spill/swap-back 从“按 run 回填”扩展为“按当前 query + evidence tag 相关性回填”；
  - lifecycle tiering：统一 `hot|warm|cold` 上下文分层与 TTL/淘汰治理，串联 write/compress/prune/spill 策略；
  - task-aware recap：将 tail recap 从固定模板升级为“基于本轮实际选择/剪裁/外化动作”的结构化总结。
- Contract 字段（最小集，建议）：
  - `runtime.context.jit.reference_first.*`
  - `runtime.context.jit.isolate_handoff.*`
  - `runtime.context.jit.edit_gate.*`
  - `runtime.context.jit.swap_back.*`
  - `runtime.context.jit.lifecycle_tiering.*`
  - QueryRuns additive：`context_ref_discover_count`、`context_ref_resolve_count`、`context_edit_estimated_saved_tokens`、`context_edit_gate_decision`、`context_swapback_relevance_score`、`context_lifecycle_tier_stats`、`context_recap_source`
- Replay 与 Gate（最小集合）：
  - fixtures：`context_reference_first.v1`、`context_isolate_handoff.v1`、`context_edit_gate.v1`、`context_relevance_swapback.v1`、`context_lifecycle_tiering.v1`
  - drift 分类至少包含：`reference_resolution_drift`、`isolate_handoff_drift`、`edit_gate_threshold_drift`、`swapback_relevance_drift`、`lifecycle_tiering_drift`、`recap_semantic_drift`
  - 独立门禁：`check-context-jit-organization-contract.sh/.ps1`
- 一次性补齐边界（Context JIT Organization 内闭环）：
  - 上下文组织同域需求统一在 `reference-first + isolate handoff + edit gate + relevance swap-back + lifecycle tiering + task-aware recap` 六件套内吸收；
  - 不再新增平行 context 组织提案，只允许在 Context JIT Organization tasks 中补充增量 contract/replay/gate。
- 硬约束（简版）：
  - 不改变 react loop + tool-calling parity contract ReAct loop 终止 taxonomy 与 policy precedence + decision trace contract 决策解释语义；
  - 不绕过 sandbox egress governance + adapter allowlist contract sandbox/egress/allowlist 治理链路；
  - `context/*` 继续禁止直连 provider 官方 SDK，检索与模型能力经既有抽象接入；
  - 运行态写入仅走 `RuntimeRecorder` 单写入口，新增字段保持 `additive + nullable + default`。
- 依赖与排序：放在 realtime event protocol + interrupt/resume contract 之后、codebase consolidation and semantic labeling contract 之前；realtime event protocol + interrupt/resume contract 若启用，其 interrupt/resume 事件语义需作为本提案的上下文分层与回填边界输入。
- 退出条件（DoD）：
  - 六类 fixture 与 gate 全绿，且 Run/Stream 语义不漂移；
  - context 组织新增诉求可在 Context JIT Organization 内闭环吸收，不再拆分平行提案。
- 当前状态：已归档（详见 `openspec/changes/archive/113-introduce-jit-context-organization-and-reference-first-assembly-contract-react plan notebook + plan-change hook contract-ctx`）。

提案（已完成待归档）：`introduce-codebase-consolidation-and-semantic-labeling-contract-codebase consolidation and semantic labeling contract`
- 目标（简版）：在不改变运行时语义前提下，完成仓库“代码与文档收敛整顿”，降低历史负担与命名歧义。
- 范围（简版）：
  - 临时文档/目录治理：清理或归档 `docs/drafts`、示例与脚手架生成物等临时资产，建立统一收口规则；
  - 离线生成物治理：收敛 `examples/adapters/_a23-offline-work/*` 这类离线 scaffold 产物，仅保留最小可复现样本与索引说明，其余转离线缓存或清理；
  - Context Assembler 统一命名（强制范围）：在活动代码/测试/脚本/文档中统一语义命名，收敛后不再保留 `ca1`、`ca2`、`ca3`、`ca4` 作为模块/阶段命名口径；
  - Axx 字眼消除（强制范围）：在活动代码、测试、文档、脚本、示例说明中移除 `Axx` 编号表述并替换为语义化描述；Spec 编号映射仅集中保留在索引文档，不在实现与用户向说明中散落耦合；
  - 阶段性工具命名治理：`cmd/*` 与 `scripts/*` 中编号化阶段命名（如 `ca3-threshold-*`、`ca4-benchmark-*`）统一收敛到语义主入口，并删除编号化兼容入口，避免新入口继续放大编号耦合；
  - 临时注释与占位清理：清理 `TODO/future milestone` 类临时注释并转化为 roadmap/index 可追踪事项，避免代码内长期悬挂。
  - 当前已转化到 roadmap 的代码内延后项：`model/providererror` 的错误细粒度拆分与 `context/assembler` 的 embedding provider 绑定，均通过稳定 SPI/契约边界推进，不再在代码中保留 TODO 占位。
  - 兼容清退与回滚（补漏）：对公开配置键、诊断字段、脚本入口和测试夹具执行语义主名收敛并删除历史编号兼容桥，同时保留可回滚变更记录，确保“语义不变 + 行为不变”；
  - 回流阻断（补漏）：增加命名治理扫描门禁（shell/PowerShell 等价），阻断 `ca2|ca3|ca4|A[0-9]{2,3}` 在活动目录回流；
  - 语义词表集中化（补漏）：维护唯一“语义名称 <-> 历史编号/旧名”映射表，供代码注释、README、脚本帮助信息与测试命名统一引用。
  - 运行时 Harness 架构文档收口（补漏）：补齐单一总览文档（建议 `docs/runtime-harness-architecture.md`），统一描述 `state surfaces -> guides/sensors -> tool mediation -> entropy control` 与主干 contract/gate 映射，避免语义散落在多文档重复维护。
- 硬约束（简版）：
  - 不改变 Run/Stream、readiness/admission、reason taxonomy、diagnostics/replay 契约语义；
  - 删除活动目录中的历史编号兼容入口与兼容别名，统一仅保留语义主名；
  - 所有重命名或目录调整必须提供可回滚路径（不要求保留兼容跳板）。
  - 编号化保留边界：`openspec/changes` 与 `openspec/changes/archive` 作为历史索引允许保留编号，代码与用户向文档默认使用语义名称。
- 回放与门禁（最小集合）：
  - docs 一致性门禁：`check-docs-consistency.*`；
  - 语义稳定门禁：`check-quality-gate.*` + 受影响 contract/replay suites；
  - 编号映射治理：新增/维护集中索引并纳入 docs consistency 校验，阻断散落编号回流。
- 一次性补齐边界（本提案内闭环）：
  - 文档与命名收敛同域需求（临时文档清理、语义命名、编号映射集中化、README 完整性对齐）统一在本提案内吸收，不再平行拆分“文档清理/命名治理”提案。
- 退出条件（DoD）：
  - 核心模块 README 与实际实现一致，临时资产完成归档/清理并可追踪；
  - 活动代码/测试/脚本/文档中不再存在 `ca|ca2|ca3|ca4` 作为 Context Assembler 命名口径；
  - 活动代码、测试与文档描述不再依赖散落 `Axx` 字眼，编号映射仅保留在索引层；
  - 命名治理 gate 持续阻断旧命名回流，且不改变既有 contract/replay 语义。
- 当前状态：已完成（OpenSpec `all_done`，待归档）；与 a64 并行推进阶段的语义不变约束已完成闭环验证。

提案 a64（进行中）：`introduce-engineering-and-performance-optimization-contract-a64`
- 目标（简版）：在“语义不变”前提下推进工程优化与性能优化（如 goroutine pool、buffer/slice pool、导出批处理等常规路径）。
- 一次性补齐边界（a64 内闭环）：
  - 性能同域需求统一按 a64-S1~S10 子项吸收；新增热点必须映射到现有子项或其增量任务，不再新增平行性能提案；
  - 每个子项合并前必须提供“优化前后基线 + 回归阈值 + 语义稳定证明”三件套。
- 子项目（性能治理，优先落地）：
  - a64-S1：`context-assembler-loop-hotpath-governance`
  - 目标：降低每轮 `Assemble` 固定开销与长跑内存累积风险，在不改变 CA1/CA2/CA3/CA4 语义前提下提升稳定吞吐。
  - 范围（第一批）：
    - 为 `prefixCache` / `ca3State` 增加 run-finished 清理与 TTL/LRU 上限治理，避免常驻进程无界增长；
    - 为 context journal 增加可开关批量写入路径（默认保持同步语义），并补齐 flush/异常中断边界测试；
    - 增加 CA3 stage2 “无增量跳过”优化开关（仅在 stage2 未追加有效上下文且输入签名不变时跳过第二次 CA3）；
    - 为 `stage2 provider=file` 增加索引化读取或分段扫描策略，降低大文件线性扫描成本；
    - 为 `stage2 provider=external(http/rag/db/elasticsearch)` 增加请求/响应编解码快路径与有界缓冲复用，降低 `json marshal/unmarshal + body read` 抖动；
    - 增加热点基准与回归门禁：`BenchmarkContextAssemblerLoop*`、`BenchmarkCA3Stage2Pass*`、`BenchmarkStage2FileProvider*`。
  - 非目标（第一批）：
    - 不修改 runtime cost/latency budget admission contract 成本/时延 budget admission 公式与降级动作；
    - 不调整 Run/Stream 行为、reason taxonomy、diagnostics 字段语义。
  - a64-S2：`runtime-recorder-and-diagnostics-hotpath-governance`
  - 目标：降低 `run.finished` 大负载映射、query 聚合与排序复制带来的 CPU/GC 抖动，保持 recorder/query 语义不变。
  - 范围（第一批）：
    - 为 `RuntimeRecorder` 增加可复用映射缓冲与按需字段投影，减少 `run.finished` 事件大对象重复分配；
    - 为 diagnostics store 查询路径增加可开关索引/分页游标策略，减少全量筛选 + 排序 + 复制；
    - 为 `MailboxAggregates`/P95 聚合引入有界统计优化（保持输出字段与解释口径不变）；
    - 增加 `BenchmarkRuntimeRecorderRunFinished*`、`BenchmarkDiagnosticsQueryRuns*`、`BenchmarkDiagnosticsMailboxAggregates*` 与回归 gate。
  - 非目标（第一批）：
    - 不改写 `RuntimeRecorder` 单写入口契约；
    - 不变更 QueryRuns/QueryMailbox 对外字段、排序解释与 replay 语义。
  - a64-S3：`scheduler-mailbox-file-backend-persistence-governance`
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
  - a64-S4：`mcp-transport-invoke-and-event-emission-hotpath-governance`
  - 目标：降低 MCP stdio/http 调用链 goroutine 峰值与事件发射分配开销，保持 call contract 与诊断解释稳定。
  - 范围（第一批）：
    - 为 stdio/http client 的 invoke 路径增加有界 worker/复用策略开关，抑制每调用 goroutine 激增；
    - 为 MCP 事件发射 map 构建引入模板复用与延迟填充，减少短生命周期分配；
    - 增加 `BenchmarkMCPInvokePath*`、`BenchmarkMCPEventEmit*` 与 transport 回归 gate。
  - 非目标（第一批）：
    - 不调整 MCP 对外 API、重试/超时语义或错误 taxonomy；
    - 不改变已有 tracing/diagnostics 字段口径。
  - a64-S5：`skill-loader-discover-compile-io-and-scoring-governance`
  - 目标：降低 skill discover/compile 重复 I/O 与评分路径开销，保证 `agents.md|folder|hybrid` 解析结果一致。
  - 范围（第一批）：
    - 为 discover/compile 建立可开关元数据缓存与文件读取复用，避免同轮重复解析；
    - 为评分 tokenization/sort 路径引入有界缓存与短路策略（仅优化实现，不改分数语义）；
    - 增加 `BenchmarkSkillLoaderDiscover*`、`BenchmarkSkillLoaderCompile*`、`BenchmarkSkillSelectionScore*` 与回归 gate。
  - 非目标（第一批）：
    - 不改 discovery precedence、去重顺序、`SkillBundle -> prompt/tool whitelist` 映射语义；
    - 不替代 agent lifecycle hooks + tool middleware contract/a62 的 skill contract 主线治理职责。
  - a64-S6：`memory-filesystem-engine-write-query-index-governance`
  - 目标：降低 filesystem memory 引擎 WAL 写入、查询排序与索引重建成本，保持 memory scope + builtin filesystem governance contract scope/lifecycle/search 契约不变。
  - 范围（第一批）：
    - 为 WAL 增加可开关批量 fsync/组提交策略（默认保留现有 durability 语义）；
    - 为 query 路径引入命名空间级索引/缓存与稳定排序复用，降低每次全量扫描；
    - 为索引 checksum/compaction 增加分段重建与后台节流治理；
    - 增加 `BenchmarkMemoryFilesystemWrite*`、`BenchmarkMemoryFilesystemQuery*`、`BenchmarkMemoryFilesystemCompaction*` 与回归 gate。
  - 非目标（第一批）：
    - 不修改 memory scope + builtin filesystem governance contract 的 scope resolution、retrieval quality 阈值、lifecycle policy 与可解释字段；
    - 不引入新 memory provider 协议面或第二套事实源。
  - a64-S7：`runner-loop-and-local-dispatch-hotpath-governance`
  - 目标：降低 Runner 每轮 timeline/run-finished 构造开销与 local tool dispatch 分类开销，稳定高迭代场景吞吐。
  - 范围（第一批）：
    - 为 Runner 引入 run-scope 配置快照/派生值复用，减少循环内重复 `EffectiveConfig()` 读取与大对象复制；
    - 为 `emitTimeline` / `runFinishedPayload` 增加可复用 payload 构建策略，降低 map 分配与拷贝；
    - 为 local dispatcher 的 `drop_low_priority` 分类链路增加关键字预编译与签名缓存，避免每调用重复排序/归一化；
    - 增加 `BenchmarkRunnerLoopHotpath*`、`BenchmarkRunnerTimelineEmit*`、`BenchmarkLocalDispatchPriorityClassify*` 与回归 gate。
  - 非目标（第一批）：
    - 不改变 action timeline 事件顺序、字段语义与 reason taxonomy；
    - 不改变 backpressure/retry/fail-fast 决策语义。
  - a64-S8：`provider-adapter-stream-and-decode-hotpath-governance`
  - 目标：降低 OpenAI/Anthropic/Gemini 适配器在流式事件映射与非流式解码阶段的分配与序列化开销。
  - 范围（第一批）：
    - 为 provider stream 事件映射引入 meta/payload 复用策略，减少每事件 map 临时分配；
    - 为 tool-call 参数解码增加快速路径与有界缓冲复用，降低高频 `json.Unmarshal` 抖动；
    - 为非流式响应解码优先使用 typed 字段读取，减少全量 `json.Marshal + gjson` 回退路径触发；
    - 增加 `BenchmarkProviderStreamEventMap*`、`BenchmarkProviderResponseDecode*` 与 provider parity gate。
  - 非目标（第一批）：
    - 不改变 provider capability 判定、tool_call 触发条件、事件顺序与 token usage 口径；
    - 不改写已有 provider 错误分类与重试语义。
  - a64-S9：`runtime-config-readpath-and-policy-resolve-hotpath-governance`
  - 目标：降低高并发场景下 runtime config 读取与 MCP policy 解析开销，保持配置治理与热更新语义稳定。
  - 范围（第一批）：
    - 为 runtime config 增加只读快照引用/派生缓存机制，减少频繁值拷贝；
    - 为 MCP runtime policy resolve 增加按 `profile + explicit override` 的可失效缓存（reload 后自动失效）；
    - 为关键热路径补齐 `BenchmarkRuntimeConfigReadPath*`、`BenchmarkMCPPolicyResolve*` 与回归 gate。
  - 非目标（第一批）：
    - 不改变 `env > file > default`、fail-fast 与热更新原子回滚语义；
    - 不改写 policy precedence、admission 与 sandbox rollout contract 字段。
  - a64-S10：`observability-event-pipeline-throughput-governance`
  - 目标：降低 observability 事件管线（dispatcher/logger/exporter）在高事件率场景下的分配与串行阻塞开销。
  - 范围（第一批）：
    - 为 runtime exporter 增加批量导出与批次聚合开关，替代逐事件 `ExportEvents([]event{...})` 热路径；
    - 为 dispatcher 增加可配置 fanout 策略与 handler 隔离治理，避免慢 handler 放大主链路延迟；
    - 为 JSON logger 增加编码器/缓冲复用与最小化字段构建路径，降低 per-event `map + json.Marshal` 开销；
    - 增加 `BenchmarkRuntimeExporterBatch*`、`BenchmarkEventDispatcherFanout*`、`BenchmarkJSONLoggerEmit*` 与回归 gate。
  - 非目标（第一批）：
    - 不改变事件 schema、timeline 序列、RuntimeRecorder 单写入口和 replay 字段语义；
    - 不引入平台化 observability 控制面或外置必选依赖。
- a64 增量补充（2026-04-05 复核，按 S1~S10 吸收，不新增平行提案）：
  - S1（context assembler + stage2 + journal）补充：
    - `sanitizeRecap` 去除多次 `json marshal/unmarshal` 往返路径，改为结构化字段级脱敏快路径（语义保持不变）；
    - `context/journal` 与 CA3 spill file backend 增加“句柄复用 + 批量 flush”可开关路径，降低 append 高频 open/close 抖动；
    - stage2 file/external provider 增加“预解码/复用缓冲”治理，限制大输入场景下重复分配。
  - S2（runtime recorder + diagnostics）补充：
    - query 聚合路径增加“锁内快照、锁外过滤/排序/聚合”策略，降低长时间读锁占用；
    - `percentileP95`/trend 聚合增加有界统计路径，避免重复全量 copy+sort。
  - S3（scheduler/mailbox/composer file backend）补充：
    - file store 增加 debounce/group-commit 选项与 flush 边界契约，降低每次状态变更全量 `marshal+rename` 放大；
    - task-board/mailbox 查询增加增量索引策略，减少全量过滤+排序频率。
  - S4（MCP transport）补充：
    - stdio/http 调用路径评估并收敛 `invokeAsync` 每调用 goroutine 包装层，优先复用现有 timeout/retry 框架；
    - `mcp/diag.Store` 增加 ring-buffer 语义，替换 overflow 时整段切片复制。
  - S5（skill loader）补充：
    - discover/compile 引入基于 `path + mtime + size` 的元数据缓存，减少重复读取与重复 parse；
    - tokenization/关键字优先级计算增加预编译缓存，避免同轮重复排序与构建。
  - S6（memory filesystem）补充：
    - query 路径拆分“TTL 维护写路径”与“只读查询路径”，默认查询不触发写锁；
    - snapshot/index 持久化增加增量/流式编码治理，降低全量序列化放大。
  - S7（runner + local dispatch）补充：
    - 将 payload 构建优化范围扩展到 `orchestration/teams`、`orchestration/workflow` 的 timeline/run-finished 热路径，统一复用构造策略；
    - runner `runFinishedPayload` 增加按需字段构建策略，降低大 map 组装成本。
  - S8（provider adapters）补充：
    - Anthropic/Gemini 非流式解码优先 typed fast-path，严格收敛 `json.Marshal + gjson` 回退触发频率；
    - stream tool-call 参数解码引入有界缓冲复用，减少高频 `json.Unmarshal` 抖动。
  - S9（runtime config + policy resolve）补充：
    - 提供只读配置引用/快照读取接口（避免热路径大对象值拷贝），并以 reload 版本号驱动失效；
    - policy resolve 增加派生结果 memoization（按 profile+override 签名）并保证 reload 后原子失效。
  - S10（observability pipeline）补充：
    - exporter 批量导出增加 `max_batch_size + max_flush_latency` 双阈值，避免低流量场景批次长期滞留；
    - dispatcher 增加可配置异步 fanout 与慢 handler 隔离策略，保持事件顺序与失败语义可验证。
  - S2/S9（inferential feedback 闭环）补充：
    - 在不改变 readiness/admission deny 语义前提下，新增“推断型反馈传感器” advisory 通道，接入 `runtime.eval.*` 与运行态质量信号，仅输出可观测建议，不直接改写阻断决策；
    - 复用既有 explainability 字段与 replay 框架，新增推断反馈 drift 夹具与分类，保证 Run/Stream/replay 语义等价。
  - S3（realtime/handoff 可恢复状态面）补充：
    - 补齐 realtime interrupt/resume cursor 与 isolate-handoff 关键状态的持久化恢复边界，优先复用 state/session snapshot 合同段扩展，不引入第二套事实源；
    - 增加 crash/restart/replay 一致性回归，确保恢复路径不改变 unified state/session snapshot contract/Context JIT Organization/realtime event protocol + interrupt/resume contract 既有语义。
  - S3/S9（snapshot 熵预算治理）补充：
    - 新增 snapshot `retention/quota/cleanup` 治理参数与 fail-fast 校验，默认行为保持不变；
    - 热更新非法值必须原子回滚，并补齐 bounded-cardinality 与回放一致性断言。
  - 横切工程优化补充（纳入 a64 强门禁）：
    - repo hygiene 扩展到未跟踪临时产物检测（`git ls-files --others --exclude-standard`），阻断 `*.go.<digits>` / `*.tmp` / `*.bak` / `*~` 回流；
    - a64 子项若新增/调整 benchmark，必须同步提供按模块可单独执行的基准入口，避免单一超大 benchmark 文件持续膨胀。
    - 新增 harnessability scorecard（contract 覆盖、drift 统计、主干 gate 覆盖、文档一致性）并接入质量门禁阻断，输出 machine-readable 报告供 PR/CI 消费。
    - 新增 multi-agent 涌现行为矩阵（并发/交错/重试/重放）与 drift 分类阻断，覆盖 scheduler/mailbox/composer、runner dispatch、observability fanout 等并发敏感路径。
    - 新增 harness ROI/depth 治理（token/latency/quality 三维指标 + 复杂度分层 + 超阈值降级），避免“harness 过度工程”。
    - 固化 `computational-first, inferential-second` 传感器分层：客观 correctness 阻断必须来自 computational suites，inferential 仅用于主观质量补充且需结构化证据。
    - 新增门禁执行效率治理：按改动路径映射 impacted suites，支持 `fast`（增量）/`full`（全量）执行层级；`fast` 只裁剪无关套件，不允许跳过 mandatory contract/perf suites。
    - 新增门禁耗时预算治理：记录 gate step 级耗时指标并设阈值回归阻断，防止 gate 退化拖慢交付反馈周期。
- 强门禁（a64 子项共用，阻断合入）：
  - 必须新增并接入 `check-a64-semantic-stability-contract.sh/.ps1`，阻断“对外语义漂移”：
    - Run/Stream 行为等价；
    - diagnostics schema 与 reason taxonomy 不漂移；
    - replay fixture idempotency 稳定。
  - 必须新增并接入 `check-a64-performance-regression.sh/.ps1`，阻断关键基准退化（`ns/op`、`allocs/op`、`B/op`）；
  - 必须新增并接入 `check-a64-impacted-gate-selection.sh/.ps1`，校验 changed-files 到 `a64 impacted-contract suites` 的映射与 `fast/full` 选择正确性；
  - 必须新增并接入 `check-a64-gate-latency-budget.sh/.ps1`，阻断 gate step 级耗时超阈值回归；
  - 必须接入 `a64 impacted-contract suites` 校验（按改动模块选择主干 contract suites），要求主干 contract/replay suites 全绿且无漂移豁免；
  - a64 任一子项未通过 contract/replay/perf gate，不允许合入主干。
- `a64 impacted-contract suites` 模块映射（最低必跑，shell/PowerShell 必须语义等价）：
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
  - 横切兜底（所有 a64 子项合并前必跑）：
    - `scripts/check-quality-gate.sh/.ps1`
- 硬约束（简版）：
  - 不改变 Run/Stream、backpressure、fail_fast、timeout/cancel、reason taxonomy、decision trace 语义；
  - 不绕过现有 contract gate 与 replay 约束；
  - 所有优化都必须可开关、可回滚。
- 当前状态：进行中（a64-S1~S10 子项目按风险链路增量吸收）；S10 `11.1~11.4` 与 `12.1~12.4/12.7~12.18` 已落地，`12.5/12.6` 已执行并形成阻断风险记录（语义命名债与 lint 历史债需独立收口）。

提案 a69（候选，建议前置于 a62）：`introduce-context-compression-production-hardening-contract-a69`
- 目标（production-ready）：在不改写 Context JIT Organization 已归档语义的前提下，补齐 context pressure `semantic compaction + spill/swap-back + lifecycle tiering` 的生产可用治理闭环。
- 范围（a69 内闭环，避免平行拆分）：
  - a69-S1 语义压缩稳定性治理：固化 semantic compaction 的质量门槛、降级策略与失败分类；补齐 `ToolResult` 压力与可压缩对象边界一致性，避免“估算计入但压缩无收益”。
  - a69-S2 冷热分层与回填策略治理：统一 `hot|warm|cold|pruned` 规则化迁移；swap-back 从“按文件顺序取前 N”升级为“相关性 + 新近性”优先。
  - a69-S3 file 冷存生产治理：补齐 `retention/quota/cleanup/compact`，控制 `context-spill.jsonl` 无限增长；保持默认 file backend 可独立运行。
  - a69-S4 一致性与恢复治理：补齐 crash/restart/replay 下的 spill/swap-back 幂等与去重一致性，不引入第二套状态事实源。
  - a69-S5 观测与回放治理：新增/收敛 context 压缩生产化 additive 字段、fixture 与 drift 分类，保持 `additive + nullable + default`。
  - a69-S6 强门禁治理：新增 `check-context-compression-production-contract.sh/.ps1` 并接入 `check-quality-gate.*`；与 `check-context-jit-organization-contract.*`、`check-context-production-hardening-benchmark-regression.*` 组成阻断集合。
- 非目标：
  - 不新增 context 能力族（reference-first/isolate/edit-gate/task-aware recap）语义，相关能力定义继续以 Context JIT Organization 为准。
  - 不替代 a64 性能治理职责；a69 聚焦“生产可用合同稳定性”，a64 聚焦“语义不变性能优化”。
- 与 a62 的排序关系（前置建议）：
  - a69 建议前置到 a62 之前：先冻结 context 压缩/冷存生产语义，再收口 `a62-T15 context-governed-reference-first` 等示例，降低示例回滚成本。
  - 已在进行中的 a62 非 context 主题任务可并行推进；涉及 context-governed 的完成判定以后置 a69 合同为准。
- 退出条件（DoD）：
  - context 压缩在长会话/高频工具调用场景下具备可预测收益与稳定回退；
  - 冷存文件具备有界增长治理（retention/quota/cleanup）且回填语义稳定；
  - Run/Stream/replay 无语义漂移，contract/perf gates 全绿。
- 当前状态：候选（roadmap 已登记，待建立 OpenSpec change）。

提案 a71（进行中）：`introduce-real-runtime-agent-mode-examples-contract-a71`
- 目标：将“主要 agent 模式”沉淀为可直接复用、可回归验证、与主线 contract 同步的 example pack，提升交付易用性与迁移效率。
- 模式覆盖（最低要求，PocketFlow + Baymax 扩展）：
  - PocketFlow 模式对齐：
    - `agent`（最小 chat/任务执行主链路）；
    - `workflow`（串行/条件分支/重试与 fail-fast 编排）；
    - `rag`（检索增强主链路：检索、注入、回答与回退）；
    - `mapreduce`（分片执行、聚合归并、错误分区处理）；
    - `structured output`（schema 约束输出、解析兼容、错误分类）；
    - `multi agents`（协作链路、异步通道、handoff/recovery）。
  - Baymax 特性扩展：
    - `skill-driven agent`（`AGENTS.md|folder|hybrid` 三类技能发现与映射）；
    - `mcp-governed agent`（`mcp/http|stdio` 接入、策略解析、诊断回放）；
    - `react agent`（推理-行动-观察闭环，Run/Stream 等价）；
    - `hitl-governed agent`（审批边界、人工接管、拒绝/恢复语义）；
    - `context-governed agent`（reference-first、isolate handoff、edit gate、tiering 治理）；
    - `sandbox-governed agent`（sandbox/egress/allowlist 治理链路可演示）；
    - `realtime interrupt/resume agent`（在 realtime event protocol + interrupt/resume contract 启用时纳入实时中断恢复矩阵）。
- 对标参考（示例组织方法）：
  - PocketFlow design patterns：`Agent / Workflow / RAG / MapReduce / Structured Output / Multi Agents` 的模式化分层与最小可运行示例组织；
  - 本仓库 `examples/01-09` 现有主链路示例（避免重复造样例，优先改造为统一模式矩阵）。
- 范围：
  - 建立 `examples/agent-modes` 统一目录或等价索引（支持按模式检索）；
  - 建立模式覆盖矩阵（建议 `examples/agent-modes/MATRIX.md`）：`pattern -> minimal/production-ish -> required contracts/gates -> diagnostics/replay coverage`；
  - 每种模式提供 `minimal` + `production-ish` 两档示例（前者用于上手，后者用于治理链路演示）；
  - `workflow/mapreduce` 示例需覆盖同步与异步（mailbox）两种执行路径，并验证 Run/Stream 语义等价；
  - `rag` 示例需覆盖本地 memory 检索与外部检索（含 MCP 数据源）两类来源，并包含 fail-fast/fallback 边界；
  - `structured output` 示例需覆盖 schema 校验、parser compatibility 与 drift 回放断言；
  - `multi agents` 示例需覆盖协作编排、并行执行与恢复语义，不引入第二套状态事实源；
  - 主干流程示例需覆盖 mailbox `sync/async/delayed/reconcile`、task-board `query/control`、scheduler `qos/backoff/dead-letter` 与 readiness/admission 降级链路；
  - `mcp-governed` 与 `skill-driven` 示例需覆盖配置优先级、发现顺序、权限/allowlist 边界与诊断字段稳定性；
  - 自定义 adapter 示例需覆盖 `mcp/model/tool/memory` 四类接入与 `manifest/capability/profile-replay/health-readiness-circuit` 治理链路；
  - `hitl-governed` 示例需覆盖人工审批 checkpoint、拒绝后回退路径、人工恢复与超时语义；
  - `context-governed` 示例需覆盖 `reference-first + isolate handoff + edit gate + lifecycle tiering` 组合路径与回放断言；
  - 在 Context JIT Organization 启用时，`react` 模式必须补齐 `reference-first + isolate handoff + edit gate` 变体示例；
  - 在 realtime event protocol + interrupt/resume contract 启用时，补齐 `realtime interrupt/resume` 变体示例并进入同一 smoke 矩阵；
  - 示例统一注入 diagnostics/tracing 标记，确保可进入 replay 与 gate；
  - 提供模式级 `README` 与迁移指引（从旧示例到模式化示例的映射）。
  - 新增 `example -> production` 迁移手册（建议 `examples/agent-modes/PLAYBOOK.md`），统一 `config/permission/observability/capacity/rollback/gates` 上线检查项；
  - 每个 `production-ish` 示例 README 必须补齐 `prod delta` 章节，明确相对 `minimal` 的生产差异、风险边界与必跑门禁。
  - 清理历史示例遗留占位：`examples/` 下既有示例中的 `TODO/TBD/FIXME/待补` 必须清零，未完项迁移到 `MATRIX.md`/`PLAYBOOK.md`/`tasks.md` 可追踪条目。
- a62 范围内固化示例清单（执行优先级）：
  - P0（优先落地）：
    - `examples/agent-modes/rag-hybrid-retrieval`：memory + MCP 双来源检索、fallback、Run/Stream 等价；
    - `examples/agent-modes/structured-output-schema-contract`：schema 校验、parser compatibility、drift replay；
    - `examples/agent-modes/skill-driven-discovery-hybrid`：`AGENTS.md|folder|hybrid` 发现顺序与映射；
    - `examples/agent-modes/mcp-governed-stdio-http`：`mcp/http|stdio` 接线、重连/failover、策略与诊断；
    - `examples/agent-modes/hitl-governed-checkpoint`：审批 checkpoint、拒绝回退、人工恢复、超时语义；
    - `examples/agent-modes/context-governed-reference-first`：`reference-first + isolate handoff + edit gate + tiering`；
    - `examples/agent-modes/sandbox-governed-toolchain`：allowlist/egress/sandbox 决策、deny/fallback；
    - `examples/agent-modes/realtime-interrupt-resume`：cursor、顺序/幂等、恢复语义；
    - `examples/agent-modes/multi-agents-collab-recovery`：协作编排、并行执行、mailbox/task-board 控制、recovery 回放一致性。
  - P1（增强补齐）：
    - `examples/agent-modes/workflow-branch-retry-failfast`：条件分支、重试、fail-fast；
    - `examples/agent-modes/mapreduce-large-batch`：大批量分片与聚合、错误分区治理；
    - `examples/agent-modes/state-session-snapshot-recovery`：unified state/session snapshot contract 导出/恢复/回放链路；
    - `examples/agent-modes/policy-budget-admission`：policy precedence + decision trace contract/runtime cost/latency budget admission contract 决策与预算准入协同；
    - `examples/agent-modes/tracing-eval-smoke`：OTel tracing + agent eval interoperability contract tracing/eval 最小闭环；
    - `examples/agent-modes/react-plan-notebook-loop`：ReAct + plan-notebook 语义、Run/Stream 等价；
    - `examples/agent-modes/hooks-middleware-extension-pipeline`：lifecycle hooks + tool middleware onion-chain 执行语义；
    - `examples/agent-modes/observability-export-bundle`：观测导出与 bundle 组装最小闭环。
  - P2（可选增强）：
    - `examples/agent-modes/adapter-onboarding-manifest-capability`：adapter manifest 激活、capability 协商与 sandbox profile-pack 接线；
    - `examples/agent-modes/security-policy-event-delivery`：security policy 决策、security event 归一化、delivery 重试/熔断链路；
    - `examples/agent-modes/config-hot-reload-rollback`：`env > file > default`、非法热更新 fail-fast 与原子回滚演示；
    - `examples/agent-modes/workflow-routing-strategy-switch`：按输入/置信度/成本/capability 的显式路由策略切换；
    - `examples/agent-modes/multi-agents-hierarchical-planner-validator`：planner-worker-validator 分层协作与回退语义；
    - `examples/agent-modes/mainline-mailbox-async-delayed-reconcile`：mailbox `sync/async/delayed/reconcile` 主干闭环与 Run/Stream 等价；
    - `examples/agent-modes/mainline-task-board-query-control`：task-board 查询/手动控制/重试预算/幂等回放；
    - `examples/agent-modes/mainline-scheduler-qos-backoff-dlq`：优先级公平性、重试退避、dead-letter 与恢复一致性；
    - `examples/agent-modes/mainline-readiness-admission-degradation`：preflight 分类、strict/non-strict 映射、admission 降级/阻断语义；
    - `examples/agent-modes/custom-adapter-mcp-model-tool-memory-pack`：自定义 adapter 四类接入（mcp/model/tool/memory）与统一 contract/gate 验收；
    - `examples/agent-modes/custom-adapter-health-readiness-circuit`：adapter health probe、circuit 状态、readiness/admission 映射与回退策略。
- a62 提案落地拆解（可直接转 `openspec/tasks.md`）：
  - 阶段 A（骨架与矩阵，`a62-T00~T05`）：
    - `a62-T00`：建立 `examples/agent-modes/` 目录与统一入口 README；
    - `a62-T01`：落地 `examples/agent-modes/MATRIX.md`（`pattern -> minimal -> production-ish -> contracts -> gates -> replay`）；
    - `a62-T02`：为每个模式创建最小目录骨架（`main.go` + `README.md` + 运行命令）；
    - `a62-T03`：新增 `check-agent-mode-pattern-coverage.sh/.ps1` 并接入 `check-quality-gate.*`；
    - `a62-T04`：扩展 `check-agent-mode-examples-smoke.sh/.ps1`，覆盖 minimal 场景最小冒烟；
    - `a62-T05`：更新 root README 与 `docs/mainline-contract-test-index.md` 示例索引映射。
  - 阶段 B（P0 模式落地，`a62-T10~T18`）：
    - `a62-T10` `rag-hybrid-retrieval`：`minimal` 覆盖 memory 检索；`production-ish` 增加 MCP 外部检索 + fallback；门禁对齐 `check-memory-contract-conformance.*`、`check-memory-scope-and-search-contract.*`；
    - `a62-T11` `structured-output-schema-contract`：`minimal` 覆盖 schema 输出；`production-ish` 增加 parser compatibility + drift fixture；门禁对齐 `check-diagnostics-replay-contract.*`；
    - `a62-T12` `skill-driven-discovery-hybrid`：`minimal` 覆盖单一 source；`production-ish` 覆盖 `AGENTS.md|folder|hybrid` 顺序与映射；门禁对齐 `check-hooks-middleware-contract.*`；
    - `a62-T13` `mcp-governed-stdio-http`：`minimal` 覆盖单传输；`production-ish` 覆盖 `stdio+http`、重连/failover、策略解析；门禁对齐 `check-multi-agent-shared-contract.*`；
    - `a62-T14` `hitl-governed-checkpoint`：`minimal` 覆盖 await/resume；`production-ish` 覆盖 reject/timeout/recover；门禁对齐 `check-react-contract.*` + 受影响 replay suites；
    - `a62-T15` `context-governed-reference-first`：`minimal` 覆盖 reference-first；`production-ish` 覆盖 isolate/edit-gate/tiering 组合；门禁对齐 `check-context-jit-organization-contract.*`；
    - `a62-T16` `sandbox-governed-toolchain`：`minimal` 覆盖 allow/deny；`production-ish` 覆盖 egress/allowlist/fallback 与 executor capability/session conformance；门禁对齐 `check-security-sandbox-contract.*`、`check-sandbox-egress-allowlist-contract.*`、`check-sandbox-rollout-governance-contract.*`、`check-sandbox-executor-conformance.*`；
    - `a62-T17` `realtime-interrupt-resume`：`minimal` 覆盖 interrupt/resume；`production-ish` 覆盖 cursor 幂等与恢复；门禁对齐 `check-realtime-protocol-contract.*`；
    - `a62-T18` `multi-agents-collab-recovery`：`minimal` 覆盖多 agent 协作链路；`production-ish` 覆盖 mailbox/task-board 控制、恢复与回放一致性；门禁对齐 `check-multi-agent-shared-contract.*`、`check-canonical-mailbox-entrypoints.*`、`check-full-chain-example-smoke.*`。
  - 阶段 C（P1/P2 增强落地，`a62-T20~T38`）：
    - `a62-T20` `workflow-branch-retry-failfast`：分支、重试、fail-fast 组合；
    - `a62-T21` `mapreduce-large-batch`：大批量分片 + 聚合 + 错误分区；
    - `a62-T22` `state-session-snapshot-recovery`：导出/恢复/回放闭环；门禁对齐 `check-state-snapshot-contract.*`；
    - `a62-T23` `policy-budget-admission`：precedence + budget 协同；门禁对齐 `check-policy-precedence-contract.*`、`check-runtime-budget-admission-contract.*`；
    - `a62-T24` `tracing-eval-smoke`：最小 tracing/eval 可观测链路；门禁对齐 `check-agent-eval-and-tracing-interop-contract.*`；
    - `a62-T25` `react-plan-notebook-loop`：ReAct 推理-行动闭环 + plan-notebook 记录/恢复语义；门禁对齐 `check-react-contract.*`、`check-react-plan-notebook-contract.*`；
    - `a62-T26` `hooks-middleware-extension-pipeline`：lifecycle hooks + tool middleware onion-chain 顺序、错误冒泡与上下文透传；门禁对齐 `check-hooks-middleware-contract.*`；
    - `a62-T27` `observability-export-bundle`：runtime recorder 导出、bundle 组装与回放漂移分类；门禁对齐 `check-observability-export-and-bundle-contract.*`、`check-diagnostics-replay-contract.*`；
    - `a62-T28` `adapter-onboarding-manifest-capability`：manifest 激活、required/optional 能力协商、profile replay 与 scaffold 漂移防护；门禁对齐 `check-adapter-manifest-contract.*`、`check-adapter-capability-contract.*`、`check-adapter-contract-replay.*`、`check-adapter-conformance.*`、`check-adapter-scaffold-drift.*`、`check-sandbox-adapter-conformance-contract.*`；
    - `a62-T29` `security-policy-event-delivery`：policy 判定、event taxonomy、callback delivery（drop_old/retry/circuit）语义；门禁对齐 `check-security-policy-contract.*`、`check-security-event-contract.*`、`check-security-delivery-contract.*`；
    - `a62-T30` `config-hot-reload-rollback`：配置优先级、热更新失败回滚与诊断回放稳定性；门禁对齐 `check-diagnostics-replay-contract.*`、`check-quality-gate.*`；
    - `a62-T31` `workflow-routing-strategy-switch`：显式路由策略切换（输入/置信度/成本/capability）与 fallback 稳定性；门禁对齐 `check-multi-agent-shared-contract.*`、`check-full-chain-example-smoke.*`；
    - `a62-T32` `multi-agents-hierarchical-planner-validator`：planner-worker-validator 分层协作、冲突回退与恢复一致性；门禁对齐 `check-multi-agent-shared-contract.*`、`check-full-chain-example-smoke.*`；
    - `a62-T33` `mainline-mailbox-async-delayed-reconcile`：mailbox `sync/async/delayed/reconcile` 语义矩阵与 recovery 回放；门禁对齐 `check-multi-agent-shared-contract.*`、`check-canonical-mailbox-entrypoints.*`、`check-full-chain-example-smoke.*`；
    - `a62-T34` `mainline-task-board-query-control`：task-board query/control、manual retry、operation-id 幂等与 replay 稳定性；门禁对齐 `check-multi-agent-shared-contract.*`；
    - `a62-T35` `mainline-scheduler-qos-backoff-dlq`：qos 公平性、backoff、dead-letter 与恢复回放；门禁对齐 `check-multi-agent-shared-contract.*`、`check-full-chain-example-smoke.*`；
    - `a62-T36` `mainline-readiness-admission-degradation`：readiness 分类、strict/non-strict、admission 降级/阻断与无副作用断言；门禁对齐 `check-policy-precedence-contract.*`、`check-quality-gate.*`；
    - `a62-T37` `custom-adapter-mcp-model-tool-memory-pack`：自定义 adapter 四类接入（mcp/model/tool/memory）最小+production-ish 双档示例；门禁对齐 `check-adapter-conformance.*`、`check-adapter-manifest-contract.*`、`check-adapter-capability-contract.*`、`check-adapter-contract-replay.*`、`check-memory-contract-conformance.*`、`check-memory-scope-and-search-contract.*`；
    - `a62-T38` `custom-adapter-health-readiness-circuit`：adapter health probe、circuit 开关、readiness finding 映射与 admission 决策等价；门禁对齐 `check-adapter-conformance.*`、`check-quality-gate.*`。
  - 阶段 D（统一收口，`a62-T90~T100`）：
    - `a62-T90`：每个模式补齐 `minimal/prod-ish` 运行说明与“边界不覆盖”声明；
    - `a62-T91`：示例统一注入 diagnostics/tracing 标记并补 replay fixture；
    - `a62-T92`：新增/更新 smoke 脚本以支持按模式子集执行（便于 CI 分片）；
    - `a62-T93`：补齐 shell/PowerShell parity 校验，required native command 非零即 fail-fast；
    - `a62-T94`：执行 `check-quality-gate.*` 与 docs consistency，全绿后再标记 a62 完成。
    - `a62-T95`：新增 `examples/agent-modes/PLAYBOOK.md`，固化 `example -> production` 迁移路径、分层风险与回滚策略。
    - `a62-T96`：为每个 `production-ish` 示例补齐 `prod delta` 检查清单（配置/权限/容量/观测/回放/门禁）并与 `MATRIX.md` 对齐。
    - `a62-T97`：新增 `check-agent-mode-migration-playbook-consistency.sh/.ps1`，校验示例索引、`MATRIX.md` 与 playbook 的映射完整性。
    - `a62-T98`：将 `migration-playbook-consistency` 接入 `check-quality-gate.*`，并产出 `missing-checklist/missing-gate` 阻断分类。
    - `a62-T99`：清理 `examples/` 历史示例中的 `TODO/TBD/FIXME/待补` 占位，并将未完项迁移至 `MATRIX.md`/`PLAYBOOK.md`/`tasks.md` 可追踪条目。
    - `a62-T100`：新增 `check-agent-mode-legacy-todo-cleanup.sh/.ps1` 并接入 `check-quality-gate.*`，阻断 TODO 类占位回流。
- Gate：
  - `check-agent-mode-examples-smoke.sh/.ps1`（按 PocketFlow+Baymax 模式矩阵执行最小冒烟）
  - `check-agent-mode-pattern-coverage.sh/.ps1`（校验模式覆盖矩阵完整性与文档索引一致性）
  - `check-agent-mode-migration-playbook-consistency.sh/.ps1`（校验 `example -> production` 迁移手册与示例索引、门禁映射一致）
  - `check-agent-mode-legacy-todo-cleanup.sh/.ps1`（校验 `examples/` 历史示例不存在 `TODO/TBD/FIXME/待补` 占位）
  - required-check 候选：`agent-mode-examples-smoke-gate`
- 依赖：sandbox egress governance + adapter allowlist contract-OTel tracing + agent eval interoperability contract 与 hooks/snapshot/plan/realtime baseline contracts 主链路冻结，并完成 Context JIT Organization（若启用）收敛，且 codebase consolidation and semantic labeling contract/a64/a69 收敛完成后实施（已开工的 a62 非 context 子项可并行，不受 a69 前置影响）。
- 一次性补齐边界（a62 内闭环）：
  - 交付易用性同域需求（模式补齐、示例索引、README 规范、smoke 矩阵、`example -> production` 迁移手册、历史示例 TODO 清理）统一在 a62 吸收，不再新增平行 example pack 提案。
- 启动条件：新增团队接入成本偏高、PoC 转生产迁移慢、或示例与 contract 漂移信号出现。

policy precedence + decision trace contract-a62 验收摘要（现状）：

- policy + memory + budget + tracing baseline contracts：已归档并稳定；具体 contract/replay/gate 口径以
  `docs/mainline-contract-test-index.md` 与 `openspec/changes/archive/INDEX.md` 为准。
- a62：进行中收口项（delivery usability + example pack），按 a64 主链路节奏分批推进。

统一验收前提（当前主线共用）：
- 配置治理：`env > file > default`，非法值 fail-fast，热更新失败原子回滚。
- 观测治理：运行态写入仅走 `RuntimeRecorder` 单写入口；QueryRuns 新增字段保持 additive。
- 回放治理：新增或变更 contract 时必须补 replay fixture 与 drift 分类。
- 门禁治理：新增或变更 contract 时必须补独立 gate（shell/PowerShell 等价）并接入 `check-quality-gate.*`。
- 兼容治理：Run/Stream 语义保持等价；未经提案声明不引入公开破坏性变更。

hooks/snapshot/plan/realtime baseline contracts 与 Context JIT Organization 验收口径（简版）：
- agent lifecycle hooks + tool middleware contract（hooks + middleware）：
  - 字段：`runtime.hooks.*`、`runtime.tool_middleware.*`、`runtime.skill.discovery.*`、`runtime.skill.preprocess.*`、`runtime.skill.bundle_mapping.*`
  - 回放：`hooks_middleware.v1`、`skill_discovery_sources.v1`（覆盖 `agents_md|folder|hybrid`）、`skill_preprocess_and_mapping.v1`
  - 门禁：`check-hooks-middleware-contract.*`
- unified state/session snapshot contract（state/session snapshot）：
  - 字段：`runtime.state.snapshot.*`、`runtime.session.state.*`
  - 回放：`state_session_snapshot.v1`
  - 门禁：`check-state-snapshot-contract.*`
  - CI 候选：`state-snapshot-contract-gate`
- react plan notebook + plan-change hook contract（react plan notebook）：
  - 字段：`runtime.react.plan_notebook.*`、`runtime.react.plan_change_hook.*`
  - 回放：`react_plan_notebook.v1`
  - 门禁：`check-react-plan-notebook-contract.*`
- realtime event protocol + interrupt/resume contract（realtime protocol）：
  - 字段：`runtime.realtime.protocol.*`、`runtime.realtime.interrupt_resume.*`
  - 回放：`realtime_event_protocol.v1`
  - 边界断言：`realtime_control_plane_absent`（禁止平台化实时网关/托管控制面）
  - 门禁：`check-realtime-protocol-contract.*`
- Context JIT Organization：
  - 字段：`runtime.context.jit.reference_first.*`、`runtime.context.jit.isolate_handoff.*`、`runtime.context.jit.edit_gate.*`、`runtime.context.jit.swap_back.*`、`runtime.context.jit.lifecycle_tiering.*`
  - 回放：`context_reference_first.v1`、`context_isolate_handoff.v1`、`context_edit_gate.v1`、`context_relevance_swapback.v1`、`context_lifecycle_tiering.v1`
  - 边界断言：`context_provider_sdk_absent`（禁止 `context/*` 直连 provider 官方 SDK）
  - 门禁：`check-context-jit-organization-contract.*`
  - CI 候选：`context-jit-organization-contract-gate`

跨提案联动收口（避免后续再开同域提案）：
- Policy precedence 冻结 `policy_decision_path` 与 `deny_source` 后，Runtime budget admission 与 tracing+eval 禁止重定义同义字段，仅允许引用。
- Memory scope/search/lifecycle 冻结后，Runtime budget admission 预算计算必须复用该口径，不再另起成本定义。
- Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Tracing+eval 的 eval 指标与 distributed 执行聚合必须复用 policy precedence、memory governance、runtime budget-admission 的 contract 输出字段，禁止引入平行观测数据面。
- Tracing+eval distributed evaluator execution 仅允许库内嵌入式执行治理，不得演进为托管评测控制面或服务化调度平面。
- Hooks/middleware 不得绕过 policy precedence 与 sandbox egress/allowlist 安全治理链路；hook/middleware 输出仅走 `RuntimeRecorder` 单写入口。
- skill discovery source 同域需求（`AGENTS.md`/目录路径/混合加载、配置校验、去重顺序、回放与门禁）优先在 agent lifecycle hooks + tool middleware contract/a62 内增量吸收，不再新增平行提案。
- `Discover/Compile` 预处理接线与 `SkillBundle -> prompt/tool whitelist` 映射同域需求统一在 agent lifecycle hooks + tool middleware contract/a62 内增量吸收，不再新增平行提案。
- State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。
- ReAct plan notebook 必须复用 ReAct loop 终止 taxonomy 与 hooks/middleware 合同，不得新增平行 ReAct 主循环。
- Realtime 事件协议必须复用 policy precedence 与 ReAct plan notebook 决策/计划解释字段，不得引入第二套 interrupt/resume 语义。
- Realtime 合同仅定义协议与嵌入式接缝，不得新增平台化实时网关或托管连接控制面。
- Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。
- Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。
- codebase consolidation and semantic labeling contract 的命名与文档整合必须复用现有契约字段，不得改写 contract 语义；并以“消除 `ca|ca2|ca3|ca4` 与 `Axx` 活动表述”为强制范围，映射集中维护于索引，不在多处重复定义。
- codebase consolidation and semantic labeling contract 文档/命名同域新增需求（临时文档治理、编号语义化、README 对齐、索引集中化、命名回流阻断）仅允许在 codebase consolidation and semantic labeling contract 内增量吸收，不再新开平行整治提案。
- 运行时 Harness 架构总览文档（`state surfaces/guides/sensors/tool mediation/entropy control`）仅允许在 codebase consolidation and semantic labeling contract 文档治理范围内增量吸收，不再新增平行文档提案。
- a64 的优化实现必须复用 policy precedence + decision trace contract-realtime event protocol + interrupt/resume contract 与 Context JIT Organization（若启用）既有契约字段与 reason taxonomy，禁止以性能优化引入语义分叉。
- Context Assembler 循环热路径同域需求（cache 回收、journal 批写、CA3 stage2 pass 优化、stage2 file 读取优化）统一在 a64-S1 内增量吸收，不再新增平行性能提案。
- Context Assembler 的生产可用合同治理（semantic 质量门槛、spill/swap-back 检索策略、冷存 retention/quota/cleanup、恢复一致性）统一在 a69 内增量吸收，与 a64-S1 性能优化边界分离，避免语义与性能改造交叉漂移。
- RuntimeRecorder/diagnostics、scheduler-file/mailbox/composer recovery、MCP 调用链、skill loader、memory filesystem 引擎的同域性能需求统一在 a64-S2~S6 内增量吸收，不再新增平行性能提案。
- Runner 循环、local dispatch、provider adapter、runtime config/policy resolve 的同域性能需求统一在 a64-S7~S9 内增量吸收，不再新增平行性能提案。
- observability dispatcher/logger/exporter 事件管线的同域性能需求统一在 a64-S10 内增量吸收，不再新增平行性能提案。
- Harness Engineering 对齐同域需求（推断型反馈闭环、realtime/handoff 可恢复状态面、snapshot 熵预算、harnessability scorecard）统一在 a64-S2/S3/S9 与横切门禁内增量吸收，不再新增平行提案。
- Harness Engineering 被忽视问题补漏（multi-agent 涌现行为、harness ROI/depth、harness 可测试性分层）统一在 a64 横切门禁与增量任务吸收，不新增平行提案。
- 门禁执行效率同域需求（影响面映射 `fast/full`、mandatory suites 完备性、gate latency 预算与回归阻断）统一在 a64 横切门禁内增量吸收，不新增平行提案。
- a64 所有子项必须通过 `semantic-stability + replay + perf-regression` 强门禁；任何 gate 漂移均按阻断处理，不得以“仅性能优化”为由豁免。
- a62 的示例字段与观测语义必须引用 react loop + tool-calling parity contract-realtime event protocol + interrupt/resume contract 与 Context JIT Organization（若启用）既有 contract 输出，禁止在 examples 侧定义平行语义。
- a62 交付易用性同域新增需求（PocketFlow `agent/workflow/rag/mapreduce/structured output/multi agents` 覆盖 + Baymax `mcp/skill/react/hitl/context/sandbox/realtime` 扩展、示例矩阵、README 规范、smoke/gate、`example -> production` 迁移手册、历史示例 TODO 清理）仅允许在 a62 内增量吸收，不再新开平行示例提案。
- Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。
- 若出现新增需求，优先以 policy precedence + decision trace contract-realtime event protocol + interrupt/resume contract 与 Context JIT Organization 的“增量任务”吸收，默认不新增 additional same-domain proposal series+ 同域提案。

状态对齐说明（2026-04-09）：
- 已归档并稳定：policy precedence + decision trace contract-realtime event protocol + interrupt/resume contract（A4-sandbox egress governance + adapter allowlist contract 归档历史见 `openspec/changes/archive/INDEX.md`）。
- 进行中：a71。
- 已归档：Context JIT Organization、a70、a64、a69、a62（详细清单见 `openspec/changes/archive/INDEX.md`）。
- 顺序约束调整：继续推进 a71 示例真实化，保持 docs/quality gate 同步收敛。

### P2：0.x 质量与治理持续收敛

执行要求：
- 所有变更继续通过质量门禁（`check-quality-gate.*`）与契约索引追踪。
- shell 与 PowerShell 门禁 required checks 维持语义等价：native command 非零即 fail-fast；仅 `govulncheck + warn` 允许告警放行。
- 继续按“小步提案 + 契约测试 + 文档同步”推进，不引入平台化控制面范围。
- 对外发布继续以 `0.x` 说明风险与兼容预期。

### P2：Examples Backlog（示例增强收敛）

说明：
- 原示例待办已收敛到本节，避免分散维护。
- 示例运行态与使用方式以 `examples/*/README.md` 为准；增强项排期以本 roadmap 为准。
- a62 启动后，本节 backlog 统一并入“agent mode example pack”任务编排，按模式矩阵优先收口：
  - PocketFlow：`agent/workflow/rag/mapreduce/structured output/multi agents`
  - Baymax：`skill-driven/mcp-governed/react/hitl/context/sandbox/realtime`

当前 backlog（按示例编号，摘要）：
- `01-chat-minimal`：实网变体与最小延迟观测。
- `02-tool-loop-basic`：工具失败重试、背压对比与 fanout 诊断。
- `03-mcp-mixed-call`：真实 stdio/http 接线与重连/failover。
- `04-streaming-interrupt`：中断 flush、delta 渲染与 cancel 稳定性。
- `05-parallel-tools-fanout`：并发度调优与串并行对比。
- `06-async-job-progress`：重试/dead-letter/取消传播与吞吐观测。
- `07-multi-agent-async-channel`：并发 worker、补偿重试与队列背压。
- `08-multi-agent-network-bridge`：JSON-RPC batch、错误码与超时重试。
- a62 迁移映射建议（旧示例 -> agent-modes）：
  - `01-chat-minimal` -> `agent`
  - `02-tool-loop-basic` -> `react agent` + `structured-output-schema-contract`（增量）
  - `03-mcp-mixed-call` -> `mcp-governed-stdio-http`
  - `04-streaming-interrupt` -> `realtime-interrupt-resume`（中断恢复部分）
  - `05-parallel-tools-fanout` -> `workflow-branch-retry-failfast`（并发分支部分）
  - `06-async-job-progress` -> `mapreduce-large-batch`
  - `07-multi-agent-async-channel` -> `multi agents` + `hitl-governed-checkpoint`（增量）
  - `08-multi-agent-network-bridge` -> `multi agents`（network bridge 变体）
  - `09-multi-agent-full-chain-reference` -> `state-session-snapshot-recovery` + `policy-budget-admission`（增量）
  - `examples/templates/mcp-adapter-template` + `examples/templates/model-adapter-template` + `examples/templates/tool-adapter-template` + `examples/templates/memory-adapter-template` -> `custom-adapter-mcp-model-tool-memory-pack`
- a62 固化新增（按优先级）：
  - P0：`rag-hybrid-retrieval`、`structured-output-schema-contract`、`skill-driven-discovery-hybrid`、`mcp-governed-stdio-http`、`hitl-governed-checkpoint`、`context-governed-reference-first`、`sandbox-governed-toolchain`、`realtime-interrupt-resume`、`multi-agents-collab-recovery`
  - P1：`workflow-branch-retry-failfast`、`mapreduce-large-batch`、`state-session-snapshot-recovery`、`policy-budget-admission`、`tracing-eval-smoke`、`react-plan-notebook-loop`、`hooks-middleware-extension-pipeline`、`observability-export-bundle`
  - P2：`adapter-onboarding-manifest-capability`、`security-policy-event-delivery`、`config-hot-reload-rollback`、`workflow-routing-strategy-switch`、`multi-agents-hierarchical-planner-validator`、`mainline-mailbox-async-delayed-reconcile`、`mainline-task-board-query-control`、`mainline-scheduler-qos-backoff-dlq`、`mainline-readiness-admission-degradation`、`custom-adapter-mcp-model-tool-memory-pack`、`custom-adapter-health-readiness-circuit`

## 维护提示（状态快照更新）

每次归档或切换活跃 change 后，维护者应同步执行以下最小流程，避免触发 release status parity governance 口径漂移阻断：

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




