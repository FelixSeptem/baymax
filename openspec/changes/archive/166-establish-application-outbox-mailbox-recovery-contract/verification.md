# Verification — Application Outbox Mailbox / Scheduler Recovery Contract

Verified on 2026-10-18 in worktree `D:\code\baymax\.worktrees\application-outbox-mailbox-recovery` on branch `feature/application-outbox-mailbox-recovery`.

## Focused contract evidence

| Command | Result |
| --- | --- |
| `$env:GOCACHE = (Join-Path (Get-Location) '.gocache-implementation'); go test ./orchestration/outboxbridge ./orchestration/mailbox ./orchestration/scheduler ./tool/diagnosticsreplay ./integration -count=1` | exit 0; validates bridge admission, duplicate and binding convergence, mailbox/scheduler native lifecycle ownership, reference-only reconciliation, diagnostics replay, and Run/Stream source conformance. |
| `$env:GOCACHE = (Join-Path (Get-Location) '.gocache-binding-mismatch'); go test ./orchestration/outboxbridge ./orchestration/mailbox ./orchestration/scheduler ./integration -count=1` | exit 0; regression coverage confirms native mailbox/scheduler binding conflicts map to the documented `outbox.binding_mismatch` classification while preserving the native error for inspection. |
| `$env:GOCACHE = (Join-Path (Get-Location) '.gocache-final-contract'); pwsh -NoProfile -File scripts/check-application-outbox-mailbox-recovery-contract.ps1` | exit 0; final rerun passed bridge, mailbox, scheduler and replay suites after the binding taxonomy and delivered scheduler documentation corrections. |
| `$env:GOCACHE = '.gocache-implementation'; go run ./examples/application-outbox-recovery` | exit 0; emitted `mailbox_dispatch_accepted`, `duplicate_converged`, `scheduler_dispatch_accepted`, `publish_before_outbox_confirmation_reconciled`, `effect_committed_before_target_reconciliation`, and `rollback_preserves_pending_facts`. |
| `$env:GOCACHE = '.gocache-implementation'; pwsh -NoProfile -File scripts/check-application-outbox-mailbox-recovery-contract.ps1` | exit 0; focused mailbox, scheduler, bridge and replay suites reported `ok`, and the contract gate reported `passed`. |
| `bash scripts/check-application-outbox-mailbox-recovery-contract.sh` | unavailable in this Windows environment: WSL cannot start `/bin/bash` (`execvpe(/bin/bash) failed: No such file or directory`, exit 1). The shell script remains present and the PowerShell gate exercises the equivalent suites. |
| `$env:GOCACHE = '.gocache-implementation'; go test ./tool/diagnosticsreplay ./tool/contributioncheck -count=1` | exit 0; replay reference-only and contribution boundary tests passed. |

## OpenSpec and documentation evidence

| Command | Result |
| --- | --- |
| `openspec validate establish-application-outbox-mailbox-recovery-contract --strict` | exit 0; change is valid. |
| `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1` | exit 0; all declared public semantic documentation paths are changed in the feature diff. |
| `pwsh -NoProfile -File scripts/check-openspec-example-impact-declaration.ps1` | exit 0. |
| `pwsh -NoProfile -File scripts/check-openspec-roadmap-status-consistency.ps1` | exit 0. |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | exit 0; documentation consistency and roadmap checks passed. |
| `git diff --check` | exit 0. |

The first direct final contract-gate invocation inherited the restricted global Go
build cache and exited 1 with `Access is denied`; rerunning with the repository-local
absolute `GOCACHE` above passed. This is an environment limitation rather than a
contract failure.

## Archive-precheck completion

On 2026-10-18, task 5.3 rechecked the public API and bounded limits, finite error
taxonomy, mailbox/scheduler support matrix, recovery and rollback wording, and the
three `Public Semantic Documentation Commitment` sections. The commitment sections
in `proposal.md`, `design.md` and `tasks.md` were byte-identical after normalization
(SHA-256 `45bbd85d0f8ca1767f6d8a3318d10211bf6dac9b64844f598934f179165c2b5d`). A
fresh declared-path inspection confirmed all twelve committed documentation paths
exist and are in the branch diff. The planned-delivery wording scan produced no
matches in the affected public documentation. `openspec validate
establish-application-outbox-mailbox-recovery-contract --strict`, the documentation
impact gate, the example-impact gate, `git diff --check`, and the PowerShell contract
gate each exited 0 during the recheck.

## Standard quality evidence

| Command | Result |
| --- | --- |
| `go test ./... -count=1` | exit 0 in the first full implementation run. |
| `go test -race ./...` | exit 0 in the first full implementation run; the quality gate also completed this suite successfully before its initial lint failure. |
| `golangci-lint run --config .golangci.yml` | exit 0 after fixing the three findings surfaced by the initial full gate (`ifElseChain`, De Morgan simplification, and goimports). The root cache emitted only environment-specific write warnings when the cache directory was restricted. |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | exit 0; final rerun completed 79 steps in 1201.16s under the 1800s budget. `go test ./...`, `go test -race ./...`, lint (`0 issues.`), A64 semantic/performance gates, full-chain smoke, and govulncheck completed successfully. The host-execution-plan wrapper reported its pre-existing `TestHostExecutionPlanReplayGoldenAndDrift` fixture mismatch (`host_execution_plan.replay_drift`) but intentionally allowed that known drift and reported the wrapper as passed. |

## Scope, risks, and rollback

- The bridge is constructor-only and opt-in. Mailbox and scheduler retain publish/claim/ack/nack/requeue/delay/lease/retry/terminal/DLQ ownership.
- `Inspect` and `Reconcile` are reference-only observations. They do not acknowledge mailbox records, complete scheduler tasks, execute effects, confirm outbox records, or synthesize Run/Stream business terminal outcomes.
- Host admission, effect idempotency/CAS, business revision and final outbox confirmation remain host-owned. Delivery or execution acceptance is not business commit evidence.
- Recovery remains at-least-once. Mailbox delivery retry, scheduler execution retry and host business-commit retry are separate owners and are not composed by the bridge.
- Rollback stops constructing the bridge and leaves retained outbox, target, attempt and effect facts for host-owned reconciliation; it does not delete or rewrite those facts.

## Generated artifacts

The quality gate refreshes `.artifacts/a64/harnessability-scorecard.json` and `.artifacts/a64/impacted-full-report.json`. These generated outputs are retained as gate evidence and are not manually reset.

Task 5.3 is complete. The public API, limits, error taxonomy, support matrix,
rollback wording and the three matching documentation commitments were rechecked;
all declared documentation paths are present in the branch diff and no planned
wording claims unimplemented delivery. The change is now ready for its separate
archive lifecycle; no archive, commit, merge, push or worktree cleanup is performed
implicitly by this task completion.
