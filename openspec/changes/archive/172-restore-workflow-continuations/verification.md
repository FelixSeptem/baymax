# Verification Evidence

Date: 2026-10-21. Branch: `propose/restore-workflow-continuations`.

## Delivered scope

Workflow is now the only composite form opened by this change for source-owned
continuation. The implementation persists a bounded graph continuation, supports
`input_required`, `approval_required`, and `budget_exhausted` safe points, and
restores the same native workflow graph after a configured FileStore is reopened
by a new Engine. Atomic resume claims converge duplicate requests to one
projection and reject foreign, stale, conflicting, expired, corrupt, unavailable,
and in-progress states with typed outcomes. Completed steps and effects are not
redispatched. Child/fanout, team parallel/vote, mailbox host plans, and planbridge
resumed execution remain explicitly `unsupported_capability`.

The deployment boundary remains same-host, single-writer, sequential reopen.
This change does not provide multi-writer CAS, cross-machine fencing, a
distributed journal, multi-instance claim coordination, or a second executor.

## Focused verification

| Command | Result | Evidence |
| --- | --- | --- |
| `go test ./orchestration/workflow ./orchestration/planbridge ./tool/diagnosticsreplay -count=1` | Passed | Workflow DTO/store/restore tests, capability matrix tests, and reference-only replay tests. |
| `go test -race ./...` | Passed | Full race coverage, including workflow continuation restore and duplicate-claim paths. |
| `pwsh -File scripts/check-workflow-continuation-contract.ps1` | Passed | Positive, negative, boundary, crash-window, Run/Stream parity, unsupported-form, and intentional replay-drift cases. |
| `openspec validate restore-workflow-continuations --strict` | Passed | Proposal, design, five spec deltas, and task artifact validation. |
| `pwsh -File scripts/check-openspec-documentation-impact.ps1` | Passed | Public semantic documentation declarations and changed paths are consistent. |
| `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` | Passed | Example impact declaration is present and valid. |
| `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` | Passed | Roadmap has no stale active entry after all tasks completed. |
| `pwsh -File scripts/check-docs-consistency.ps1` | Passed | Documentation links and declared paths are consistent. |
| `pwsh -File scripts/check-go-file-line-budget.ps1` | Passed | Workflow implementation remains within the repository file budget. |
| `golangci-lint run --config .golangci.yml` | Passed | `0 issues`. |
| `git diff --check` | Passed | No whitespace errors. |

## Full quality gate

`pwsh -File scripts/check-quality-gate.ps1` completed the full test, race,
lint, contract, documentation, performance, and example-smoke chain. Functional
and performance stages passed, including the multi-agent performance regression
gate and full-chain example smoke. The final strict vulnerability step exited
with code 3 because the installed Go standard library `crypto/x509@go1.26.9`
is affected by external advisory `GO-2026-6809` (fixed in Go 1.26.10). This is
an environment/toolchain dependency finding, not a workflow continuation
regression; the quality gate was not weakened and the advisory is recorded as
an outstanding release prerequisite.

## Replay and side-effect boundary

The workflow continuation fixtures are bounded and reference-only. Replay does
not construct a workflow owner, claim a checkpoint, invoke Run or Stream, retry,
enqueue, call a model or tool, or mutate source/diagnostic state. Callback event
IDs and application projections are not treated as restore material.

## Rollback

Rollback withdraws workflow checkpoint/restore capability advertisement and
retains persisted records for inspection. It must not reinterpret a checkpoint
as a fresh parent Run, latest replay, or a new enqueue. Archive and later merge
remain separate lifecycle actions and have not been performed in this handoff.
