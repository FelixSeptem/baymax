# Baymax Agent Loop (Go)

Baymax 是一个 library-first、contract-first 的 Go Agent runtime，面向可嵌入的单 agent、多 agent、工具和 MCP 编排。它提供 Run/Stream 主循环、Provider 适配、Context projection、结构化诊断、replay 和离线治理门禁；宿主仍拥有 credential、网络、持久化和全局路由。

## 5 分钟开始

```bash
go run ./examples/01-chat-minimal
```

接着按 [集成指南](docs/guides/integration.md) 选择工具、MCP 或多 agent 路径。最小示例只展示 library 接入，不代表生产配置、重试或安全策略。

## 能力边界

- 统一 Run/Stream 主循环与终态语义。
- OpenAI、Anthropic、Gemini 等 Provider adapter（协议细节位于 `model/<provider>`）。
- local tool、MCP HTTP/STDIO、workflow、teams、A2A、scheduler、composer。
- Context budget/projection、结构化 timeline/diagnostics、RuntimeRecorder 单写入口。
- contract/replay/gate 作为离线、bounded、可重复的证据。
- opt-in 原生执行事件 adapter：真实回调、独立宿主 references、保守 source capabilities 和有界观察失败反馈，见 [接入示例](examples/native-execution-events/README.md)。

不提供自动 Provider 探测、隐式全局 registry、远程控制面或跨租户状态。架构与依赖边界见 [架构总览](docs/architecture/overview.md) 和 [模块边界](docs/runtime-module-boundaries.md)。

## 按读者进入

- 新用户： [文档索引](docs/README.md) → [集成指南](docs/guides/integration.md) → [最佳实践](docs/guides/best-practices.md)。
- 集成方： [架构总览](docs/architecture/overview.md) → [配置与诊断](docs/guides/configuration-and-diagnostics.md) → [测试/replay/gate](docs/guides/testing-replay-gates.md)。
- 贡献者： [贡献指南](docs/guides/contribution.md) → [OpenSpec 工作流](docs/guides/openspec-workflow.md) → [故障排查](docs/guides/troubleshooting.md)。

## 当前状态与路线图

当前状态的唯一依据是 `openspec list --json`、[开发路线图](docs/development-roadmap.md) 和 [归档索引](openspec/changes/archive/INDEX.md)。

### 版本阶段快照

项目处于 **`0.x` pre-1 阶段**：不做 `1.0.0/prod-ready` 承诺；`0.x` 阶段允许新增能力型提案，但必须遵守 OpenSpec、测试、文档影响评估和回滚要求。

- `source-history-failure-and-workflow-claim-recovery`（归档 175；显式 source history append health 与 workflow resume claim recovery/fencing）。

- 最近归档：`make-source-history-projection-transactional`（2026-10-24）；为显式 source owner 增加 bounded source commit/projection recovery evidence。该 change 不新增执行 form、队列或业务数据库事务；未绑定 ledger 的 history 仍只提供 retained replay，事务不确定窗口返回 typed `recovery-required`。

- `retain-run-and-composite-source-history`（归档 171；source-native history/cursor/replay 增量）

- `restore-workflow-continuations`（归档 172；workflow-first composite continuation）：为 workflow owner 增加 source-owned graph checkpoint/restore、三类 pause safe point、原子 claim、new-Engine file-store reopen 与只读 replay evidence。child/team/mailbox/planbridge resumed 继续在准入前返回 `unsupported_capability`。

