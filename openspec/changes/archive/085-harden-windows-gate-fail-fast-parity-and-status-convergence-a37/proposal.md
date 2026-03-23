## Why

仓库当前已进入 A35 归档、A36 实施阶段，主风险从“能力缺失”转向“门禁可信度”。  
现状中 PowerShell 门禁脚本存在失败传播不严格的问题（示例：`check-docs-consistency.ps1` 在测试失败后仍输出通过），叠加 README/roadmap 状态口径漂移，会削弱收敛阶段的可回归性与发布信心。

## What Changes

- 收敛 Windows (`.ps1`) 门禁执行语义，统一 native command 失败传播为 fail-fast（非零退出即阻断）。
- 修复 `scripts/check-docs-consistency.ps1` 的失败传播路径，确保状态口径冲突时脚本 deterministic non-zero。
- 统一 shared/quality gate 在 shell 与 PowerShell 下的阻断语义，避免“Linux 阻断、Windows 误通过”的不一致。
- 补齐门禁级契约测试与回归用例，覆盖失败传播、状态口径漂移阻断、跨平台语义一致性。
- 同步 README 与 roadmap 的 active/archived 状态口径到 OpenSpec authority（`openspec list --json` + `archive/INDEX.md`）。

## Capabilities

### New Capabilities

- `powershell-gate-fail-fast-governance`: 规范并验证 PowerShell 门禁脚本的 fail-fast 失败传播与跨脚本一致执行语义。

### Modified Capabilities

- `go-quality-gate`: 强化 cross-platform gate parity，要求关键 `.ps1` 路径对 native command 失败 deterministic non-zero。
- `release-status-parity-governance`: 增强状态口径收敛约束，要求 docs consistency gate 对 authority 冲突强阻断。

## Impact

- 代码：
  - `scripts/check-docs-consistency.ps1`
  - `scripts/check-quality-gate.ps1`
  - `scripts/check-multi-agent-shared-contract.ps1`
  - `scripts/check-adapter-*.ps1`、`scripts/check-security-*.ps1`（按范围收敛）
  - `tool/contributioncheck/*`（状态口径与门禁契约测试）
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 无新增运行时 API；
  - 变更集中在门禁与文档状态治理，属于 fail-fast 收敛增强。
