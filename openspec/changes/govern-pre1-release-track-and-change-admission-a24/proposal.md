## Why

roadmap 已从 `1.0.0` 收口叙事回切到 `0.x` 持续迭代，但当前仓库缺少可执行的“pre-1 阶段治理约束”，容易在后续提案中再次出现版本口径与发布预期漂移。A24 目标是把 `0.x` 阶段的发布口径与提案准入标准固化为契约与门禁。

## What Changes

- 新增 `0.x` release-track 治理能力：定义 pre-1 阶段的 roadmap 必填口径、提案准入标准与长期方向边界。
- 统一“非 1.0/prod-ready 承诺”文档语义，要求 roadmap 与版本策略文档一致表达 pre-1 兼容姿态。
- 在质量门禁中增加 pre-1 口径一致性检查（roadmap/versioning/README 关键段落一致性）。
- 建立“新增提案准入说明”模板要求：`Why now`、风险、回滚、文档影响、验证命令。
- 保持 lib-first 边界：不引入平台化能力或 runtime 行为变更。

## Capabilities

### New Capabilities
- `pre1-release-track-governance`: 定义 `0.x` 阶段的发布口径、提案准入标准、长期方向边界和文档同步要求。

### Modified Capabilities
- `open-source-governance-baseline`: 增加 pre-1 治理口径跨文档一致性要求，避免出现隐式 `1.0/prod-ready` 承诺漂移。
- `go-quality-gate`: 增加 pre-1 口径一致性检查为标准阻断验证路径。

## Impact

- 文档：
  - `docs/development-roadmap.md`
  - `docs/versioning-and-compatibility.md`
  - `README.md`（如需同步版本阶段快照表述）
- 门禁与测试：
  - `scripts/check-docs-consistency.sh`
  - `scripts/check-docs-consistency.ps1`
  - `tool/contributioncheck/*`（新增或扩展 pre-1 口径一致性测试）
- OpenSpec：
  - `openspec/specs/pre1-release-track-governance/spec.md`（新增）
  - `openspec/specs/open-source-governance-baseline/spec.md`（修改增量）
  - `openspec/specs/go-quality-gate/spec.md`（修改增量）
