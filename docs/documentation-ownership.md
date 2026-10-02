# 文档事实源与迁移矩阵

本文档是 Baymax 技术文档的 canonical source map。概览页可以解释边界和导航，但不得复制另一份完整 contract；发生冲突时，以本表的事实源和代码测试为准。

| 主题 | canonical source | 入口/别名 | owner | 验证 |
| --- | --- | --- | --- | --- |
| 项目定位与快速开始 | `README.md` | `docs/README.md` | maintainer | `scripts/check-docs-consistency.ps1` |
| 架构边界 | `docs/runtime-module-boundaries.md` | `docs/architecture/module-ownership.md` | runtime maintainers | `scripts/check-runtime-boundaries.sh` |
| 运行生命周期 | `docs/architecture/runtime-lifecycle.md` | `docs/runtime-harness-architecture.md` | core/runner | `go test ./core/runner` |
| 配置与诊断 | `docs/runtime-config-diagnostics.md` | `docs/guides/configuration-and-diagnostics.md` | runtime/config, runtime/diagnostics | `go test ./runtime/config ./runtime/diagnostics` |
| Provider 协议适配 | `model/*`, `docs/api-reference-d1.md` | `docs/components/providers.md` | model owners | provider contract gates |
| 工具与 MCP | `tool/*`, `mcp/*`, `docs/mcp-runtime-profiles.md` | `docs/components/tools.md`, `docs/components/mcp.md` | tool/mcp owners | adapter and MCP gates |
| 编排 | `orchestration/*`, `a2a/*` | `docs/components/orchestration.md` | orchestration owners | `go test ./orchestration/... ./a2a/...` |
| 可观测性 | `observability/*`, `runtime/diagnostics/*` | `docs/components/observability.md` | observability owners | diagnostics replay gate |
| 适配器 | `adapter/*`, `integration/*`, `docs/adapter-migration-mapping.md` | `docs/components/adapters.md` | adapter owners | adapter conformance gates |
| 示例 | `examples/*`, `examples/agent-modes/MATRIX.md` | `docs/guides/integration.md` | example owners | example smoke/doc-first gates |
| OpenSpec 流程 | `openspec/`, `CONTRIBUTING.md` | `docs/guides/openspec-workflow.md` | maintainers | `openspec validate --all` |
| 路线图与状态 | `docs/development-roadmap.md`, `openspec/changes/archive/INDEX.md` | README 状态快照 | release owner | roadmap status gate |

## 迁移规则

1. 新页面先在本表登记 owner、事实源和验证命令，再加入索引。
2. 拆分旧页面时保留旧路径，并在旧页面顶部放置指向 canonical 页面和迁移说明的链接。
3. 代码、测试或 OpenSpec contract 是行为事实源；概览文档只描述语义和链接，不重新定义字段约束。
4. 页面废弃至少保留一个 minor 周期；删除前必须在变更提案中声明迁移和回滚路径。
5. 页面文件名使用业务语义，禁止使用提案编号或临时分支名。

## 维护责任

每个变更的 proposal 必须完成 Documentation Impact Assessment。受影响主题必须同时更新本表（若 owner、事实源或验证命令改变），并在 tasks 中留下可复核的路径和命令证据。