- `enforce-openspec-public-semantics-documentation-contract`（归档 161；[公开语义文档同步治理](openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/proposal.md)）：要求每个 active OpenSpec 的三份工件声明对外语义文档同步承诺，并由双平台门禁检查。
- `establish-native-execution-event-adapter-contract`（归档 160；[原生执行事件与宿主接入契约](openspec/changes/archive/160-establish-native-execution-event-adapter-contract/proposal.md)）：已完成版本化 native execution event、bounded host references、source capability 声明、观察失败反馈、真实 Run/Stream 集成、离线 replay、示例与维护文档同步。
- `establish-host-execution-plan-bridge-contract`（归档 162；[宿主执行计划桥接提案](openspec/changes/archive/162-establish-host-execution-plan-bridge-contract/proposal.md)）：已交付 opt-in、bounded 的 `host_execution_plan.v1` 编译与薄委托 API，将版本化计划和独立 native binding 在无副作用的全计划预检后交给既有 runner/teams/workflow/scheduler owner。支持矩阵、独立宿主授权、回滚与离线运行示例见 [设计](openspec/changes/archive/162-establish-host-execution-plan-bridge-contract/design.md) 和 [示例](examples/host-execution-plan/README.md)。其余 3 项未启动候选见路线图。
- `establish-host-resume-admission-revalidation-contract`（归档 164；[恢复准入提案](openspec/changes/archive/164-establish-host-resume-admission-revalidation-contract/proposal.md)）：`runner.WithHostResumeVerifier` 提供默认关闭的宿主授权/revision 重验证、完整 duplicate identity 与并发单次提交，保留原生 Run/Stream 和 resolution/terminal 语义。[接入指南](docs/guides/host-resume-admission.md) 与[离线示例](examples/host-resume-admission/README.md) 说明 API、limits、错误和迁移；仅支持原 Engine 恢复，不开放 planbridge resumed。
- `establish-host-context-assembly-handoff-contract`（归档 165；[proposal](openspec/changes/archive/165-establish-host-context-assembly-handoff-contract/proposal.md)）：已交付显式 opt-in、reference-only 的 `host_context_projection.v1` direct assembly contract。其固定 API 限制为 32 KiB descriptor、64 个 opaque references、每个 reference 256 UTF-8 bytes、每个 digest 128 UTF-8 bytes；有限的 `host_context.*` 拒绝分类不泄露 source body 或 callback error。direct Run/Stream 仅在既有 Context Assembler 启用时复用该 owner；禁用时保留 assembler bypass 语义且不 admit/resolve projection。合法 `handoff.v1` correlation 与带 fresh current-authority resolver 的 restore 已复用既有 owner；dedicated replay、shell/PowerShell gate、标准 quality path 的静态接线、全量 quality verification 与[可运行离线示例](examples/host-context-assembly/README.md)均已交付。
- `remediate-strict-security-vulnerabilities`（归档 163；[安全维护提案](openspec/changes/archive/163-remediate-strict-security-vulnerabilities/proposal.md)）：已完成 Go patched toolchain 与严格漏洞扫描最小依赖图升级，保持 strict 扫描策略，并以双入口完整质量门禁验证收口。开发环境基线见 [贡献指南](CONTRIBUTING.md)。
- `establish-application-outbox-mailbox-recovery-contract`（归档 166；[提案](openspec/changes/archive/166-establish-application-outbox-mailbox-recovery-contract/proposal.md)）：已交付 `application_outbox_dispatch.v1` 的显式 opt-in、reference-only `orchestration/outboxbridge`，复用 mailbox/scheduler 原生 owner，包含离线示例、replay fixture、Run/Stream conformance 与双平台 contract gate；不能把 dispatch acceptance 当作业务提交。
- `retain-run-and-composite-source-history`（Spec 2 增量；BXM-2）：已为 runner、isolated child、team、workflow 提供显式 source-owner journal facade 与 bounded replay primitive；默认未注入 journal 仍显式 unsupported，FileStore 只承诺同机单写顺序重开。
- `persist-native-continuations-and-restore-composite-graphs`（Spec 3；BXM-3 + BXM-4，归档 169）：已建立 direct 的 file-backed checkpoint、跨进程 claim/restore 与 `input_required`、`approval_required`、`budget_exhausted` source evidence；child/team/workflow/mailbox 与 planbridge resumed 仍在 admission 前拒绝。checkpoint restore 不等于 history replay 或重新 Run parent。离线验证入口见 [durable continuation](examples/durable-continuation-restore/README.md)。
- `evaluate-bounded-harness-candidate-comparisons`（已归档 170）：已落地离线、review-only 的候选实验身份、scope admission、baseline selection、评分隔离、逐 attempt 成本证据与只读 replay；七类 fixture、legacy/idempotency replay、双平台专项门禁及全量质量门禁已通过。契约与使用边界见 [评测契约](docs/evaluation-contract.md)。

