# Development Roadmap

更新时间：2026-09-18

## 定位

Baymax 主线保持 `library-first + contract-first`：

- 交付可嵌入的 Go runtime，不建设平台化控制面。
- 以 OpenSpec、契约测试、回放和质量门禁驱动行为变更。
- 代码、测试、文档与 spec delta 在同一变更中收敛。

本文件是运行态决策入口，不重复归档提案的设计和任务细节。历史范围、设计理由和完整任务以 `openspec/changes/archive/` 为准。

## 当前状态

状态权威来源：

1. 活跃变更：`openspec list --json`。
2. 已归档变更：`openspec/changes/archive/INDEX.md`。
3. 示例交付状态：`examples/agent-modes/MATRIX.md`。

截至 2026-09-20：

- 已归档：
  - `introduce-runtime-steering-and-follow-up-input-contract`
  - `standardize-runtime-failure-taxonomy-and-terminal-outcome-contract`
  - `harden-runtime-event-stream-terminal-recovery-contract`
  - `harden-tool-lifecycle-and-failure-isolation-contract`
  - `establish-session-history-checkpoint-replay-contract`
  - `context-compression-runtime-handoff-contract`
  - `extension-lifecycle-governance-resource-resolution-contract`
  - `introduce-provider-model-capability-and-credential-preflight-contract`
  - `establish-embedded-host-command-response-and-event-correlation-contract`
  - `harden-cross-provider-handoff-and-stream-edge-conformance`（跨 Provider handoff、stream edge、fallback fence、Run/Stream parity 与 replay/gate conformance）
  - `harden-durable-attempt-workspace-binding-and-completion-safepoint`（task/attempt/workspace binding、lease/retry/recovery reconciliation 与 completion safe-point ownership；fixture、replay、contract、gate 和文档已收口）
  - `establish-eval-continuity-comparison-and-replay-contract`（bounded reference-only continuity projection、跨 compaction/handoff/snapshot/recovery 的确定性比较，以及 fixture/replay/contract/gate 已收口）
  - `external-extension-authoring-conformance`（离线 authoring conformance、版本化 fixture、replay 与双平台 gate 已收口；不新增运行时配置或 agent-mode 示例语义）
  - `establish-eval-first-error-attribution-and-trajectory-boundary-contract`（归档 141；bounded 首错归因/比较、轨迹前缀决策边界、Badcase/experiment/feedback additive 关联、memory application corpus fixture、replay 与 gate 已收口；不修改 runtime loop 或示例语义）
  - `establish-budget-aware-derived-context-projection-contract`（归档 143；有界只读预算投影契约、离线 benchmark、fixture、replay 与 gate 已收口；未改变运行时行为）
  - `establish-model-catalog-routing-admission-audit-contract`（归档 145；host-supplied model catalog/routing admission 的有界审计、版本化 fixture/replay、Run/Stream parity 及由可复现 exact-identity gap 触发的纯函数 deterministic resolver 已收口；不引入远程 catalog、credential store 或全局 router）
  已归档提案的后续修复必须以新的 OpenSpec change 从最新 `master` 切出。
- 候选：
  - 观察候选：`Model catalog 与本地模型路由增量`。该方向需满足自身触发条件，不因外部项目存在同名能力而自动立项。
  - `Provider 结构化上下文投影与 Prompt Cache 可观测性` 的审计基线已由归档 142 收口；native request projection 修复已作为归档 144 `preserve-provider-native-request-projection-parity` 交付，cache usage schema 仍须等待独立的成本/P95 触发证据。
  - `预算感知派生上下文投影` 的启动条件（先建立 benchmark、fixture 与 replay，比较完成率、预算利用、重复循环与恢复重算）已由归档 143（`establish-budget-aware-derived-context-projection-contract`）交付并收口，不再作为观察候选。
- 已归档：
  - `preserve-provider-native-request-projection-parity`（归档 144；以归档 142 已固定的 role/tool-result-native/ordering drift 为证据，修复三家 adapter 的 provider-native SDK 请求投影及 Run/Stream/CountTokens 对等；不增加 cache schema、共享 wire protocol 或 runtime 配置）。

