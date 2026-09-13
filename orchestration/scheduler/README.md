# orchestration/scheduler

`orchestration/scheduler` owns durable task, attempt, lease, retry, and terminal-commit state. The scheduler is the source of truth for task lifecycle and stale-attempt protection; it is not a workspace manager, completion queue, or diagnostics store.

## Ownership and boundaries

- `Task`, `Attempt`, lease generation, retry/rollover, and terminal commit idempotency remain scheduler-owned.
- `WorkspaceProvenance` is an optional, reference-only association. The scheduler validates that task, attempt, checkpoint, and terminal commit references agree, but never resolves, reads, mutates, or persists workspace contents.
- Mailbox owns durable completion delivery and correlation. `core/runner` owns runtime-input admission and safe-point application. A completion reference may be admitted to that existing lane, but scheduler does not create a second queue or terminal state machine.
- Snapshot/recovery validates and reconciles persisted scheduler references before mutation. `orchestration/composer` remains orchestration glue and does not become a scheduler or workspace owner.
- Diagnostics are emitted through standard events and `observability/event.RuntimeRecorder`; this package never writes `runtime/diagnostics` directly.

## Compatibility and privacy

Workspace references are additive, nullable, and defaultable. Legacy tasks and snapshots without `workspace_provenance` retain existing behavior and are interpreted as `binding absent`. Retry must be explicit: `reuse_after_integrity_validation` or `rebind_with_rollover`; omitted or unknown modes are not inferred.

Only bounded identifiers, change-set/checkpoint references, integrity classifications, lease/attempt correlation, and stable reason codes are observable. Workspace paths, file lists, content, credentials, completion bodies, and model reasoning are never stored in scheduler records, replay fixtures, or diagnostics.

## Configuration

This contract adds no runtime configuration keys. Existing `scheduler.*`, `mailbox.*`, `recovery.*`, and `runtime.state.snapshot.*` settings retain the fixed `env > file > default` precedence and current defaults. Binding validation is driven by supplied references and fixture policy, not by a new scheduler toggle or hot-update domain.

## Recovery and rollback

Strict snapshot recovery rejects invalid, stale, conflicting, dirty, missing, or integrity-drifted associations before mutation. Compatible recovery may apply only the existing bounded downgrade policy and records its classification. Repeated imports and completion promotions are idempotent and never resurrect a closed Run.

Rollback removes the additive workspace/completion reference projections, replay fixtures, and gates. Existing task/attempt/lease lifecycle, mailbox delivery, runtime-input safe points, and terminal outcomes remain usable without data migration.

## Non-goals

No Git/worktree manager, runtime Git or shell execution, hosted workspace/artifact service, automatic merge/push, global queue, notification service, or parallel task/session/coordination state machine is introduced.

## Verification

- `go test ./orchestration/scheduler -count=1`
- `go test ./orchestration/scheduler -run 'Workspace|Snapshot|Recovery|Commit' -count=1`
- `go test ./core/runner -run 'CompletionReference|RuntimeInput' -count=1`
- `go test ./tool/diagnosticsreplay -run 'DurableAttemptWorkspaceBinding|CompletionSafePointOwnership' -count=1`
- `pwsh -File scripts/check-durable-attempt-completion-replay-contract.ps1`