- `enable-team-and-mailbox-native-continuations`（已归档 177；team/mailbox source-owned continuation）：已完成 team parallel/vote 与 mailbox host-plan 的 native pause/checkpoint/restore、atomic claim、lease/retry/DLQ 与 Run/Stream parity；缺失 source evidence 的 form/pause-reason 继续保持 `unsupported_capability` 或 `recovery-required`，不得降级为重新 Run、重新 enqueue、latest replay 或 callback replay。

- `bound-durable-store-deployment-scope`（已归档 174；same-host single-writer sequential reopen deployment scope）：已固化 source history、checkpoint、mailbox 与 composite owner store 的部署边界；多写者、跨机器、分布式 cursor transaction 和多实例 resume claim 在 source evidence 不足时保持 typed `unsupported` 或 `recovery-required`。

最近归档：

- `establish-host-resume-admission-revalidation-contract`（归档 164）：宿主恢复准入重验证、完整 duplicate identity、Run/Stream/Host conformance、离线 replay、双平台 gate 与严格联合质量门禁。
- `remediate-strict-security-vulnerabilities`（归档 163）：Go patched toolchain、最小严格漏洞扫描依赖图、受控 recorder baseline 校准与完整质量门禁。
- `establish-host-execution-plan-bridge-contract`（归档 162）：版本化计划描述与 native binding、全计划预检、既有执行 owner 薄委托、离线 replay、示例与双平台门禁。
- `enforce-openspec-public-semantics-documentation-contract`（归档 161）：公开语义变化的维护文档同步承诺、三份提案工件一致性及双平台门禁。
- `establish-native-execution-event-adapter-contract`（归档 160）：版本化原生执行事件与宿主 adapter 合同、bounded references、observer failure 反馈、Run/Stream parity、离线 replay、示例及维护文档同步。
- `fix-dynamic-action-resolution-terminal-event-semantics`（归档 159）：拆分动态 action resolution 与 canceled terminal 事件；confirm、deny、timeout 统一发出一次 resolution，只有 deny/timeout 发出 canceled terminal，并补齐 Run/Stream 与 duplicate resume 事件回归。
- `introduce-dynamic-action-gate-and-native-run-resume`（归档 158）：动态 PendingAction 的 opaque action reference、`input_required` pause、同一 Run/Stream checkpoint resume。
- `add-explicit-model-capability-adapter`（归档 155）：显式 model capability adapter、Stream preflight 诊断与最小模板澄清。
- `layered-technical-documentation-and-drift-governance`（归档 156）：分层技术文档、README 导航和新提案 Documentation Impact Assessment/漂移门禁。

## 文档与事实源

- [技术文档索引](docs/README.md)
- [事实源与迁移矩阵](docs/documentation-ownership.md)
- [文档写作与页面模板](docs/documentation-style-guide.md)
- [开发路线图](docs/development-roadmap.md)
- [运行时模块边界](docs/runtime-module-boundaries.md)
- [Runtime Harness 架构](docs/runtime-harness-architecture.md)
- [主线契约测试索引](docs/mainline-contract-test-index.md)
- [运行时配置与诊断](docs/runtime-config-diagnostics.md)
- [Diagnostics Replay](docs/diagnostics-replay.md)
- [外部适配模板索引](docs/external-adapter-template-index.md)
- [适配迁移映射](docs/adapter-migration-mapping.md)
- [版本与兼容](docs/versioning-and-compatibility.md)

### 模块 README

`a2a/README.md` · `core/runner/README.md` · `core/types/README.md` · `tool/local/README.md` · `mcp/README.md` · `model/README.md` · `context/README.md` · `orchestration/README.md` · `adapter/README.md` · `runtime/config/README.md` · `runtime/diagnostics/README.md` · `runtime/security/README.md` · `observability/README.md` · `skill/loader/README.md`