`harden-cross-provider-handoff-and-stream-edge-conformance`、`establish-eval-continuity-comparison-and-replay-contract` 与 `external-extension-authoring-conformance` 已归档并纳入主线基线；其既有 fixture、replay 与 gate 不再作为新候选重复排期。后续只能在新的可复现 drift、真实宿主需求或稳定成本/质量瓶颈下，以既有 owner 的增量 change 处理。

最近归档的变更完成了运行终态、事件恢复、工具失败隔离、会话/回放、上下文交接、扩展治理、provider/model 准入以及嵌入式宿主命令/事件关联的主线收口。较早的已完成能力请直接查阅 [Archive Index](../openspec/changes/archive/INDEX.md)。

归档 142（`establish-provider-request-projection-and-cache-observability-contract`）范围限定为「Runtime → Provider」方向：建立版本化 `provider_request_projection.v1`、`source`/`observed` 双投影与 canonical digest、把已确认的 role / tool-result-native / 能力投影语义丢失写成被钉住的 `declared_gap`、给出 cache 用量 `additive + nullable + default` 兼容口径，并交付离线 replay、版本化 fixture 与双平台 gate。归档 142 本身**未修改适配器运行时投影行为**；归档 144 已在其证据基线上显式修复已验证的 role、ordering 与 native tool-result gap，不新增 provider、配置键、credential store、远程 catalog 或诊断落盘字段。

该提案留下的后续增量触发证据已由归档 144 处理：三个适配器的 Generate/Stream 现在复用 SDK-neutral canonical facts 构造原生 role/tool-result request，Anthropic/Gemini CountTokens 复用相同可表达事实，OpenAI 继续保留官方 SDK 无 token-count API 的明确 capability exception。Prompt-cache usage schema 仍不在归档 144 内，只有稳定成本/P95 与 provider usage 证据成立时才另行立项。

## 版本阶段口径（延续 0.x）

当前仓库不做 `1.0.0` / prod-ready 承诺，继续沿用 `0.x` 治理口径（见 `docs/versioning-and-compatibility.md`）。在 `0.x` 阶段，版本号用于表达变更范围，不构成稳定兼容承诺；主线目标是持续收敛、可回归迭代。

`0.x` 阶段允许新增能力型提案，但新增能力必须有明确的宿主价值、可验证的 contract 边界和回滚路径。

## 已交付基线

以下能力已稳定并由归档提案、主干测试和门禁覆盖。后续需求必须作为增量扩展，复用既有术语和 source-of-truth，不得重新定义平行语义。

| 域 | 已交付基线 | 后续约束 |
| --- | --- | --- |
| Runtime 与 Protocol | Run/Stream、Agent Runtime Protocol、capability/context/admission、ReAct plan notebook、checkpoint/workspace provenance、权威终态 | Run/Stream 保持语义等价；执行、计划、恢复和排队仍由各自 source runtime owner 拥有。 |
| Realtime、Host 与 HITL | interrupt/resume、durable stream binding、cursor、catch-up/live-tail、terminal recovery、embedded host command/event correlation、HITL reverse request、steering/follow-up | 不创建第二套 event ordering、cursor、pending、输入队列或终态状态机。 |
| Tool、MCP 与 Security | 本地工具生命周期、MCP profiles、sandbox isolation/egress、allowlist、policy precedence | 工具和扩展不得绕过 policy、sandbox 或 `RuntimeRecorder`。 |
| Context 与 Memory | Context Assembler、reference-first、task-aware tail recap、压缩生产治理、压缩 handoff、memory SPI、scope/search/lifecycle | `context/*` 不直连 provider SDK；recap/status 只能是 source-owned facts 的有界派生投影，snapshot 保持唯一事实源。 |
| Orchestration | Workflow、Teams、A2A、Scheduler、Mailbox、task board、recovery | 不以新 change 建立平台化调度或统一多代理拓扑。 |
| Config、Readiness 与 Diagnostics | `env > file > default`、fail-fast、热更新原子回滚、readiness/admission、diagnostics replay | QueryRuns 和诊断 schema 仅 additive + nullable + default。 |
| Extension 与 Provider | extension lifecycle/resource resolution、adapter manifest/capability、静态 provider/model catalog、credential preflight | Provider 细节在 `model/<provider>`；不引入远程目录、credential store 或扩展市场。 |
| Evaluation 与 Gates | OTel/eval、corpus/Badcase/experiment、review-only feedback、continuity comparison、semantic/performance/docs/contract gates | 评测和观测不演化为托管控制面；反馈不得自动修改 prompt、Skill、tool、policy、memory、runtime 配置或 gate。 |

