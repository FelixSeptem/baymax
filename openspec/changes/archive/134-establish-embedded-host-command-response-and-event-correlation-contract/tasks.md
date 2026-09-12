## Example Impact Assessment

修改示例

先修改 `examples/agent-modes/MATRIX.md` 与 `examples/agent-modes/realtime-interrupt-resume/README.md` 的文档基线，再修改该模式代码或勾选任何实现任务。现有 minimal 和 production-ish 变体增加宿主命令关联、运行中控制、HITL reverse request、断连恢复、JSONL framing 和 stdout 纯净 marker，不创建平行 numbered example。

## 1. Documentation-First Baseline and Test Matrix

- [x] 1.1 Update `examples/agent-modes/MATRIX.md` and `examples/agent-modes/realtime-interrupt-resume/README.md` first with the host-protocol semantic anchor, runtime path, expected markers, rollback notes, contracts, replay fixture, and gates; verify the doc-first contributioncheck tests pass before editing example/runtime code.
- [x] 1.2 Update the minimal and production-ish mode README files with command admission/event separation, active control, HITL, disconnect, framing, and output-purity expectations; verify all referenced existing and planned paths are explicit and no task in groups 2-7 is checked before this documentation baseline is complete.
- [x] 1.3 Add failing contract-test cases or fixtures for envelope validation, command/event separation, duplicate/late operations, terminal races, HITL response races, Realtime ingress, disconnect cleanup, JSONL framing, backpressure, and Run/Stream parity; verify the focused tests fail for the intended missing behavior before implementation. Evidence: `red-evidence.md` reconstructs `HEAD=027e904035b19abe0ba1d812ab3c2cb242f36854` in an isolated baseline, injects only the contract tests/fixture, and records six focused suites failing with the expected missing DTO/coordinator/transport/replay implementation signals.

## 2. Host Contract Model and Validation

- [x] 2.1 Add additive versioned host envelope, command kind, response status, stable reason, correlation, reverse-request, and pending-outcome DTOs in `core/types`; verify JSON round-trip and validation tests cover required/optional fields and backward-compatible omission.
- [x] 2.2 Implement finite first-profile command validation for Run start, advertised lifecycle actions, Realtime interrupt/resume, HITL response, event subscription, and Run outcome query; verify unknown version/kind, missing correlation, invalid identifier, mismatched Session/Run, and oversized bounded fields fail before source mutation.
- [x] 2.3 Implement command admission normalization that separates `accepted|rejected|duplicate` from asynchronous progress and terminal outcome; verify tests prove accepted commands do not synthesize completion and rejected commands emit no business terminal event.
- [x] 2.4 Add source-owner and authorization validation helpers that reuse ProtocolDescriptor, readiness, policy, sandbox, durable binding, and terminal recovery classifications; verify action availability never grants authorization and diagnostics history is never consulted as an active registry.

## 3. Source-Owned Active Run Control and Realtime Ingress

- [x] 3.1 Add a per-Engine bounded active Run control registry with unique `run_id` registration, scoped cleanup, lifecycle snapshot, cancellation, and optional Realtime ingress; verify duplicate registration, cleanup after success/error/panic, and multi-Engine isolation tests pass under `go test -race`.
- [x] 3.2 Route Run and Stream through the same registration and first-terminal-wins cleanup path; verify completion-wins, cancel-wins, duplicate cancel, late cancel, timeout, and panic races preserve exactly one authoritative terminal outcome.
- [x] 3.3 Implement advertised action delegation so cancel reaches the source context, resume reaches only a valid input-required source control, and retry creates a causally related source Run/attempt only when supported; verify unsupported and unauthorized actions cause no mutation and the original terminal Run remains immutable.
- [x] 3.4 Add bounded mid-run Realtime interrupt/resume ingress processed at equivalent safe points in Run and Stream; verify valid control, unknown/inactive Run, Session mismatch, sequence gap, duplicate envelope, invalid cursor, full/closed ingress, and disconnect-without-mutation cases.
- [x] 3.5 Add focused concurrency and Run/Stream parity tests for active registration, control admission, Realtime cursor/sequence/dedupe facts, partial output, and terminal classification; verify `go test -race ./core/types ./core/runner -count=1` passes.

## 4. Transport-Neutral Host Coordinator and HITL Bridge

- [x] 4.1 Add the top-level `host` coordinator using injected Runner, source control, event subscription, terminal query, readiness, and authorization interfaces; verify package dependency tests prove it owns only bounded connection correlation and introduces no global registry, event store, terminal state machine, or diagnostics writer.
- [x] 4.2 Implement dispatch for the first-profile command set and emit one immediate admission response plus correlated asynchronous events; verify integration tests cover Run start, action rejection, cancel races, subscription recovery, authoritative terminal query, and source reason preservation.
- [x] 4.3 Implement connection-scoped pending correlation with exactly-once removal on response, timeout, cancellation, output failure, and disconnect; verify duplicate/late/unknown response, shutdown, and concurrent completion tests leave zero pending entries under race detection.
- [x] 4.4 Implement host-mediated `ClarificationResolver` and `ActionGateResolver` adapters using existing RequestID and timeout semantics; verify clarification resumes once, clarification timeout follows `canceled_by_user`, Action Gate transport loss fails closed, and Run/Stream timelines remain equivalent.
- [x] 4.5 Isolate source callbacks from transport writes through bounded per-connection delivery and existing source policies; verify slow or panicking writers cannot block Runner indefinitely, critical frames are never silently dropped, eligible low-priority drops are recorded through standard events, and disconnect does not implicitly cancel unrelated Runs.

