# Development Roadmap

更新时间：2026-09-13

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

截至 2026-09-12：

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
- 进行中：
  - `harden-durable-attempt-workspace-binding-and-completion-safepoint`（P1 审计提案：task/attempt/workspace binding、lease/retry/recovery 接缝与 completion safe-point ownership；首阶段以 gap fixture、replay 和 gate 证明真实漂移）
- 候选：
  - 当前没有默认启动的 P0/P1 change。新 change 必须满足本文件的准入规则，并由明确的风险信号或宿主需求触发。

`harden-cross-provider-handoff-and-stream-edge-conformance` 已于 2026-09-13 归档并纳入主线基线；其 conformance fixture、replay 与 gate 不再作为当前 P2 候选重复排期。

最近归档的变更完成了运行终态、事件恢复、工具失败隔离、会话/回放、上下文交接、扩展治理、provider/model 准入以及嵌入式宿主命令/事件关联的主线收口。较早的已完成能力请直接查阅 [Archive Index](../openspec/changes/archive/INDEX.md)。

## 版本阶段口径（延续 0.x）

当前仓库不做 `1.0.0` / prod-ready 承诺，继续沿用 `0.x` 治理口径（见 `docs/versioning-and-compatibility.md`）。在 `0.x` 阶段，版本号用于表达变更范围，不构成稳定兼容承诺；主线目标是持续收敛、可回归迭代。

`0.x` 阶段允许新增能力型提案，但新增能力必须有明确的宿主价值、可验证的 contract 边界和回滚路径。

## 已交付基线

以下能力已稳定并由归档提案、主干测试和门禁覆盖。后续需求必须作为增量扩展，复用既有术语和 source-of-truth，不得重新定义平行语义。

| 域 | 已交付基线 | 后续约束 |
| --- | --- | --- |
| Runtime 与 Protocol | Run/Stream、Agent Runtime Protocol、capability/context/admission、checkpoint/workspace provenance、权威终态 | Run/Stream 保持语义等价；执行、恢复和排队仍由 source runtime 拥有。 |
| Realtime、Host 与 HITL | interrupt/resume、durable stream binding、cursor、catch-up/live-tail、terminal recovery、embedded host command/event correlation、HITL reverse request、steering/follow-up | 不创建第二套 event ordering、cursor、pending、输入队列或终态状态机。 |
| Tool、MCP 与 Security | 本地工具生命周期、MCP profiles、sandbox isolation/egress、allowlist、policy precedence | 工具和扩展不得绕过 policy、sandbox 或 `RuntimeRecorder`。 |
| Context 与 Memory | Context Assembler、压缩生产治理、压缩 handoff、memory SPI、scope/search/lifecycle | `context/*` 不直连 provider SDK；snapshot 保持唯一事实源。 |
| Orchestration | Workflow、Teams、A2A、Scheduler、Mailbox、task board、recovery | 不以新 change 建立平台化调度或统一多代理拓扑。 |
| Config、Readiness 与 Diagnostics | `env > file > default`、fail-fast、热更新原子回滚、readiness/admission、diagnostics replay | QueryRuns 和诊断 schema 仅 additive + nullable + default。 |
| Extension 与 Provider | extension lifecycle/resource resolution、adapter manifest/capability、静态 provider/model catalog、credential preflight | Provider 细节在 `model/<provider>`；不引入远程目录、credential store 或扩展市场。 |
| Evaluation 与 Gates | OTel/eval、corpus/badcase/experiment、semantic/performance/docs/contract gates | 评测和观测不演化为托管控制面。 |

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

### P1：Durable task-attempt/workspace binding 与 completion safe-point 所有权审计

**触发信号**：checkpoint/snapshot 已具备 workspace provenance 和完整性漂移检测，但 scheduler 的 `Task`/`Attempt`/lease rollover 没有结构化 workspace binding；后台 completion 已分别存在于 mailbox/scheduler 与 runtime-input safe point，却缺少一条被证明的统一 promotion ownership 接缝。该方向先以 gap fixture 验证真实冲突，再决定是否引入最小 contract 增量。

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

## 后续提案备选池（外部项目对照增量）

同域收口声明：Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。

备选池用于记录可验证方向，不代表承诺排期。候选必须由真实宿主需求或可复现风险触发；没有触发证据时保持观察。提案启动后，其状态只进入“当前状态”，不在本表维护第二份进度。外部项目的同名能力必须先路由到本表已有 owner：`learn-claude-code` 的 context compact/identity reinjection 进入 Eval 与 context continuity 候选，background completion 与 task/worktree isolation 进入 Durable operation 审计，team request-response 进入既有 Host/HITL/runtime-input conformance；不得据此复制 agent loop、task board、mailbox、skill loader、worktree manager 或平行状态机。

