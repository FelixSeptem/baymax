## Why

Workflow resume currently persists a newly reached pause checkpoint before committing the predecessor's resume result. A crash between those writes leaves a retained but unreachable successor and a predecessor classified as recovery-required, so the source cannot return the checkpoint needed to continue the original graph. The existing composite contract already requires atomic successor lineage; runtime evidence now shows the workflow owner does not meet it.

## What Changes

- Publish a resumed workflow's successor checkpoint and predecessor resume projection as one workflow-store transition.
- Reject mismatched successor lineage before commit and preserve the original run, graph, authority, budget, step/effect state, and generation.
- Make store failure and crash-window outcomes explicit and fail closed; do not auto-run or dispatch a replacement workflow.
- Keep child, team, mailbox, and resumed-plan claims independent; this change does not promote their continuation capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow-graph-composability`: A repeated workflow pause must atomically publish a reachable successor and the predecessor's retained resume projection.

## Impact

- Code: `orchestration/workflow/continuation.go` and its continuation tests.
- Public docs: `orchestration/workflow/README.md`, `docs/development-roadmap.md`, and `docs/mainline-contract-test-index.md`.

## Example Impact Assessment

无需示例变更（附理由）：this changes source-owned checkpoint publication only and does not add or alter an end-user example flow.

## Documentation Impact Assessment

| Area | Outcome | Maintainer | Verification | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | No module boundary changes. |
| components | 修改文档 | workflow owner | `go test ./orchestration/workflow -count=1` | `orchestration/workflow/README.md` records atomic publication. |
| configuration | 修改文档 | runtime configuration owner | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/runtime-config-diagnostics.md` records that no new key is introduced. |
| contract/API | 修改文档 | contract-test index owner | `pwsh -File scripts/check-quality-gate.ps1` | `docs/mainline-contract-test-index.md` maps the new gate. |
| diagnostics | 修改文档 | runtime diagnostics owner | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/runtime-config-diagnostics.md` records that no diagnostic field is introduced. |
| examples | 修改文档 | native-events example owner | `go test ./orchestration/workflow -count=1` | `examples/native-execution-events/README.md` states the example remains outside owner commit. |
| CLI/integration | 无需文档变更（附理由） | — | — | No CLI or integration surface changes. |
| best practices | 无需文档变更（附理由） | — | — | No usage guidance changes beyond the workflow owner README. |
| roadmap | 修改文档 | roadmap maintainer | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/development-roadmap.md` records the audited implementation direction. |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/workflow/README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md`, `examples/native-execution-events/README.md`
- 维护 owner：workflow owner, roadmap maintainer, contract-test index owner, runtime configuration/diagnostics owner, native-events example owner
- 验证证据：`go test ./orchestration/workflow -count=1`; `pwsh -File scripts/check-docs-consistency.ps1`; `pwsh -File scripts/check-quality-gate.ps1`

## Risk and Rollback

A partial multi-record write could make a checkpoint appear committed without its predecessor result. Rollback reverts the atomic owner transition and keeps this capability fail-closed; it must never fall back to re-running the workflow.