State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。

### Harnessability scorecard 与门禁耗时预算治理

A64 的 harnessability scorecard 用于衡量契约覆盖、回放漂移、门禁接线、文档一致性和验证开销；它是门禁可审计性的计算型指标，不改变运行时语义。评估遵循 **harness ROI/depth** 分层，并坚持 **computational-first, inferential-second**：客观测试与结构化证据先于主观评估结论。门禁执行同时受门禁耗时预算治理约束，超出预算时记录并按既有回滚/降级路径处理，不自动放宽质量阈值。

完整的 proposal、design、spec、fixture 和 gate 映射见：

- `openspec/changes/archive/INDEX.md`
- `docs/mainline-contract-test-index.md`
- `docs/runtime-config-diagnostics.md`
- `docs/runtime-module-boundaries.md`
- `docs/pi-agent-comparison-and-adoption-study.md`（外部项目对照方法与后续提案筛选依据）

## 可启动候选

候选不是承诺排期。启动前必须先完成现状审计，并在 OpenSpec proposal 中记录 `Why now`、风险、回滚点、文档影响、Example Impact Assessment 和验证命令。

### 已归档基线：Durable task-attempt/workspace binding 与 completion safe-point 所有权审计

**交付状态**：已于 2026-09-13 归档为 `138-harden-durable-attempt-workspace-binding-and-completion-safepoint`，并纳入主线基线。

**触发信号**：checkpoint/snapshot 已具备 workspace provenance 和完整性漂移检测，但 scheduler 的 `Task`/`Attempt`/lease rollover 没有结构化 workspace binding；后台 completion 已分别存在于 mailbox/scheduler 与 runtime-input safe point，却缺少一条被证明的统一 promotion ownership 接缝。该方向已通过 gap fixture、replay、contract 和 gate 收口。

**目标**：审计并验证 task/attempt 与 workspace provenance 的关联、retry/lease rollover 隔离、snapshot/recovery integrity reconciliation，以及后台 completion 进入下一模型决策安全点时的 correlation、dedupe、late、disconnect、recovery 和 Run/Stream parity 语义。

**必须复用**：scheduler `Task`/`Attempt` 与 lease、checkpoint/workspace provenance、snapshot/recovery、mailbox、source-owned runtime input、`RuntimeRecorder`、现有 terminal outcome 与 A2A correlation。

**第一阶段交付**：只新增 bounded gap fixture、replay 分类、必要的 contract/gate 测试和审计文档；只有 fixture 证明 lease rollover、retry 或 recovery 会误用 workspace，或 completion 会丢失/重复 promotion，才进入最小运行时字段/API 变更。

**Ownership / compatibility / privacy**：scheduler 继续拥有 task/attempt/lease/retry/terminal commit；checkpoint/snapshot 只拥有 reference-only provenance 与恢复前校验；mailbox 拥有 durable completion delivery；`core/runner` 拥有 runtime-input safe-point admission/application；`tool/diagnosticsreplay` 只做离线、无副作用归一化。新增引用必须 additive + nullable + default，历史记录缺失时按 binding absent 处理，未知字段安全忽略；workspace 内容、路径、Git 元数据、completion body、reasoning、credentials 和无界 payload 不进入快照、诊断或 replay。

**No-new-config / rollback**：本审计不新增 runtime 配置键或 hot-update 分支，继续复用现有 `env > file > default` 配置域。回滚只移除新增 reference projection、fixture/replay/gate 与恢复适配，不需要持久化迁移，也不改变既有 scheduler、mailbox、snapshot、Run/Stream 或 terminal outcome 语义。

