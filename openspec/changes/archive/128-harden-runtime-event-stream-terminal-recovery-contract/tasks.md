## 1. Contract baseline and documentation

## Example Impact Assessment

修改示例

- [x] 1.1 Audit existing Realtime event, durable binding, terminal outcome, diagnostics, and replay owners; record source-of-truth and non-goals in the change notes
- [x] 1.2 Freeze `runtime-event-stream-terminal-recovery` outcome vocabulary, correlation fields, cursor handoff rules, and deterministic drift classifications
- [x] 1.3 Update `examples/agent-modes/MATRIX.md` with the recovery scenario, semantic anchor, runtime path, expected markers, and rollback notes
- [x] 1.4 Update the selected `realtime-interrupt-resume` or equivalent agent-runtime example README with disconnect, catch-up, live-tail dedupe, and terminal-query expectations before code changes
- [x] 1.5 Update README, roadmap, runtime diagnostics documentation, module-boundary documentation, and mainline contract index with the new capability and gate mapping

## 2. Recovery projection and lifecycle

- [x] 2.1 Add a transport-neutral observer recovery result model with `catching_up`, `live`, `disconnected`, `stopped`, `terminal_available`, `expired`, `gap`, and `backpressure` outcomes
- [x] 2.2 Implement source-owned cursor validation and catch-up/live-tail handoff projection without synthesizing history, cursor, sequence, or exactly-once guarantees
- [x] 2.3 Reuse event ID/sequence/deduplication semantics to make catch-up/live overlap idempotent and classify missing progression as `stream_sequence_gap`
- [x] 2.4 Integrate observer stop/disconnect handling without mutating Run state or triggering retry/resume actions
- [x] 2.5 Integrate terminal snapshot/event convergence through the existing terminal arbiter with first-terminal-wins, repeat idempotency, and late-conflict recording
- [x] 2.6 Preserve partial output, completed tool-call facts, and session/run/step correlation across recovery projection
- [x] 2.7 Validate retention expiry and backpressure outcomes as explicit source-owned results with no binding-owned global queue

## 3. Diagnostics, protocol, and replay

- [x] 3.1 Add additive nullable/defaultable diagnostics fields for observer lifecycle, cursor state, terminal correlation, retained facts, and conflict markers
- [x] 3.2 Route all recovery diagnostics through `observability/event.RuntimeRecorder` and keep high-cardinality cursor/causation values out of OTel dimensions
- [x] 3.3 Add `runtime_event_stream_terminal_recovery.v1` canonical fixture covering success, disconnect, stop, catch-up/live handoff, overlap dedupe, gap, expiry, backpressure, cancel, timeout, Provider failure, and terminal conflict
- [x] 3.4 Extend diagnostics replay normalization and stable drift taxonomy for cursor handoff, dedupe, terminal convergence, retained facts, and Run/Stream parity
- [x] 3.5 Add malformed, unsupported-version, retention, gap, duplicate, and hosted-ownership negative fixtures with deterministic classifications
- [x] 3.6 Verify historical fixtures remain parseable and mixed-version replay remains deterministic

## 4. Contract and integration tests

- [x] 4.1 Add unit tests for valid/invalid cursor, handoff boundary, overlap dedupe, sequence gap, expired cursor, and backpressure outcomes
- [x] 4.2 Add unit tests for observer stop/disconnect idempotency and the invariant that terminal Runs never return to `working`
- [x] 4.3 Add unit tests for terminal snapshot-before-event, repeated terminal, late conflict, partial output, and completed tool-call retention
- [x] 4.4 Add Run/Stream parity tests covering reconnect, cancellation, timeout, and Provider stream failure after partial output
- [x] 4.5 Add integration tests proving event stream, terminal query, RuntimeRecorder diagnostics, and replay expose equivalent normalized facts
- [x] 4.6 Add integration tests for source-owned recovery capability absence and library-first boundary enforcement
- [x] 4.7 Add executable example smoke assertions for disconnect, cursor catch-up, live-tail transition, dedupe, and terminal query

## 5. Gates and verification

- [x] 5.1 Add shell and PowerShell `check-runtime-event-stream-terminal-recovery-contract` gates with identical classifications and fail-fast behavior
- [x] 5.2 Wire the new gate into `check-quality-gate.sh/.ps1` and update required-check documentation
- [x] 5.3 Add dependency scans that reject transport listeners, hosted event/session stores, global binding queues, or a second terminal state machine in the recovery path
- [x] 5.4 Run `openspec validate harden-runtime-event-stream-terminal-recovery-contract` and resolve all schema/spec drift
- [x] 5.5 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`
- [x] 5.6 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`
- [x] 5.7 Review staged diff for additive compatibility, module-boundary compliance, example evidence, rollback notes, and absence of proposal-number identifiers
