## Why

当前仓库已具备治理基线文档与模板，但现有策略更偏“稳定期开源项目”口径，与当前 `pre-1.x` 个人项目阶段不一致：兼容性承诺与安全响应时限会放大维护负担并制造外部预期偏差。需要在不改 runtime 能力的前提下，收敛为轻承诺策略，并把贡献入口改为阻断级必填门禁。

## What Changes

- 回调版本治理策略：保留 SemVer 版本表达，但明确 `pre-1.x` 阶段不提供兼容性承诺，不要求 breaking-change 迁移承诺。
- 调整维护窗口：文档声明仅维护最近 `minor` 分支。
- 回调安全披露策略：`SECURITY.md` 使用邮箱私报通道，移除响应时限（SLA）承诺，仅保留 best-effort 流程。
- 强化贡献与评审模板：PR/Issue 模板切换为中文优先、接受英文，并将关键字段设为必填。
- 增加阻断级门禁：在 CI 中新增/收敛模板完整性检查，缺失必填项时直接失败，作为 required check 使用。

## Capabilities

### New Capabilities

- `contribution-template-enforcement`: 对 Issue/PR 必填模板项提供仓库内可执行的阻断校验能力。

### Modified Capabilities

- `open-source-governance-baseline`: 将“稳定期承诺”改为 `pre-1.x` 轻承诺策略，调整安全通道与贡献流程口径。
- `go-quality-gate`: 扩展质量门禁范围，纳入贡献模板必填项阻断校验。

## Impact

- 主要影响文档与治理资产：`docs/versioning-and-compatibility.md`、`SECURITY.md`、`CONTRIBUTING.md`、`.github/ISSUE_TEMPLATE/*`、`.github/pull_request_template.md`、CI/workflow 与相关校验脚本。
- 不涉及 runtime 行为与对外 API 改动。
- 对协作流程有直接影响：外部贡献提交前需满足模板必填约束，否则 CI 阻断。