**明确不做**：Git/worktree manager、runtime 直接执行 Git/shell、hosted workspace/artifact service、自动 merge/push、第二套 task/session/coordination 状态机，或复制外部项目的 daemon thread、`shell=True`、固定轮询和非事务 worktree index。

**Example Impact Assessment（立项时）**：`无需示例变更（附理由）`。首阶段只验证 scheduler、recovery、mailbox、runtime-input 接缝，不修改 `examples/agent-modes`；若后续确需示例变化，必须先完成 `MATRIX.md` 与对应模式 README 的文档基线。

### 已归档基线：跨 Provider handoff 与 stream edge conformance

**触发信号**：多 Provider、fallback、context handoff 与 tool-result feedback 已成为主线能力，但 OpenAI、Anthropic、Gemini 的 tool-call/thinking/usage/abort/Unicode/空内容边界仍缺少一份统一、可回放、可阻断的 conformance matrix。随着 embedded host 与运行中输入合同落地，这些边界漂移会直接暴露给宿主，具备现在收口的风险信号。

**目标**：基于现有 provider adapter、capability/preflight、context handoff、tool-result feedback、failure taxonomy 与 Run/Stream parity，建立跨 Provider 的确定性 handoff 和 stream edge conformance 合同。优先补齐测试、版本化 fixture、replay 与 gate；只有 fixture 证明漂移时才最小化修正 `model/<provider>`。

**必须复用**：`model/<provider>` owner、Provider capability/preflight、`model/toolcontract`、context handoff、failure taxonomy、authoritative terminal outcome、Run/Stream parity、diagnostics replay 与 `RuntimeRecorder`。

**验证方向**：tool-call/tool-result round-trip、thinking/reasoning 投影、stream start/partial/abort、usage 保留、overflow、Unicode、空内容、fallback/handoff causation、错误分层、终态唯一性、Run/Stream parity、replay idempotency 与 shell/PowerShell gate parity。

**明确不做**：新增 Provider、重写 Provider SDK、远程 model catalog、credential store、通用 provider-agnostic wire protocol、全局路由器、托管 gateway，或在 `context/*` 引入 Provider 官方 SDK。

**Example Impact Assessment（立项时）**：`修改示例`。若修改 `examples/agent-modes`，必须先更新 `MATRIX.md` 与对应模式 README 的 semantic anchor、runtime path、expected markers 和 rollback notes；示例不得依赖 live provider。

已归档的嵌入式宿主接缝、HITL reverse request 与 steering/follow-up 合同作为本候选的宿主侧前置基线，不再以新提案重复定义。完整对照、证据基线和不吸收项见 `docs/pi-agent-comparison-and-adoption-study.md`。

### 已归档基线：Eval 首错归因、轨迹前缀决策边界与证据化改进建议

**交付状态**：归档 141 已在既有 eval/corpus/Badcase/experiment/feedback owner 内交付 `first error step/kind`、root-cause owner、primary/secondary cause、bounded evidence refs、recoverability/confidence、轨迹前缀 acceptable/forbidden actions 与 review-only recommendation，并覆盖 memory retrieval/application 的同一 corpus 场景；未新建 transcript、artifact 或 memory 事实源，未改 runtime loop。

**后续触发**：只有新的 Badcase 无法由既有首错归因、evidence 或 action-boundary taxonomy 稳定解释时，才以新的增量 change 扩展既有 owner；建议继续保持 review-only，不得自动修改 prompt、Skill、tool、policy、memory、runtime 配置、测试、gate 或代码。

### 已归档：Provider 原生请求投影与 Run/Stream 对等

**交付状态**：归档 144（`preserve-provider-native-request-projection-parity`）以归档 142 的真实 SDK fixture/gap 为依据，修复 `ModelRequest.Input`、`Messages`、`ToolResult` 到 SDK request 的原生 role、顺序、tool-result correlation 与 Run/Stream/CountTokens parity。跨 Provider handoff 与 stream edge conformance 继续复用既有 owner，不建设共享协议。

