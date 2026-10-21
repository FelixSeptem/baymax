# Verification

## Scope

This change adds source-owned retained history for explicitly configured
runner, isolated child, team, and workflow journals, while preserving the
existing mailbox owner and real-time callback adapter boundaries. It does not
add checkpoint restore, composite graph recovery, a scheduler, a queue, or a
distributed journal.

## Commands

The following completed successfully on the implementation branch:

```text
go test ./core/history ./core/types ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge ./tool/diagnosticsreplay ./integration -count=1
go test ./... -count=1
go test -race ./...
golangci-lint run --config .golangci.yml ./...
pwsh -NoProfile -File scripts/check-native-event-history-contract.ps1
pwsh -NoProfile -File scripts/check-docs-consistency.ps1
pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1
pwsh -NoProfile -File scripts/check-openspec-example-impact-declaration.ps1
pwsh -NoProfile -File scripts/check-openspec-roadmap-status-consistency.ps1
openspec validate retain-run-and-composite-source-history --strict
git diff --check
```

The repository-wide `scripts/check-quality-gate.ps1` was also executed with
the repository-local `GOCACHE` and passed. This includes the source-history,
host-plan, continuation, full Go, race, lint, documentation, contract,
performance-regression, example-smoke, and vulnerability checks. The
multi-agent benchmark initially exposed an allocation regression in the new
workflow timeline path; the default no-history/no-emitter fast path was then
guarded so native-history projection is only assembled when explicitly
configured. The follow-up performance gate passed with synchronous invocation
allocations within the configured threshold.

Go tests used repository-local `GOCACHE` because the default user cache was not
readable in this environment.

## Evidence

- `core/history` provides source-local sequence, bounded retention, floor,
  opaque cursor validation, file reopen, redaction, and atomic rollback.
- Runner, composer child, team, and workflow owners bind an already-created
  journal explicitly; default constructors remain unsupported until configured.
- `MergeNativeExecutionEvents` deduplicates identical overlap and returns typed
  `ErrHistoryConflict` for changed facts.
- Fixture replay remains reference-only and the shell/PowerShell gate checks
  source identity, ordering, retention, typed errors, restart cursor and
  no-side-effect ownership.

## Risks and rollback

The supported persistence boundary is same-host single-writer sequential
reopen. Multi-writer CAS, cross-machine fencing, distributed cursor
transactions, and multi-instance resume claims remain unsupported. A rollback
withdraws the affected manifest source-history cell or stops injecting the
owner journal; it must not replay latest, rerun a parent, or delete source
facts.

## Unexecuted or environment-limited checks

The full `check-quality-gate.ps1` / `.sh` chain was not used as the source of
feature correctness because it includes repository-wide performance and
environment-dependent checks. The focused history gate, full Go test, race
test, lint, documentation, OpenSpec, and diff checks above are the evidence
for this change. Any repository-wide quality-gate failure must be reported
without weakening the source-history contract.