## 5. Strict JSONL/Stdio Binding

- [x] 5.1 Add `host/jsonl` raw LF framing, one-object-per-frame decoding, explicit profile negotiation, a 1 MiB default frame bound, validated constructor overrides, and deterministic malformed/oversized classifications; verify LF, CRLF policy, empty frame, EOF, partial frame, maximum-boundary, over-limit, `U+2028`, and `U+2029` tests.
- [x] 5.2 Add a single serialized protocol writer that handles partial writes, flush errors, delivery deadlines, and complete-frame atomicity; verify concurrent response/event/HITL writes never interleave and write failures settle affected pending operations exactly once.
- [x] 5.3 Enforce stdout as protocol-only and use an injected log writer with stderr default in the executable adapter; verify subprocess tests reject accidental stdout logs and preserve stderr diagnostics without interpreting them as protocol frames.
- [x] 5.4 Add clean EOF, malformed input, unsupported version, blocked output, parent cancellation, and process-exit cleanup behavior; verify the subprocess harness observes deterministic exit classification and no pending request or goroutine leak.

## 6. Observability, Replay, and Contract Gates

- [x] 6.1 Emit bounded host correlation, admission, pending-close, delivery, and source-control facts through standard events and `observability/event.RuntimeRecorder` only; verify no direct `runtime/diagnostics` writes, unbounded payloads, cursor bodies, or high-cardinality OTel attributes are introduced.
- [x] 6.2 Add versioned host transcript fixtures and diagnostics replay normalization for valid flow, rejection, duplicates, HITL races, disconnect recovery, terminal conflict, framing drift, output failure, ingress backpressure, and Run/Stream parity; verify legacy Agent Runtime Protocol and Realtime fixtures remain unchanged and passing.
- [x] 6.3 Add a subprocess conformance harness for negotiation, LF framing, Unicode separators, frame bounds, stdout purity, serialized output, backpressure, EOF, and pending cleanup; verify it runs deterministically without a live model or network listener.
- [x] 6.4 Add dedicated host-contract shell and PowerShell gates with parity checks and assertions for source ownership, `RuntimeRecorder` single-write, no hosted listener/store, no global queue, no parallel terminal/cursor state, and no steering/follow-up semantics; verify both scripts produce equivalent pass/fail classifications.
- [x] 6.5 Register the new gate in `scripts/check-quality-gate.*` and update `docs/mainline-contract-test-index.md`; verify contributioncheck covers required commands, fixtures, failure codes, and shell/PowerShell parity.

## 7. Example and Documentation Delivery

- [x] 7.1 Extend the existing realtime-interrupt-resume minimal example only after tasks 1.1-1.2 are complete; verify smoke output includes deterministic markers for host negotiation, command correlation, active interrupt/resume, and authoritative terminal outcome.
- [x] 7.2 Extend the production-ish example with HITL reverse request, duplicate/late response, disconnect recovery, ingress/output backpressure, frame rejection, and stdout purity; verify positive, negative, and boundary marker assertions pass without external services.
- [x] 7.3 Update README, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `docs/development-roadmap.md`, and the Pi comparison cross-reference where governed facts changed; verify docs state the source owners, no-new-config decision, rollback path, deferred steering/follow-up scope, and long-term remote gateway exclusions consistently.
- [x] 7.4 Add or update example contract and migration/governance tests so a code-only example change, missing semantic anchor, missing rollback note, or stale Roadmap/OpenSpec status fails; verify the intended exceptional branches are covered.

## 8. Verification and Delivery

- [x] 8.1 Run focused positive, negative, boundary, concurrency, subprocess, integration, replay, source-owner, and Run/Stream parity suites for `core/types`, `core/runner`, `host`, `host/jsonl`, `observability/event`, `runtime/diagnostics`, `orchestration/composer`, and `tool/diagnosticsreplay`; record commands and results in final review evidence.
- [x] 8.2 Run `go test ./...` and `go test -race ./...`; verify both complete with zero failures.
- [x] 8.3 Run `golangci-lint run --config .golangci.yml`; verify no lint violations or proposal-number identifiers appear in code paths, filenames, or symbols.
- [x] 8.4 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`; verify host contract, Example Impact Assessment, Roadmap status, example doc-first, contract index, and shell/PowerShell parity checks all pass. Evidence: the full PowerShell gate passed in 902.27s with `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=1800`, CGO enabled through the isolated MinGW wrapper, and the existing diagnostics-query benchmark sampled with `BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=500ms` and `BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT=9` to reduce host scheduling noise without changing baselines or thresholds. All 62 steps passed, including `go test ./...`, CGO-enabled `go test -race ./...`, lint, host contract, A64 semantic stability, and A64 performance regression; docs consistency also passed.
- [x] 8.5 Review additive compatibility, bounded state, source ownership, security authorization, first-terminal-wins, disconnect semantics, rollback, and non-goals; verify OpenSpec validation passes and document residual risks and archive readiness without marking tasks complete on placeholder evidence.