**范围与后续触发**：归档 144 只最小修改 `model/<provider>` 与 SDK-neutral canonical request facts；prompt-cache usage 仍不纳入。只有 cache 成本/P95 成为稳定瓶颈且有 provider usage 证据时，才另行以 `additive + nullable + default` 和 `RuntimeRecorder` 单写入口设计 cache schema。

**必须复用**：`model/<provider>`、`model/toolcontract`、context handoff、既有 provider conformance、Run/Stream parity 与 `RuntimeRecorder`。不得新建通用 provider wire protocol、gateway、credential store 或全局路由状态机，不得在 `context/*` 引入 Provider 官方 SDK。

**Example Impact Assessment**：`无需示例变更（附理由）`。归档 144 使用 fixture/replay/gate 验证 SDK 请求构造，不改变示例 runtime path；若后续实现改变示例的 expected markers 或 usage 输出，必须先更新 `MATRIX.md` 与对应 README，并将评估改为“修改示例”。

### 已归档：预算感知派生上下文投影

**交付状态**：归档 143（`establish-budget-aware-derived-context-projection-contract`）已完成有界只读投影契约、离线预算利用 benchmark、版本化 fixture、replay 与双平台 gate；未修改运行时行为。

**校准事实**：Baymax 已有 ReAct iteration/tool-call limit、`runtime.react.on_budget_exhausted`、runtime budget admission、scheduler `ParentRemainingBudget`、ReAct Plan Notebook 与 task-aware tail recap。缺口不是预算 source-of-truth 或新的任务状态机，而是模型决策上下文尚缺少由这些 owner 计算出的、有界只读 remaining-budget 投影，也缺少"预算增加是否改善完成质量、还是只增加重复循环"的稳定评测。

**目标**：在既有 owner 之上增量交付版本化、有界、只读的派生预算投影契约 `budget_projection.v1`，以及离线、确定性、计算型的预算利用 benchmark；比较 completion rate、iteration/tool-call usage、token/latency、repeated-loop rate、Run/Stream parity，以及 snapshot/recovery 后预算投影的确定性重算。只有数据证明模型可见预算事实能稳定改善决策质量，才讨论策略提示或 tail projection 的最小 contract；不得先引入 adaptive budget 策略。

**必须复用**：LoopPolicy/ReAct counters（`RuntimeReactConfig`）、budget admission（`budget_admission.v1`）、scheduler `ParentRemainingBudget`、Plan Notebook、`context/assembler` 既有的 "source-owned facts → bounded 派生投影" 模式（`buildTaskAwareTailRecap`）与 `tool/diagnosticsreplay`。

**所有权边界**：预算事实继续由 LoopPolicy/ReAct counters、budget admission 和 scheduler parent budget 各自拥有；Plan Notebook 继续只拥有 plan lifecycle；`context/budgetprojection` 只接收 `source-owned facts -> bounded/read-only/nullable derived projection`。不得建立第二预算账本、第二 task/plan/terminal 状态机，也不得把模型生成的剩余预算写回事实源。

**Example Impact Assessment（首阶段）**：`无需示例变更（附理由）`。首阶段仅增加 benchmark/fixture/replay/gate，不改变 agent-mode 配置、runtime path 或 expected markers；若后续投影进入示例上下文，必须重新评估并遵循文档先行规则。

## 后续提案备选池（外部项目对照增量）

备选池用于记录可验证方向，不代表承诺排期。候选必须由真实宿主需求、可复现风险或稳定成本/质量瓶颈触发；没有触发证据时保持观察。提案启动后，其状态只进入“当前状态”，不在本表维护第二份进度。本轮外部研究参考 `bojieli/ai-agent-book@c8963443736004412692b1af7706c89096d46e4e`，只吸收能够路由到 Baymax 既有 owner、可由 fixture/replay/gate 验证且符合 library-first 边界的最小子集。

### 同域演进路由

