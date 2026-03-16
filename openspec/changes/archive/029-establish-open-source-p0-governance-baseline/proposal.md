## Why

项目已具备较完整的运行时能力，但作为开源项目仍缺少最小治理基线，外部用户难以判断升级风险、贡献流程与安全响应入口。现在补齐 P0 治理资产，可以在不改 runtime 行为的前提下显著降低使用与协作门槛。

## What Changes

- 新增开源治理 P0 基线文档与模板：`SECURITY.md`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`CHANGELOG.md`、版本兼容说明文档。
- 新增 GitHub 协作模板：Issue 模板（bug/feature）与 PR 模板，收敛最小评审清单（测试、文档同步、兼容性影响、breaking change 标记）。
- 调整 CI workflow 基线：固定 `golangci-lint` 版本、去除重复仓库卫生检查、补充最小 `permissions` 与 `timeout-minutes`。
- 对齐 README 与 `docs/development-roadmap.md` 的开源 P0 口径，形成单一事实源。

## Capabilities

### New Capabilities
- `open-source-governance-baseline`: 定义开源项目的最小治理资产与协作流程，包括版本兼容承诺、安全响应入口、贡献与评审闭环。

### Modified Capabilities
- `go-quality-gate`: 增补 CI 工作流治理要求（固定关键工具版本、最小权限、超时保护、避免重复质量门禁步骤）。

## Impact

- Affected docs:
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/versioning-and-compatibility.md`（new）
  - `SECURITY.md`（new）
  - `CONTRIBUTING.md`（new）
  - `CODE_OF_CONDUCT.md`（new）
  - `CHANGELOG.md`（new）
- Affected repository templates:
  - `.github/ISSUE_TEMPLATE/*`（new）
  - `.github/pull_request_template.md`（new）
- Affected CI:
  - `.github/workflows/ci.yml`
- Runtime APIs, behavior, and package contracts: no functional changes.
