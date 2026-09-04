## Why

Baymax now has stable protocol references, checkpoint/workspace provenance, durable event-stream recovery, and tool terminal-outcome semantics, but it does not yet define the boundary between session message history, run checkpoints, snapshot restore, and diagnostics replay. Without that boundary, branch/fork creation, restore idempotency, schema migration, and offline replay can produce ambiguous ownership or silently diverging lineage. This P2 change freezes those relationships before additional host integrations depend on them.

## What Changes

- Introduce a library-first session history/checkpoint/replay boundary contract that defines ownership, correlation, and lifecycle relationships across Session, Run, Step, Event, Checkpoint, Snapshot, and replay records.
- Define message-history chain integrity and leaf-based branch/fork semantics without creating a second session repository or replacing the existing snapshot fact source.
- Define checkpoint/history consistency checks, schema-version migration behavior, `strict | compatible` restore handling, atomic writes, duplicate restore behavior, and deterministic conflict classifications.
- Define offline diagnostics replay as a read-only, source-owned projection that never re-executes side effects or mutates business state.
- Preserve existing Realtime interrupt/resume, durable stream binding, tool lifecycle, terminal outcome, policy, sandbox, and Run/Stream parity semantics; this change only adds cross-boundary association and validation.
- Add canonical valid, negative, and drift fixtures plus integration, replay, shell/PowerShell gate, and documentation coverage.
- Keep all new public fields additive, nullable or defaultable, versioned, and recorded through `observability/event.RuntimeRecorder` where diagnostics are emitted.

## Example Impact Assessment

新增示例

Add a documented session/checkpoint branch and offline replay example with deterministic lineage and side-effect-free replay assertions.

## Capabilities

### New Capabilities

- `session-history-checkpoint-replay`: Defines session history ownership, checkpoint/history association, branch/fork lineage, restore and replay boundaries, conflict taxonomy, and Run/Stream-normalized outcomes.

### Modified Capabilities

- `agent-runtime-protocol-contract`: Additive Session/Run/Step/Checkpoint lineage and branch/replay association requirements.
- `unified-state-and-session-snapshot-contract`: Define how snapshot restore participates in history/checkpoint validation without changing manifest ownership or restore modes.
- `checkpoint-history-and-workspace-provenance`: Extend existing lineage references with session-history branch and replay-boundary validation.
- `diagnostics-replay-tooling`: Add session-history/checkpoint replay fixtures, side-effect prohibition, and deterministic drift classifications.

## Impact

- Affected areas: `core/types`, `core/runner`, `runtime/diagnostics`, `orchestration/snapshot`, protocol projection/mapping code, and diagnostics replay tooling.
- New contract fixtures and gates will be added under `tool/diagnosticsreplay` and `scripts`, with shell/PowerShell semantic parity.
- Documentation updates are required for `README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and `docs/mainline-contract-test-index.md`.
- No new external service, hosted control plane, remote gateway, session database, artifact content service, or provider SDK dependency is introduced.
- Existing consumers remain compatible through additive/defaultable fields; any incompatible history/checkpoint input fails fast with a deterministic classification and leaves the source state unchanged.
