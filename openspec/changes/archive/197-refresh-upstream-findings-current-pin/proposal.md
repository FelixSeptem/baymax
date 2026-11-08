## Why

The roadmap names an upstream finding refresh because the repository's formal finding record is absent while the evidence discussion refers to two incompatible Baymax pins. A prior pin must not be presented as a current-version reproduction, and a provider live-smoke that lacks endpoint/model/key configuration must remain `skipped-unconfigured` rather than being misclassified as a Baymax defect or success.

## What Changes

- Add a versioned upstream finding record anchored to Baymax pin `v0.0.0-20261103091945-d2e08cd6dc8c`, with the older `v0.0.0-20261022053159-3d0ab5949790` retained only as historical provenance.
- Add a contract evidence matrix covering child, parallel/vote team, source history, mailbox lifecycle, native cancellation/timeout, causal retry, and provider live-smoke status.
- Record deployment assumptions, evidence strength, unresolved capability limits, and the rule that source history/projection atomicity and path-specific continuation must not be overclaimed.
- Add deterministic PowerShell and shell checks for pin consistency, required matrix rows, limitation markers, and `skipped-unconfigured` live-smoke handling.
- Update the roadmap, README, documentation index, and upstream evidence navigation to point at the current-pin record.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is a documentation/evidence governance change and does not alter runtime behavior, public APIs, or existing OpenSpec requirements.

## Impact

- New evidence records under `docs/upstream/` and `docs/baymax-contract-evidence.md`.
- New offline contract gate under `scripts/` with PowerShell and shell entrypoints.
- Public documentation and roadmap status/navigation updates.
- No Go package, provider SDK, runtime configuration, diagnostic schema, or execution ownership changes.

## Example Impact Assessment

无需示例变更（附理由）

理由：本 change 只刷新外部 evidence、版本 provenance 和文档导航；不改变运行时行为、配置语义、示例输出或 capability declaration，因此现有 examples 不需要语义更新。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/baymax-contract-evidence.md`、`docs/upstream/baymax-findings.md`
- 维护 owner：upstream findings owner、runtime contracts owner、documentation maintainer
- 验证证据：`pwsh -File scripts/check-upstream-findings-current-pin-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/baymax-contract-evidence.md` | runtime contracts owner | current-pin matrix gate |
| components | 修改文档 | `docs/upstream/baymax-findings.md` | upstream findings owner | pin/provenance checks |
| configuration | 修改文档 | `docs/upstream/baymax-findings.md` | upstream findings owner | live-smoke skipped-unconfigured marker |
| contract/API | 修改文档 | `docs/baymax-contract-evidence.md` | runtime contracts owner | capability limitation matrix |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | dedicated evidence gate |
| examples | 无需文档变更（附理由） | examples unchanged | examples/documentation maintainer | no runtime or example semantic change; evidence-only update |
| CLI/integration | 修改文档 | `README.md` | documentation maintainer | navigation and status checks |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap status gate |