- **Eval**：continuity comparison 已归档；下一质量方向进入既有 corpus/Badcase/experiment/feedback owner，增量验证首错归因、轨迹前缀决策边界和 evidence-linked review-only suggestion。Memory application 作为 corpus 场景并入，不单独立项。
- **Provider**：结构化上下文与 prompt cache 进入归档 137 的增量 owner，先验证真实 SDK fixture；只有 drift 成立才最小修改 `model/<provider>`，不抽象新的共享 wire protocol。
- **Context/Budget**：预算可见性复用 ReAct counters、budget admission、scheduler parent budget、Plan Notebook 与 task-aware tail recap，只允许 source-owned facts 的有界只读派生投影，不创建平行账本或状态机。
- **Tool discovery**：按需 tool schema 投影复用现有 Skill loader、MCP、manifest/capability、allowlist 与 sandbox；只有工具规模、schema token 或选择准确率成为稳定瓶颈时才启动，不建设市场、动态下载或 credential store。
- **Realtime/Host**：事件、中断恢复、HITL、steering/follow-up、completion promotion 与 request correlation 继续由既有 owner 演进，不抽取第二套 event ordering、pending、输入队列、协调或终态控制面。

历史门禁继续保留下列同域收口锚点；后续结构调整不得改变其 owner 语义：

- Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。
- Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。
- Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。
- Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。

| 成熟度 | 候选方向 | 可吸收点 | 必须复用的 Baymax owner | 触发信号与首要边界 |
| --- | --- | --- | --- | --- |
| 已归档基线（137） | 跨 Provider handoff 与 stream edge conformance | 跨 provider tool-call/thinking 转换、abort usage、overflow、Unicode/空内容 fixture | `model/<provider>`、provider admission、context handoff、failure taxonomy、terminal outcome | 已完成并归档；后续仅在新的可复现 drift 下以增量 change 处理，不重新排期为 P2。 |
| 已归档基线（138） | Durable task-attempt/workspace binding 与 completion safe-point 所有权审计 | 已完成 task/attempt 与 workspace provenance 关联、attempt/lease rollover 隔离、missing/dirty/conflict/drift 分类、恢复 reconciliation，以及后台 completion 的 correlation、dedupe、late/disconnect/recovery 与 Run/Stream parity 验证 | scheduler `Task/Attempt` 与 lease、checkpoint/workspace provenance、snapshot/recovery、mailbox、source-owned runtime input、`RuntimeRecorder` | 已通过 gap fixture、contract/replay、shell/PowerShell gate 和文档一致性校验；后续仅在新的可复现 drift 下以增量 change 处理。实际 workspace/Git 生命周期仍由 host/tool adapter 拥有，不新增 worktree manager、通知队列或任务状态机。 |
| 已归档基线（139） | Eval continuity comparison 与 replay contract | bounded reference-only continuity projection、跨 compaction/handoff/snapshot/recovery 的确定性比较 | OTel/eval/corpus、context handoff、checkpoint/snapshot refs、diagnostics replay | 已完成并归档；后续质量增量进入首错归因与轨迹边界候选，不重复建设 transcript/artifact service。 |
| 已归档基线（140） | External extension authoring conformance | 离线 authoring conformance、版本化 fixture、replay 与双平台 gate | extension lifecycle/resource resolution、manifest/capability、allowlist、sandbox | 已完成并归档；未来仅在新的真实扩展来源暴露新增 drift 时，以既有 owner 的增量 change 处理。 |
| 观察候选 | Provider 结构化上下文投影与 Prompt Cache 可观测性 | 真实 SDK message/tool-result projection、stable prefix/tool order、cache usage 与 Run/Stream parity | `model/<provider>`、`model/toolcontract`、context handoff、provider conformance、`RuntimeRecorder` | fixture 证明 role/message/tool-result 语义丢失或顺序漂移，或 cache 成本/P95 成为稳定瓶颈。cache 字段仅 additive + nullable + default。 |
| 条件候选 | 本地/宿主准入后的按需 Tool Schema 投影 | 在已准入工具集合内按需选择并投影 schema，减少无效上下文占用 | Skill loader、MCP、manifest/capability、allowlist、sandbox、既有 tool lifecycle | 工具数量、schema token 或工具选择准确率形成稳定瓶颈。不得动态下载工具、绕过准入，或建设 marketplace/credential store。 |
| 观察候选 | Model catalog 与本地模型路由增量 | runtime model discovery、本地模型 router、明确 auth preflight | provider/model catalog、credential preflight、readiness、host injection | 静态或宿主注入 catalog 无法满足明确路由需求。不引入 credential store，不在 `context/*` 引入 provider SDK。 |

