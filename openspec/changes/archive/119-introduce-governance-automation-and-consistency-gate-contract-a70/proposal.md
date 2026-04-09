## Why

当前治理存在两个可重复出现的风险：

- `docs/development-roadmap.md` 与 OpenSpec 实际状态（`openspec list --json` + `openspec/changes/archive/INDEX.md`）容易产生状态漂移，人工对账成本高且容易漏检。
- “提案是否需要新增/修改 example”的要求已在 A62 中固化，但还缺少仓库级自动化校验，后续提案仍可能出现声明缺失或口径不一致。

这类问题不属于 runtime 新能力建设，而是治理执行自动化缺口。A70 的目标是把这两类规则变成可执行、可阻断、可审计的门禁合同。

## What Changes

- 新增 A70 主合同：`governance automation and consistency gate`。
- 固化状态对账单一事实源：
  - 活跃状态以 `openspec list --json` 为准；
  - 归档状态以 `openspec/changes/archive/INDEX.md` 为准；
  - `docs/development-roadmap.md` 必须与上述事实源一致。
- 新增 roadmap/open spec 状态一致性校验脚本：
  - `check-openspec-roadmap-status-consistency.sh/.ps1`
  - 对状态漂移输出确定性分类并阻断合并。
- 新增提案 `example impact assessment` 声明校验脚本：
  - `check-openspec-example-impact-declaration.sh/.ps1`
  - 允许值固定：`新增示例`、`修改示例`、`无需示例变更（附理由）`。
- 将上述校验接入 `check-quality-gate.*` 与 `check-docs-consistency.*`，保持 shell/PowerShell parity。
- 新增治理文档映射，明确“提案声明 -> 门禁脚本 -> CI required-check 候选”的追踪关系。

## Example Impact Assessment

- 无需示例变更（附理由）：本提案仅新增治理自动化脚本、门禁接线与文档追踪映射，不引入运行时行为变更。

## Capabilities

### New Capabilities

- `proposal-governance-automation-contract`: 统一定义 proposal 影响声明与 roadmap/open spec 状态一致性自动化治理。

### Modified Capabilities

- `go-quality-gate`: 扩展治理自动化阻断步骤，新增状态一致性与 example-impact 声明校验。

## Impact

- 脚本与门禁：`scripts/check-openspec-roadmap-status-consistency.*`、`scripts/check-openspec-example-impact-declaration.*`、`scripts/check-quality-gate.*`、`scripts/check-docs-consistency.*`。
- 文档与索引：`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`AGENTS.md`（提案协作约束映射）。
- OpenSpec 工件：后续变更的 `proposal/design/tasks` 需满足统一声明口径。
- 不涉及 runtime 行为语义或 API 变更。
