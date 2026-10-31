## 1. Reproduce and implement atomic publication

- [x] 1.1 Add a failing file-store test that pauses twice, injects failure between current successor save and predecessor commit, reopens the store, and proves no successor is reachable from the predecessor; run `go test ./orchestration/workflow -run TestWorkflowSuccessorPublicationIsAtomic -count=1` and confirm failure on the current implementation.
- [x] 1.2 Add a successor-aware commit operation to the workflow continuation owner store and validate predecessor, successor, generation, graph, run/session, and authority bindings before mutation; verify positive, conflicting, and malformed-store tests.
- [x] 1.3 Make memory and file stores commit predecessor result and successor record atomically; verify persistence failure leaves the pre-commit snapshot intact and fresh reopen sees either the old state or full transition.
- [x] 1.4 Update `ResumeContinuation` to use one atomic owner transition for repeated pauses and to return typed failure without exposing an uncommitted successor; verify duplicate resume returns the same successor projection without redispatch.

## 2. Contract and documentation evidence

- [x] 2.1 Add fresh-store-reopen coverage for successor follow-up resume, preserving original graph/run/session/causation and ensuring completed steps/effects do not rerun; verify terminal Run/Stream parity remains unchanged.
- [x] 2.2 Update `orchestration/workflow/README.md`, `docs/development-roadmap.md`, and `docs/mainline-contract-test-index.md` with the atomic successor-publication guarantee, failure behavior, same-host single-writer scope, test owner, and validation command.
- [x] 2.3 Run focused workflow tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; archive only after all pass.

## Example Impact Assessment

无需示例变更（附理由）：the owner-store transaction is internal to continuation persistence; no example behavior changes.

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
