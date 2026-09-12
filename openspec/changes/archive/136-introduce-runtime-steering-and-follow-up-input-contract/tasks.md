## Example Impact Assessment

修改示例

Example documentation baseline must be completed before any example code or implementation task is checked: update `examples/agent-modes/MATRIX.md` and `examples/agent-modes/realtime-interrupt-resume/README.md` with semantic anchor, runtime path, expected markers, rollback notes, steering/follow-up boundaries, and replay/gate mapping.

## 1. Documentation-First Baseline and RED Matrix

- [x] 1.1 Update `examples/agent-modes/MATRIX.md` with steering/follow-up semantic anchor, source owner, safe-point path, expected admission/apply/not-applied markers, rollback notes, contracts, gates, and replay references; verify the doc-first gate recognizes the baseline before implementation tasks are checked.
- [x] 1.2 Update both `minimal` and `production-ish` `realtime-interrupt-resume` READMEs with input kind, safe-point, queue-bound, disconnect, terminal-race, and Run/Stream parity expectations; verify referenced runtime paths and rollback notes resolve to existing or planned paths.
- [x] 1.3 Add failing contract tests and transcript cases for envelope validation, steering/follow-up separation, bounded admission, duplicate/stale/terminal outcomes, safe-point application, follow-up promotion, cancel/Realtime races, disconnect, replay idempotency, and Run/Stream parity; verify the focused suites fail for missing behavior before implementation.

## 2. Core Input Contract and Validation

- [x] 2.1 Add versioned steering/follow-up envelope, input identity, source correlation, admission status/reason, apply-boundary, and bounded outcome DTOs in `core/types`; verify JSON round-trip, omission compatibility, and stable normalized identity tests.
- [x] 2.2 Implement validation for input kind, message bounds, Session/Run correlation, causation, profile version, and malformed payloads; verify positive, negative, oversized, missing-correlation, and control-character cases fail before source mutation.
- [x] 2.3 Define deterministic reason codes and outcome vocabulary for accepted, rejected, duplicate, stale, terminal, backpressure, disconnected, and not-applied cases; verify reason normalization and backward-compatible default handling.

## 3. Source-Owned Runner Input Semantics

- [x] 3.1 Add bounded source-owned steering and follow-up lanes associated with active Run/causal execution, with duplicate identity tracking and non-blocking backpressure; verify capacity, ordering, duplicate, shutdown, and race tests under `go test -race ./core/runner`.
- [x] 3.2 Add safe-point application after atomic model, tool, and HITL boundaries and before the next model decision; verify steering is never injected into an in-flight Provider call, tool execution, or pending resolver.
- [x] 3.3 Integrate input admission with existing readiness, policy, sandbox, capability, and host authorization owners; verify denied, unknown Run, Session mismatch, stale, and terminal inputs fail-fast without mutating source state.
- [x] 3.4 Define follow-up promotion at the source terminal/idle boundary through the existing Run/Stream admission path; verify a promoted follow-up receives a distinct causal Run and the original terminal Run remains immutable.
- [x] 3.5 Arbitrate steering/follow-up against cancel, retry, Realtime interrupt/resume, terminal publication, and source shutdown; verify first-terminal-wins, cancel precedence, Realtime cursor ownership, and no parallel lifecycle state machine.
- [x] 3.6 Exercise equivalent Run and Stream workloads through model, tool, HITL, interrupt, terminal, and follow-up boundaries; verify normalized admission, apply/not-applied, causation, and terminal outcomes remain equivalent.

## 4. Embedded Host Command Integration

- [x] 4.1 Extend the transport-neutral host command vocabulary and capability projection for steering/follow-up while preserving command-response versus runtime-event separation; verify accepted admission never claims input application or follow-up execution completion.
- [x] 4.2 Route host input commands to source-owned admission and expose correlated apply/not-applied/promotion facts; verify the host adapter owns no input queue, Session history, terminal state, or Realtime cursor.
- [x] 4.3 Extend `host/jsonl` framing, serialized delivery, disconnect cleanup, and bounded backpressure for input commands; verify malformed, oversized, duplicate, stale, terminal, and blocked-output cases settle deterministically with stdout purity preserved.
- [x] 4.4 Verify host authorization and capability availability remain distinct for both input kinds; ensure unsupported input capability fails before source mutation and existing 134 command behavior remains unchanged.

