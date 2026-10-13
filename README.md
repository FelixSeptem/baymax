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

- `enforce-openspec-public-semantics-documentation-contract`（归档 161；[公开语义文档同步治理](openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/proposal.md)）：要求每个 active OpenSpec 的三份工件声明对外语义文档同步承诺，并由双平台门禁检查。
- `establish-native-execution-event-adapter-contract`（归档 160；[原生执行事件与宿主接入契约](openspec/changes/archive/160-establish-native-execution-event-adapter-contract/proposal.md)）：已完成版本化 native execution event、bounded host references、source capability 声明、观察失败反馈、真实 Run/Stream 集成、离线 replay、示例与维护文档同步。
- `establish-host-execution-plan-bridge-contract`（进行中；[宿主执行计划桥接提案](openspec/changes/establish-host-execution-plan-bridge-contract/proposal.md)）：已交付 opt-in、bounded 的 `host_execution_plan.v1` 编译与薄委托 API，将版本化计划和独立 native binding 在无副作用的全计划预检后交给既有 runner/teams/workflow/scheduler owner。支持矩阵、独立宿主授权、回滚与离线运行示例见 [设计](openspec/changes/establish-host-execution-plan-bridge-contract/design.md) 和 [示例](examples/host-execution-plan/README.md)；归档仍以 change 的完整验证和任务证据为前提。其余 3 项未启动候选见路线图。

最近归档：

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

Baymax includes the opt-in [`host_execution_plan.v1`](orchestration/planbridge/README.md) bridge for hosts that need a bounded descriptor compiled against exactly one native execution owner. It supports inline direct, terminal-only parallel/vote teams, flat workflows and isolated local child dispatch. It intentionally rejects mailbox delivery, resumed/durable continuation, subgraphs and shared child queues before activation. See the [offline example](examples/host-execution-plan/README.md); descriptor digests are consistency evidence, not authorization or business commit.
