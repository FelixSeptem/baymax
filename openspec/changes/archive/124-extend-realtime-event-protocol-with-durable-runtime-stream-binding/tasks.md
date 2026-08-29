## Example Impact Assessment

修改示例

The existing `realtime-interrupt-resume` example must be updated documentation-first to show bounded subscription, cursor catch-up, live-tail handoff, disconnect recovery, cursor expiry, backpressure classification, and rollback to the source-owned event path.

## 1. Binding Contract Model and Validation

- [x] 1.1 Add additive, versioned event-stream subscription, opaque cursor, bounded filter, delivery-policy, phase/outcome, and deterministic reason DTOs in the existing cross-module protocol owner.
- [x] 1.2 Validate subscription identifiers, start modes, cursor/filter/batch bounds, delivery-policy/outcome compatibility, and stable source/binding reason classification without source state mutation.
- [x] 1.3 Add pure normalization helpers for source-owned catch-up history, live-tail handoff, overlap deduplication, cursor expiry, disconnect recovery, and sequence-gap classification without creating a queue, event store, or transport runtime.
- [x] 1.4 Add positive, negative, boundary, and backward-compatibility unit coverage for latest/after-cursor starts, invalid requests, expired/unresolved history, incompatible backpressure outcomes, duplicate overlap, and missing handoff correlation.

## 2. Realtime and Protocol Projection

- [x] 2.1 Add opt-in Realtime source integration that projects existing cursor, sequence, deduplication, interrupt/resume, history availability, and source-owned delivery outcomes without changing their owners.
- [x] 2.2 Add additive Agent Runtime Protocol mapping for subscription correlation, binding decision/phase/reason, cursor mode, and available sequence boundary while preserving nullable v1 fields.
- [x] 2.3 Preserve Run/Stream semantic parity for equivalent subscription, history, cursor, delivery policy, and source outcomes; add parity tests for catch-up-to-live, duplicate overlap, expiry, gap, disconnect, and backpressure.
- [x] 2.4 Add source-owner and control-plane-absence tests proving no listener, connection manager, hosted event/session service, external event store, global queue, synthetic cursor, or parallel interrupt/resume state machine is introduced.

## 3. Diagnostics, Replay, and Contract Gates

- [x] 3.1 Add nullable bounded stream-binding correlation fields to diagnostics and OTel projection through `observability/event.RuntimeRecorder` only; prohibit cursor bodies, event payloads, and arbitrary subscriber metadata from high-cardinality attributes.
- [x] 3.2 Extend deterministic replay fixtures with accepted/latest, after-cursor catch-up, overlap deduplication, live-tail handoff, cursor expiry, history unresolved, sequence gap, disconnect recovery, backpressure mismatch, and missing subscription correlation cases.
- [x] 3.3 Extend diagnostics replay parsing and drift taxonomy for binding phase/reason/correlation changes while preserving existing v1 fixture compatibility.
- [x] 3.4 Extend the Realtime and Agent Runtime Protocol contract gates plus shell/PowerShell parity checks for binding validation, handoff, retention, disconnect, backpressure, Run/Stream parity, `RuntimeRecorder` single-write, and control-plane/queue absence.
- [x] 3.5 Update `docs/mainline-contract-test-index.md` with fixture, replay, and gate-to-requirement mappings.

## 4. Example and Documentation Delivery

- [x] 4.1 **Example Impact Assessment: 修改示例** - update `examples/agent-modes/MATRIX.md` and `realtime-interrupt-resume/README.md` first with semantic anchor, runtime path, expected markers, reconnect/catch-up/handoff behavior, expiry/backpressure behavior, and rollback notes.
- [x] 4.2 Update the `realtime-interrupt-resume` implementation and example contract tests only after the documentation baseline is complete.
- [x] 4.3 Add example smoke coverage for latest, cursor catch-up, live-tail handoff, overlap deduplication, expired cursor, disconnect recovery, and backpressure classifications while retaining interrupt/resume behavior.
- [x] 4.4 Update README, runtime configuration/diagnostics documentation, module-boundary notes, and roadmap status/next-step references without advertising a transport gateway or hosted service.

## 5. Verification and Delivery

- [x] 5.1 Run affected package and integration suites, including positive, negative, boundary, replay, control-plane-absence, and Run/Stream parity cases.
- [x] 5.2 Run `go test ./...` and `go test -race ./...`.
- [x] 5.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 5.4 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`, confirming no roadmap status drift or missing/invalid Example Impact Assessment declaration.
- [x] 5.5 Review library-first boundary, additive compatibility, source ownership, rollback, diagnostics cardinality, and control-plane/queue absence evidence before marking the change ready for archive.
