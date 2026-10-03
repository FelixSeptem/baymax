## 1. Dynamic action contract

- [x] 1.1 Add additive `PendingActionReference`/dynamic registration types with bounded token, action kind, resumability, Run/iteration/tool correlation, validation, and JSON round-trip tests; verify malformed, oversized, and legacy-absent values are rejected or defaulted deterministically. Evidence: `core/types/dynamic_action_resume.go`, `core/types/dynamic_action_resume_test.go`, focused and full `core/types` tests.
- [x] 1.2 Extend tool result/dispatcher plumbing to carry an optional dynamic action registration without changing existing tool behavior; verify existing local tool and adapter conformance tests remain green when the field is absent. Evidence: additive `ToolResult.PendingAction`, full `core/runner` regression suite and focused local dispatch coverage.
- [ ] 1.3 Define source-owned checkpoint and resume interfaces with version, digest, bounded metadata, idempotency identity, and explicit durability capability; verify duplicate and conflicting checkpoint operations are deterministic.

## 2. Runner pause and same-Run resume

- [x] 2.1 Normalize dynamic registrations after tool dispatch and evaluate their Action Gate decision before the next model step; verify a valid resumable registration pauses execution and suppresses subsequent model/executor calls. Evidence: `core/runner/dynamic_action_resume.go` and `TestDynamicActionPausesBeforeNextModelStepAndResumesSameRun`.
- [x] 2.2 Project a dynamic pause as `input_required` while preserving Run/session/iteration/tool correlation and additive gate/checkpoint diagnostics; verify the terminal/runtime projection does not report `completed` for a paused Run. Evidence: `TerminalOutcome` projection and the same Runner test.
- [ ] 2.3 Implement same-Run checkpoint restore for Run and Stream, including tool-result references and iteration boundary; verify confirmation resumes the original `run_id` exactly once without re-executing completed tools.
- [ ] 2.4 Implement fail-closed resume validation for missing, stale, expired, mismatched, conflicting, duplicate, and terminal checkpoints; verify no provider, tool, executor, or terminal mutation occurs on rejection.

## 3. Host and adapter integration

- [ ] 3.1 Add versioned host action-resume command/response envelopes and source-owned runtime event projections; verify command admission is separated from resumed, denied, timed-out, duplicate, stale, and terminal outcomes.
- [ ] 3.2 Add an Application/adapter contract fixture that maps a PendingAction business object to an opaque token and back without copying business payload into Runner checkpoint state; verify authorization and executor ownership remain in the adapter.
- [ ] 3.3 Preserve distinct semantics for Realtime interrupt/resume, retry, and follow-up promotion; verify existing lifecycle tests reject accidental conversion into dynamic action resume.

## 4. Replay, diagnostics, and examples

- [ ] 4.1 Add versioned dynamic pause/resume replay fixtures covering confirmation, denial, timeout, duplicate, stale, mismatch, terminal race, and Run/Stream parity; verify replay is side-effect-free and does not invoke live providers/tools/executors.
- [ ] 4.2 Add contract/gate coverage for additive nullable/default diagnostics, checkpoint digest correlation, `gate_checks`, resume attempts, and canonical `RuntimeRecorder` writes; verify shell/PowerShell gate parity.
- [ ] 4.3 Add a minimal integration example showing adapter-owned PendingAction plus Baymax native `input_required`/same-Run resume, with semantic anchor, runtime path, expected markers, and rollback notes; verify the example demonstrates opaque references only.
- [ ] 4.4 Update runtime, host, adapter, and mainline contract documentation with ownership boundaries and failure taxonomy; verify docs consistency and roadmap status checks pass.

## 5. Verification and rollout

- [ ] 5.1 Run affected package tests, integration contract/replay tests, and race tests for dynamic action pause/resume; verify Run/Stream parity and legacy behavior.
- [ ] 5.2 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; record any unrelated pre-existing agent-mode gate failures explicitly.