备选池合并与排序规则：

1. **Eval 首错归因与轨迹边界优先审计**：它可以 fixture-only 起步、运行时风险最低，并为 Provider、Budget、Memory application 等后续方向提供统一的质量归因证据；在 gap 被证明前不改 runtime。
2. **Provider 结构化上下文与 cache 可观测性次序跟进**：真实 SDK fixture/replay/benchmark 先行，只在 role、tool-result、ordering、usage 或 Run/Stream drift 成立时最小修复既有 adapter。
3. **预算感知投影归档后仍坚持先评测后策略**：归档 143 仅交付 benchmark、fixture、replay 与 gate；没有稳定收益不得引入模型提示、adaptive budget 或新的状态持久化。
4. **Tool schema-on-demand 只由规模瓶颈触发**：保持本地/宿主准入、安全上界和确定性选择，工具规模、schema token 或准确率没有形成稳定瓶颈时不立项。
5. **Memory application 与 evidence-linked suggestion 并入 Eval**：memory retrieval/application 作为 corpus 场景，建议保持 review-only；不拆分独立反馈控制面，也不允许自动自改代码、测试或 gate。
6. **既有 Realtime/Host/Durable/Pi 边界继续有效**：Run control、HITL、steering/follow-up、completion promotion、workspace binding 已由既有 owner 收口；Pi lane/register/ledger、experimental CBOR、remote Session Server、attachment/lease、SQLite hosted backend 保持长期延后，不复制平行协调或托管状态机。

### 需求触发观察项

以下方向不预设提案顺序，只有出现相应信号才进入候选审计：

| 方向 | 触发信号 | 首要边界 |
| --- | --- | --- |
| Eval 首错归因与轨迹前缀决策边界 | Badcase 只能给出最终失败，无法定位首个偏离、primary/secondary cause、责任 owner 或下一步 acceptable/forbidden action | 复用既有 eval/continuity owner；只记录 bounded evidence refs，不保存 raw reasoning，改进建议保持 review-only。 |
| Provider 结构化上下文与 Prompt Cache | 真实 SDK fixture 证明 message role、Skill/tail fragment、tool-result correlation 或 stable ordering 丢失；或 cache 成本/P95 成为稳定瓶颈 | 复用 `model/<provider>`、`model/toolcontract` 和归档 137；不建共享 wire/gateway/credential store，诊断字段仅 additive + nullable + default。 |
| 本地/宿主准入后的按需 Tool Schema 投影 | 工具数量、schema token 占用或工具选择准确率达到稳定瓶颈 | 只在已准入工具集合内投影；复用 Skill loader/MCP/manifest/allowlist/sandbox，不建设市场、下载器或 credential store。 |
| 远程 model catalog 或本地模型路由 | 静态/宿主注入目录不足以支持明确的路由需求 | 以本页“Model catalog 与本地模型路由增量”为同一候选，不创建平行提案；不在 `context/*` 引入 provider SDK，不接入 credential store。 |
| 外部 extension 生态增量 | 归档 140 之后出现新的真实扩展来源，且供应链审计、失败反馈或隔离需求超出现有 lifecycle contract | 只做 existing authoring conformance owner 的增量；复用 manifest/capability/allowlist/sandbox，不建设 package manager、marketplace 或动态下载执行链。 |

## 示例状态

`examples/agent-modes/MATRIX.md` 是 agent-mode 示例范围、文档基线、实现状态、gate 和 replay 映射的唯一来源。旧的 numbered examples backlog 已完成迁移，不在本文件重复维护。

示例变更必须遵循：

