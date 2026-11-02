# Baymax Agent Loop (Go)

Baymax 是一个 library-first、contract-first 的 Go Agent runtime，面向可嵌入的单 agent、多 agent、工具和 MCP 编排。它提供 Run/Stream 主循环、Provider 适配、Context projection、结构化诊断、replay 和离线治理门禁；宿主仍拥有 credential、网络、持久化和全局路由。

## 5 分钟开始

要求 Go 1.26（仓库通过 `go.mod` 的 `toolchain` 声明验证工具链；建议使用当前
Go 1.26 安全补丁版本）。从仓库根目录运行：

```bash
go run ./examples/01-chat-minimal
```

接着按[开始使用](docs/guides/getting-started.md)完成离线验证、真实 Provider
接入和宿主责任清单，再按[集成指南](docs/guides/integration.md)选择工具、MCP
或多 agent 路径。最小示例只展示 library 接入，不代表生产配置、重试或安全策略。

## 能力边界

- 统一 Run/Stream 主循环与终态语义。
- OpenAI、Anthropic、Gemini 等 Provider adapter（协议细节位于 `model/<provider>`）。
- bounded source-neutral execution topology projection across workflow, team and individual child dispatch; owner execution and restore digests remain separate. See [execution graph guide](docs/guides/execution-graph.md).
- bounded source-neutral execution topology projection across workflow, team and individual child dispatch; owner execution and restore digests remain separate. See [execution graph guide](docs/guides/execution-graph.md).
- local tool、MCP HTTP/STDIO、workflow、teams、A2A、scheduler、composer。
- Context budget/projection、结构化 timeline/diagnostics、RuntimeRecorder 单写入口。
- contract/replay/gate 作为离线、bounded、可重复的证据。
- opt-in 原生执行事件 adapter：真实回调、独立宿主 references、保守 source capabilities 和有界观察失败反馈，见 [接入示例](examples/native-execution-events/README.md)。

不提供自动 Provider 探测、隐式全局 registry、远程控制面或跨租户状态。架构与依赖边界见 [架构总览](docs/architecture/overview.md)、[设计原则](docs/architecture/design-principles.md) 和 [模块 ownership](docs/architecture/module-ownership.md)。

## 按读者进入

- 新用户： [文档索引](docs/README.md) → [开始使用](docs/guides/getting-started.md) → [架构总览](docs/architecture/overview.md) → [集成指南](docs/guides/integration.md) → [最小示例](examples/01-chat-minimal)。
- 集成方： [设计原则](docs/architecture/design-principles.md) → [运行生命周期](docs/architecture/runtime-lifecycle.md) → [数据与事件流](docs/architecture/data-event-flow.md) → [配置与诊断](docs/guides/configuration-and-diagnostics.md) → [测试/replay/gate](docs/guides/testing-replay-gates.md)。
- 贡献者： [贡献指南](docs/guides/contribution.md) → [OpenSpec 工作流](docs/guides/openspec-workflow.md) → [故障排查](docs/guides/troubleshooting.md)。

## 当前状态与路线图

当前状态的唯一依据是 `openspec list --json`、[开发路线图](docs/development-roadmap.md) 和 [归档索引](openspec/changes/archive/INDEX.md)。项目处于 **`0.x` pre-1 阶段**：不做 `1.0.0/prod-ready` 承诺；新增能力必须经过 OpenSpec、测试、文档影响评估和回滚审查。

### 版本阶段快照

`0.x` 阶段允许新增能力型提案，但每项提案必须明确 Why now、风险、回滚、文档影响和验证命令；本项目不把归档数量或 package 数量当作成熟度承诺。

OpenSpec 状态以 `openspec list --json`、[路线图](docs/development-roadmap.md) 和 [归档索引](openspec/changes/archive/INDEX.md) 为准。能力边界和仍不支持的组合也见 [集成能力矩阵](docs/guides/integration.md)，不要从 package 名称或旧的 release snapshot 推断。

要了解某个历史决策，请从归档索引进入对应 proposal/design/verification；不要把归档记录当作当前进行中的工作。

## 文档与事实源

- [技术文档索引](docs/README.md)
- [架构总览](docs/architecture/overview.md)
- [Execution Model](docs/architecture/execution-model.md)
- [设计原则](docs/architecture/design-principles.md)
- [运行生命周期](docs/architecture/runtime-lifecycle.md)
- [数据与事件流](docs/architecture/data-event-flow.md)
- [模块 ownership](docs/architecture/module-ownership.md)
- [事实源与迁移矩阵](docs/documentation-ownership.md)
- [文档写作与页面模板](docs/documentation-style-guide.md)
- [开发路线图](docs/development-roadmap.md)
- [运行时模块边界](docs/runtime-module-boundaries.md)
- [Runtime Harness 架构](docs/runtime-harness-architecture.md)
- [主线契约测试索引](docs/mainline-contract-test-index.md)
- [运行时配置与诊断](docs/runtime-config-diagnostics.md)
- [API 参考覆盖](docs/api-reference-d1.md)
- [Diagnostics Replay](docs/diagnostics-replay.md)
- [外部适配模板索引](docs/external-adapter-template-index.md)
- [适配迁移映射](docs/adapter-migration-mapping.md)
- [版本与兼容](docs/versioning-and-compatibility.md)

### 模块 README

`a2a/README.md` · `core/runner/README.md` · `core/types/README.md` · `tool/local/README.md` · `mcp/README.md` · `model/README.md` · `context/README.md` · `context/handoff/README.md` · `context/hostprojection/README.md` · `memory/README.md` · `orchestration/README.md` · `adapter/README.md` · `host/eventadapter/README.md` · `integration/README.md` · `runtime/config/README.md` · `runtime/diagnostics/README.md` · `runtime/security/README.md` · `observability/README.md` · `skill/loader/README.md`

### 示例

`examples/01-chat-minimal`、`examples/quickstart`、`02-tool-loop-basic`、`03-mcp-mixed-call`、`04-streaming-interrupt`、`05-parallel-tools-fanout`、`06-async-job-progress`、`07-09` 多 agent 示例，以及 `examples/agent-modes/MATRIX.md` 模式矩阵。

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

Baymax includes the opt-in [`host_execution_plan.v1`](orchestration/planbridge/README.md) bridge for hosts that need a bounded descriptor compiled against exactly one native execution owner. It supports inline direct, terminal-only parallel/vote teams, flat workflows, isolated local child dispatch and manifest-approved `mailbox_async` delegation to the existing mailbox owner. Resumed/durable continuation, subgraphs and shared child queues remain rejected before activation; descriptor digests and capability evidence are consistency facts, not authorization or business commit. See the [offline example](examples/host-execution-plan/README.md).

## Source-owned native event history

The source-history contract is source-owned and read-only. Mailbox retains lifecycle facts through its existing MemoryStore/FileStore owner; runner, child, team and workflow can expose a retained journal only when the caller explicitly injects the matching owner journal. Memory journals are process-local; FileStore supports same-host single-writer sequential reopen only. The native event adapter remains a real-time callback projection: no callback ID or host observation sequence is a durable cursor, and replay cannot run, retry, resume, restore or commit a business effect. See the [support matrix and integration guide](docs/guides/integration.md) and [offline example](examples/native-execution-events/README.md).
