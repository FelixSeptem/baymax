## Why

Baymax 已经积累了架构边界、运行时配置与诊断、provider、MCP、扩展、contract/replay/gate 和示例文档，但这些内容分散在 README、模块 README 与专题文档中，缺少面向新用户、集成方和贡献者的统一技术文档路径。README 同时承担定位、架构、快速开始、能力快照和治理说明，难以继续扩展，也容易与实施状态产生漂移。

现在需要建立一套中文为主、分层维护的系统技术文档，并把 README 重构为稳定的入口和导航页。同时，任何后续 OpenSpec 提案都必须单独完成文档影响评估，明确需要新增、修改或无需变更的文档与门禁，避免代码、contract、配置、示例和文档脱节。

## What Changes

- 建立分层技术文档信息架构：架构总览、运行生命周期、关键组件、集成指南、配置/诊断、测试/replay/gate、故障排查、最佳实践与贡献流程。
- 以现有文档和代码 owner 为事实来源，补充统一入口、术语、读者路径、模块责任、依赖边界、数据/事件流和验证入口；避免复制已有 spec 或形成第二事实源。
- 重构根 README：保留项目定位、能力边界、最小快速开始和必要示例，将详细架构、组件和治理内容下沉到 `docs/` 并提供分层导航。
- 为每个后续 OpenSpec 提案引入强制的 Documentation Impact Assessment：判断架构、组件、配置、contract、诊断 schema、示例、CLI/API、最佳实践和 roadmap 是否受影响，并在 proposal/design/tasks 中声明结果。
- 增加文档漂移治理：提案实施期间同步维护文档任务；门禁检查 README/专题文档链接、模块清单、roadmap 状态、OpenSpec 状态和变更声明的一致性。
- 提供文档迁移、命名、链接、弃用和回滚规则；本 change 不改变运行时行为、provider contract、配置语义或 examples/agent-modes 的运行语义。

## Capabilities

### New Capabilities

无。此 change 是纯文档体系与治理工具变更，不引入运行时或 API contract 能力，因此在 `.openspec.yaml` 中声明 `skip_specs: true`。

### Modified Capabilities

无。现有 runtime、provider、tool、orchestration、diagnostics、replay 和示例 contract 不变。

## Impact

- 主要影响 `README.md`、`docs/` 信息架构、文档索引和文档一致性脚本；可能新增 `docs/architecture/`、`docs/components/`、`docs/guides/` 等目录。
- 影响 OpenSpec 提案模板/校验流程，使每个新提案都必须声明文档影响与同步证据。
- 不改变 Go 包、公开 API、配置键、诊断 schema、运行时终态或 provider SDK 适配逻辑。
- `examples/agent-modes` 不改变行为；本提案若仅增加链接和文档索引，Example Impact Assessment 为：**无需示例变更（附理由）**。
- 交付应覆盖中文主文档，代码标识、API、协议和错误分类保留英文；英文完整翻译不纳入首阶段。

## Example Impact Assessment

无需示例变更（附理由）：本 change 只新增技术文档信息架构、README 导航和 OpenSpec 文档影响治理，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 新增文档 | `docs/architecture/*` | maintainer | `scripts/check-docs-consistency.ps1` |
| components | 新增文档 | `docs/components/*` | package owners | component source/test links |
| configuration | 修改文档 | `docs/guides/configuration-and-diagnostics.md` | runtime/config | `go test ./runtime/config` |
| contract/API | 无需文档变更（附理由） | — | maintainer | no runtime/API contract changes |
| diagnostics | 修改文档 | `docs/guides/configuration-and-diagnostics.md` | observability | diagnostics replay gate |
| examples | 无需文档变更（附理由） | — | example owners | no examples/agent-modes behavior changes |
| CLI/integration | 新增文档 | `docs/guides/integration.md` | integration owners | link/reference check |
| best practices | 新增文档 | `docs/guides/best-practices.md` | maintainer | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap status gate |

该评估同时定义后续提案的强制模板；漂移门禁会验证三份提案工件中的表格、路径、owner 和 verification 是否存在且合法。