- `examples/agent-modes` 先完成 `MATRIX.md` 与对应模式 README 的文档基线，再进入代码实现。
- 每项模式必须列出 semantic anchor、runtime path、expected markers、rollback notes、contracts、gates 和 replay。
- 示例只演示既有主线 contract，不得在 examples 中定义平行的配置、终态或观测语义。

## 新增提案准入规则（0.x 阶段）

从本文件生效起，`0.x` 阶段新增提案必须满足以下要求：

1. 直接服务于至少一类目标：
   - 契约一致性（Run/Stream、reason taxonomy、错误分层、兼容语义）。
   - 可靠性与安全（fail-fast、回滚、幂等、恢复边界、安全治理）。
   - 质量门禁回归治理（contract/perf/docs gate regression）。
   - 外部接入 DX（模板、迁移、脚手架、conformance）且可被 gate 验证。
2. 保持 library-first 边界，不引入平台化控制面能力。
3. 在 proposal、design 与 tasks 中显式说明：
   - `Why now`
   - 风险与非目标
   - 回滚点
   - 文档影响
   - Example Impact Assessment：`新增示例`、`修改示例`，或 `无需示例变更（附理由）`
   - 验证命令
4. 对行为、配置、诊断 schema 或 contract 的变更，必须同步 OpenSpec spec delta、正负边界测试、必要 integration、replay fixture 与 shell/PowerShell parity gate。
5. 非法配置和非法热更新必须 fail-fast，并原子回滚；新增诊断字段必须遵守 additive + nullable + default。

## 长期方向（不进入近期主线）

以下方向明确延后，除非有独立产品决策和拆分后的 OpenSpec：

- 平台化控制面（多租户、RBAC、审计和运营面板）。
- 跨租户全局调度与控制平面。
- 市场化/托管化 adapter registry 能力。
- Remote Runtime Gateway、托管 Session/Artifact/Workspace persistence、独立 session server、远程 event store 与 hosted worktree manager。
- Pi experimental protocol 的 CBOR wire、session attachment/lease 与 SQLite hosted backend。

这些方向不能以单个大提案混合交付；未来若启动，必须分别定义 gateway、persistence profile、artifact resolver 与 authorization/governance profile。

## 通用架构约束

- `runtime/*` 禁止依赖 `mcp/http` 或 `mcp/stdio`。
- 非 `mcp/*` 包禁止依赖 `mcp/internal/*`。
- `context/*` 禁止直接引入 provider 官方 SDK。
- Provider 协议和模型适配细节必须落在 `model/<provider>`。
- 诊断写入必须走 `observability/event.RuntimeRecorder` 单写入口。
- 配置优先级固定为 `env > file > default`。
- QueryRuns/诊断字段变更保持 additive + nullable + default。
- Run/Stream 对等场景不得引入平行终止或决策语义。

## 执行与归档

单变更优先；并行变更必须在 proposal 中声明依赖和不重叠边界。

标准顺序：

1. `openspec list --json` 与现状审计。
2. `proposal.md`、`design.md`、`tasks.md`、spec delta，包含 Example Impact Assessment。
3. 代码、测试、回放、独立 gate 与文档同步实施。
4. 执行门禁：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
```

5. 使用归档脚本完成重命名和归档，禁止手工移动目录：

```powershell
pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "<change-name>"
```

6. 归档后同步更新本文件的“当前状态”、README 里程碑快照、`docs/mainline-contract-test-index.md`（如 gate 映射变化）与 `openspec/changes/archive/INDEX.md`。

## 维护检查表

每次归档或切换 active change 后：

1. 运行 `openspec list --json`，以输出更新“进行中”条目。
2. 以 archive index 更新“已归档”条目；本文件只列最近一组，完整列表不复制。
3. 检查候选是否仍有未满足的触发信号；无触发信号的候选保持观察状态。
4. 涉及示例时只更新 `examples/agent-modes/MATRIX.md` 及对应 README，不恢复 numbered backlog。
5. 执行 `pwsh -File scripts/check-docs-consistency.ps1`，确保不存在 `roadmap-status-drift`。