## 5. Observability, Replay, and Contract Gates

- [x] 5.1 Emit bounded input admission, queue pressure, apply-boundary, duplicate/stale, disconnect, promotion, and terminal-race facts through `observability/event.RuntimeRecorder` only; verify raw input content and high-cardinality payloads never reach diagnostics or OTel attributes.
- [x] 5.2 Add versioned steering/follow-up transcript fixtures and diagnostics replay normalization for valid flow, rejection, duplicates, ordering drift, safe-point drift, follow-up promotion, disconnect, terminal conflict, and Run/Stream parity; verify replay is side-effect-free and idempotent.
- [x] 5.3 Add dedicated shell and PowerShell steering/follow-up contract gates covering source ownership, bounded lanes, safe-point-only application, no direct Provider injection, no global queue, no second Session/terminal/cursor state, and parity classifications; verify both scripts produce equivalent pass/fail results.
- [x] 5.4 Register the gates in `scripts/check-quality-gate.*`, update `docs/mainline-contract-test-index.md`, and extend contribution checks for required fixtures, commands, failure codes, and shell/PowerShell parity; verify missing evidence blocks the gate.

## 6. Examples and Governed Documentation

- [x] 6.1 Extend the existing minimal example with accepted steering, safe-point application, duplicate/stale rejection, and deterministic markers; verify the example runs without a live provider and preserves existing realtime markers.
- [x] 6.2 Extend the production-ish example with follow-up promotion, queue pressure, cancel/Realtime races, disconnect/not-applied outcomes, replay identity, and Run/Stream parity markers; verify positive, negative, and boundary smoke checks pass without external services.
- [x] 6.3 Update `README.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `docs/development-roadmap.md`, and the Pi comparison cross-reference with source owners, no-new-config decision, rollback path, non-persistence boundary, and deferred remote gateway exclusions; verify docs consistency and roadmap status gates pass.

## 7. Integrated Verification and Handoff

- [x] 7.1 Run focused contract, replay, host, runner, and example suites with CGO-enabled race detection; verify all new positive, negative, boundary, and parity cases pass.
  - Evidence: `go test ./core/types ./core/runner ./host ./host/jsonl ./observability/event ./tool/diagnosticsreplay -count=1` passed; both realtime example variants and the agent-mode smoke contract passed. `go test -race` was attempted with `CGO_ENABLED=1`, `CC=D:\\tools\\mingw64\\bin\\gcc.exe`, and the MinGW bin/libexec paths present, but the installed toolchain failed inside Go cgo with `gcc: fatal error: cannot execute 'cc1': CreateProcess: No such file or directory`; this is recorded as an environment-blocked race result, not a passing race result.
- [x] 7.2 Run `openspec validate --all`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`; record exact results and any intentionally skipped command with its reason.
  - Evidence: `openspec validate --all` passed (115 passed, 0 failed); `go test ./... -count=1 -timeout 20m` passed with `CGO_ENABLED=0`; `golangci-lint run --config .golangci.yml` passed (0 issues); `pwsh -File scripts/check-docs-consistency.ps1` passed; runtime steering/follow-up, embedded-host, replay, and all reached quality-gate suites passed. The full quality gate completed 58 steps successfully and then stopped while reading `go env CGO_ENABLED` because the environment denied Go telemetry access and the script received an ErrorRecord instead of text. `go test -race ./...` was attempted with the same CGO/MinGW setup and was blocked by the cgo `cc1.exe` startup failure described in 7.1.
- [x] 7.3 Perform final scope review against the design non-goals and architecture boundaries; verify no hosted service, global queue, second history/session owner, provider-specific API, raw-payload diagnostic field, or implicit terminal transition was introduced.
  - Evidence: diff and contract-gate review found source-owned per-Run steering/follow-up lanes only, no global or diagnostics-backed input queue, no host-owned Session/history/terminal/cursor state, no new hosted listener or remote gateway, no provider-specific steering API, and no raw input/cursor/frame body in bounded recorder projections. `runtime/*` has no new `mcp/http` or `mcp/stdio` dependency and non-`mcp/*` code has no new `mcp/internal/*` dependency. Existing `context/assembler` provider SDK imports predate this change and were not modified; they are outside this proposal's diff and remain a separate repository baseline concern. Terminal transitions continue through existing source-owned arbitration, and follow-up promotion creates a distinct causal Run without mutating the original terminal Run.
