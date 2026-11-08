## Context

The repository currently has a roadmap entry for an upstream finding refresh but no checked-in finding record at the paths referenced by the evidence discussion. The refresh must therefore establish a durable evidence source without implying that an upstream observation is a Baymax runtime guarantee. The current pin is `v0.0.0-20261103091945-d2e08cd6dc8c`; the prior finding pin `v0.0.0-20261022053159-3d0ab5949790` remains historical only.

## Goals / Non-Goals

**Goals:**

- Publish one current-pin finding record and one Baymax contract evidence matrix with explicit provenance and evidence class.
- Preserve the distinction between execution wiring, continuation restore, source replay, projection reconciliation, and mailbox lifecycle replay.
- Record same-host/single-writer or same-host multi-instance deployment assumptions where the existing evidence requires them.
- Make live-smoke configuration prerequisites and `skipped-unconfigured` classification machine-checkable.

**Non-Goals:**

- No runtime implementation, API, capability promotion, planner fallback, provider request, or configuration default change.
- No claim that current-pin evidence proves cross-machine durable replay, distributed CAS, cross-project atomicity, or all-form fresh-process restore.
- No external network or provider call in the offline gate; live-smoke remains explicitly unconfigured until endpoint/model/key are supplied.

## Evidence Model

`docs/upstream/baymax-findings.md` is the provenance-facing record. Each finding names the current pin, evidence class, deployment assumptions, observed result, and unresolved boundary. `docs/baymax-contract-evidence.md` is the Baymax-facing matrix that maps each execution form to execution, identity, aggregation/terminal, continuation, source-history, and live-provider evidence. A row marked `not-verified` or `skipped-unconfigured` cannot promote a capability.

The gate checks exact current/legacy pin markers, required rows, and negative limitation markers. It does not infer findings from package names, old archive text, callback IDs, or successful direct execution.

## Rollback

Rollback is documentation-only: revert the evidence records, navigation links, and gate wiring. No runtime state or persisted source facts are changed.

## Example Impact Assessment

无需示例变更（附理由）

理由：仅新增 evidence 文档和离线 governance gate；没有 example runtime path、marker、输出或配置语义变化。

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