| 成熟度 | 候选方向 | 可吸收点 | 必须复用的 Baymax owner | 触发信号与首要边界 |
| --- | --- | --- | --- | --- |
| 已归档基线（137） | 跨 Provider handoff 与 stream edge conformance | 跨 provider tool-call/thinking 转换、abort usage、overflow、Unicode/空内容 fixture | `model/<provider>`、provider admission、context handoff、failure taxonomy、terminal outcome | 已完成并归档；后续仅在新的可复现 drift 下以增量 change 处理，不重新排期为 P2。 |
| 首选审计候选（当前 P1） | Durable task-attempt/workspace binding 与 completion safe-point 所有权审计 | 显式验证 task/attempt 与 workspace provenance 的关联、attempt/lease rollover 隔离、missing/dirty/conflict/drift 分类、恢复 reconciliation；审计后台完成结果进入下一模型决策安全点时的 correlation、dedupe、late/disconnect/recovery 语义 | scheduler `Task/Attempt` 与 lease、checkpoint/workspace provenance、snapshot/recovery、mailbox、source-owned runtime input、`RuntimeRecorder` | 代码审计已确认 checkpoint 具备 workspace provenance 和完整性漂移检测，但 scheduler attempt 尚无显式 workspace binding。先以 gap fixture 证明并行、重试或恢复中的真实冲突，再启动 contract 提案；实际 workspace/Git 生命周期仍由 host/tool adapter 拥有，不新增 worktree manager、通知队列或任务状态机。 |
| 条件候选 | 外部 Extension authoring conformance | extension authoring eval、真实 workflow fixture、失败反馈 | extension lifecycle/resource resolution、manifest/capability、allowlist、sandbox | 出现新的真实扩展来源。不得建设 package manager/market，也不得无准入动态执行扩展。 |
| 观察候选 | Model catalog 与本地模型路由增量 | runtime model discovery、本地模型 router、明确 auth preflight | provider/model catalog、credential preflight、readiness、host injection | 静态或宿主注入 catalog 无法满足明确路由需求。不引入 credential store，不在 `context/*` 引入 provider SDK。 |
| 观察候选 | Eval transcript/artifact comparison 与 compaction continuity | baseline/candidate harness、有界 transcript/snapshot artifact 引用；验证 compaction/handoff 前后的 agent/role/team、task/attempt/lease、workspace binding、pending correlated request、objective 与权威 checkpoint 连续性 | OTel/eval/corpus、context handoff、checkpoint/snapshot/artifact refs、diagnostics replay | 现有 eval 无法定位可复现质量回归，或压缩/恢复后出现身份、任务或工作区事实漂移。摘要不得成为新事实源，不持久化 reasoning body；只增加有界引用与比较，不建立 transcript/artifact service。 |

备选池合并与排序规则：

1. active Run control、命令关联、JSONL framing、output integrity、event backpressure、基础 HITL bridge 与 steering/follow-up 已归档；后续只允许在既有 owner 下验证 request ID、approve/reject、timeout、duplicate、late response 与权威终态一致性，不抽取新的通用 coordination FSM。
2. 跨 Provider conformance 采用 fixture/test/gate-first，只在可复现 drift 下最小修正 `model/<provider>`，不先建设新的共享路由或 wire abstraction。
3. 跨 Provider conformance 已归档；下一审计优先验证 task/attempt 到 workspace provenance 的接缝。只有 gap fixture 证明 lease rollover、重试或恢复会误用 workspace，或后台 completion 会丢失/重复 promotion 时，才升级为 OpenSpec change；后台完成通知作为同一 owner 审计的次级切面，优先复用 mailbox 与 runtime-input safe point。
4. Eval continuity 只验证有界事实和引用在 compaction/handoff/recovery 前后的确定性，不保存 raw reasoning，也不把模型摘要提升为 task/session/workspace 的事实源。
5. Extension、Eval 方向继续采用需求触发，不因外部项目存在同名能力而自动立项；`learn-claude-code` 的 daemon thread、`shell=True`、JSON/JSONL 双写、轮询认领和非事务 worktree index 只作为反例，不进入生产基线。
6. Pi 的 lane/register/ledger、experimental CBOR protocol、remote Session Server、attachment/lease、SQLite hosted backend 保持长期延后，不成为 Baymax 默认公共模型，也不进入近期备选。

### 需求触发观察项

以下方向不预设提案顺序，只有出现相应信号才进入候选审计：

| 方向 | 触发信号 | 首要边界 |
| --- | --- | --- |
| 远程 model catalog 或本地模型路由 | 静态/宿主注入目录不足以支持明确的路由需求 | 以本页“Model catalog 与本地模型路由增量”为同一候选，不创建平行提案；不在 `context/*` 引入 provider SDK，不接入 credential store。 |
| 运行时成本或延迟治理增量 | 成本或 P95 抖动成为稳定主线瓶颈 | 复用 operation profile、timeout resolution、budget admission 与既有诊断字段。 |
| 评测、tracing 或 compaction continuity 增量 | 跨后端字段解释不一致、出现可复现质量回归，或 compaction/handoff/recovery 后 identity/task/workspace/pending-request 事实漂移 | 以本页“Eval transcript/artifact comparison 与 compaction continuity”为同一候选；复用 OTel/eval/corpus、context handoff、checkpoint/snapshot 和 bounded refs，不引入评测控制面或第二事实源。 |
| 外部 extension 生态增量 | 新的真实扩展来源、供应链审计或隔离需求超出现有 lifecycle contract | 复用 manifest/capability/allowlist；不建设 package manager 或市场。 |

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
