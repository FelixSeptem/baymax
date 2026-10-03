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

不提供自动 Provider 探测、隐式全局 registry、远程控制面或跨租户状态。架构与依赖边界见 [架构总览](docs/architecture/overview.md) 和 [模块边界](docs/runtime-module-boundaries.md)。

## 按读者进入

- 新用户： [文档索引](docs/README.md) → [集成指南](docs/guides/integration.md) → [最佳实践](docs/guides/best-practices.md)。
- 集成方： [架构总览](docs/architecture/overview.md) → [配置与诊断](docs/guides/configuration-and-diagnostics.md) → [测试/replay/gate](docs/guides/testing-replay-gates.md)。
- 贡献者： [贡献指南](docs/guides/contribution.md) → [OpenSpec 工作流](docs/guides/openspec-workflow.md) → [故障排查](docs/guides/troubleshooting.md)。

## 当前状态与路线图

当前状态的唯一依据是 `openspec list --json`、[开发路线图](docs/development-roadmap.md) 和 [归档索引](openspec/changes/archive/INDEX.md)。

### 版本阶段快照

项目处于 **`0.x` pre-1 阶段**：不做 `1.0.0/prod-ready` 承诺；`0.x` 阶段允许新增能力型提案，但必须遵守 OpenSpec、测试、文档影响评估和回滚要求。

当前进行中的 OpenSpec change：

- `introduce-dynamic-action-gate-and-native-run-resume`（进行中）：动态 PendingAction 的 opaque action reference、`input_required` pause、同一 Run/Stream checkpoint resume；当前已完成类型合同、Runner 基础切片与 Host 可选入口，完整 replay/diagnostics/docs 仍在实施。

最近归档：

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

每个新 OpenSpec proposal/design/tasks 都必须包含 Documentation Impact Assessment，逐项判断 architecture、components、configuration、contract/API、diagnostics、examples、CLI/integration、best practices、roadmap，并提供 affected paths、owner 和 verification。行为/配置/contract/诊断/示例变化还必须声明 Example Impact Assessment。

文档门禁：

- `scripts/check-openspec-documentation-impact.sh/.ps1`：文档影响声明、变更面、链接、任务和状态漂移。
- `scripts/check-openspec-example-impact-declaration.sh/.ps1`：示例影响声明。
- `scripts/check-openspec-roadmap-status-consistency.sh/.ps1`：roadmap/OpenSpec/archive 状态一致性。

## 许可证与社区

详见 [CONTRIBUTING.md](CONTRIBUTING.md)、[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)、[SECURITY.md](SECURITY.md)、[CHANGELOG.md](CHANGELOG.md) 和 [LICENSE](LICENSE)。