### 示例

`examples/01-chat-minimal`、`02-tool-loop-basic`、`03-mcp-mixed-call`、`04-streaming-interrupt`、`05-parallel-tools-fanout`、`06-async-job-progress`、`07-09` 多 agent 示例，以及 `examples/agent-modes/MATRIX.md` 模式矩阵。

Agent mode 专项门禁：`scripts/check-agent-mode-real-runtime-semantic-contract.sh`、`scripts/check-agent-mode-readme-runtime-sync-contract.sh`、`scripts/check-agent-mode-anti-template-contract.sh`、`scripts/check-agent-mode-doc-first-delivery-contract.sh`（Windows 使用同名 `.ps1` 入口）。

## 开发验证

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
bash scripts/check-docs-consistency.sh
bash scripts/check-openspec-documentation-impact.sh
bash scripts/check-quality-gate.sh
```

Windows 使用等价的 `pwsh -File scripts/check-docs-consistency.ps1`、`check-openspec-documentation-impact.ps1` 和 `check-quality-gate.ps1`。文件占用时，按 [故障排查](docs/guides/troubleshooting.md) 使用隔离缓存逐包验证并记录未执行项。

## 提案治理

每个新 OpenSpec proposal/design/tasks 都必须包含 Documentation Impact Assessment，逐项判断 architecture、components、configuration、contract/API、diagnostics、examples、CLI/integration、best practices、roadmap，并提供 affected paths、owner 和 verification。行为/配置/contract/诊断/示例变化还必须声明 Example Impact Assessment。三份工件均须接受 Public Semantic Documentation Commitment；公开 API/contract、运行时行为、事件与终态、错误、配置、诊断、CLI/integration 或示例语义发生变化时，维护文档必须与变更同批提交，声明维护路径、owner 和验证证据；无变化也须写具体理由。

文档门禁：

- `scripts/check-openspec-documentation-impact.sh/.ps1`：文档影响及公开语义同步承诺、跨工件一致性、变更面和链接；缺失、非法、无同步证据及声明漂移分别报出稳定分类码。
- `scripts/check-openspec-example-impact-declaration.sh/.ps1`：示例影响声明。
- `scripts/check-openspec-roadmap-status-consistency.sh/.ps1`：roadmap/OpenSpec/archive 状态一致性。

## 许可证与社区

详见 [CONTRIBUTING.md](CONTRIBUTING.md)、[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)、[SECURITY.md](SECURITY.md)、[CHANGELOG.md](CHANGELOG.md) 和 [LICENSE](LICENSE)。
# Host execution plan bridge

Baymax includes the opt-in [`host_execution_plan.v1`](orchestration/planbridge/README.md) bridge for hosts that need a bounded descriptor compiled against exactly one native execution owner. It supports inline direct, parallel/vote teams, flat workflows, isolated local child dispatch and manifest-approved `mailbox_async` delegation to the existing mailbox owner. Team/mailbox continuation is additionally available only through an explicitly configured File-backed owner with matching capability and pause-reason evidence; the bridge delegates claim/projection and never creates a second graph, queue or worker. Resumed planbridge execution, subgraphs and shared child queues remain rejected before activation; descriptor digests and capability evidence are consistency facts, not authorization or business commit. See the [offline example](examples/host-execution-plan/README.md).

## Source-owned native event history

The source-history contract is source-owned and read-only. Mailbox retains lifecycle facts through its existing MemoryStore/FileStore owner; runner, child, team and workflow can expose a retained journal only when the caller explicitly injects the matching owner journal. Memory journals are process-local; FileStore supports same-host single-writer sequential reopen only. The native event adapter remains a real-time callback projection: no callback ID or host observation sequence is a durable cursor, and replay cannot run, retry, resume, restore or commit a business effect. See the [support matrix and integration guide](docs/guides/integration.md) and [offline example](examples/native-execution-events/README.md).
