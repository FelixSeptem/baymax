# Dynamic Action Gate and Native Run Resume Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow a tool-produced opaque PendingAction reference to pause a Baymax Run as native `input_required` and resume the same Runner execution exactly once from a bounded checkpoint.

**Architecture:** Keep Application/adapter ownership of PendingAction payload, authorization, confirmation, and executor state. Add an additive `DynamicActionReference` carrier to tool outcomes, a source-owned bounded checkpoint/resume interface, and a Runner boundary between tool dispatch and the next model step. Host commands carry only opaque token/checkpoint references; runtime events and terminal projections remain source-owned. Realtime resume, retry, and follow-up promotion remain separate paths.

**Tech Stack:** Go, `core/types`, `core/runner`, embedded host envelopes, `RuntimeRecorder`, deterministic diagnostics replay, integration contract tests, PowerShell and shell gates.

---

## File and responsibility map

- Create `core/types/dynamic_action_resume.go`: public additive action-reference, checkpoint, decision, admission, and validation types.
- Modify `core/types/types.go`: add nullable `PendingAction` to `ToolResult` and preserve JSON compatibility.
- Modify `core/types/protocol.go`: add same-Run pause/resume state validation and protocol reference helpers without changing retry/follow-up semantics.
- Create `core/runner/dynamic_action_resume.go`: action registration normalization, checkpoint orchestration, resume admission, idempotency, and source-owned error classification.
- Modify `core/runner/runner.go`: invoke dynamic action evaluation after tool dispatch and before the next model step; emit `input_required` and prevent continuation.
- Modify `core/runner/control.go`: retain a resumable active control/checkpoint until action resolution and expose same-Run resume execution; keep terminal cleanup for settled Runs.
- Modify `core/runner/runtime_input.go`: do not route dynamic resume through follow-up promotion; add explicit action-resume admission bridge if the host adapter needs the Runner entry point.
- Modify `core/types/host_contract.go` and `host/host.go`: add versioned action-resume command/request correlation and source-outcome projection.
- Modify `runtime/diagnostics/store.go` and the runner finish metadata: add bounded nullable/defaultable action/checkpoint/resume fields.
- Create `tool/diagnosticsreplay/dynamic_action_resume.go` and fixtures under `tool/diagnosticsreplay/testdata/`: deterministic offline pause/resume normalization.
- Create `integration/dynamic_action_resume_contract_test.go`: Application-style adapter, executor suppression, same-Run resume, rejection, and Run/Stream parity.
- Create a minimal integration example under `examples/` (not `examples/agent-modes` unless the documentation baseline is extended first) and update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/mainline-contract-test-index.md`, `docs/external-adapter-template-index.md`, and `docs/development-roadmap.md`.

### Task 1: Add the additive dynamic action and checkpoint data contract

**Files:**
- Create: `core/types/dynamic_action_resume.go`
- Modify: `core/types/types.go:766-783`
- Modify: `core/types/protocol.go:786-820`
- Test: `core/types/dynamic_action_resume_test.go`
- Test: `core/types/types_test.go`

- [ ] **Step 1: Write failing validation and JSON tests**

  Add tests for:

  ```go
  func TestDynamicActionReferenceValidateRequiresBoundedCorrelation(t *testing.T)
  func TestDynamicActionReferenceJSONRoundTripPreservesOptionalField(t *testing.T)
  func TestDynamicActionCheckpointRejectsDigestAndRunMismatch(t *testing.T)
  func TestResumeAdmissionDuplicateIsIdempotent(t *testing.T)
  func TestLegacyToolResultWithoutPendingActionStillDecodes(t *testing.T)
  ```

  The valid fixture must contain token, kind, resumable flag, Run ID, session ID, iteration, call ID, and a bounded source/digest. Invalid cases must cover empty token, oversized token, missing correlation, negative iteration, and unsupported decision.

- [ ] **Step 2: Run the focused tests and verify they fail for missing types/validation**

  Run:

  ```powershell
  go test ./core/types -run 'DynamicAction|ResumeAdmission|LegacyToolResult' -count=1
  ```

  Expected: compile or assertion failures because the new carrier and validation do not exist.

- [ ] **Step 3: Implement the additive types and `ToolResult.PendingAction`**

  Define `DynamicActionReference`, `DynamicActionDecision`, `RunCheckpoint`, `RunResumeAdmission`, and `RunCheckpointStore` with explicit bounds and `Validate` methods. Add:

  ```go
  PendingAction *DynamicActionReference `json:"pending_action,omitempty"`
  ```

  to `ToolResult`. Keep all new fields optional and ensure old JSON without `pending_action` remains valid. Add protocol helpers that allow `working -> input_required -> working` only through the dynamic-resume path and reject resume from completed/failed/canceled states.

- [ ] **Step 4: Run the focused tests and compatibility checks**

  Run:

  ```powershell
  gofmt -w core/types/dynamic_action_resume.go core/types/dynamic_action_resume_test.go core/types/types.go core/types/types_test.go core/types/protocol.go
  go test ./core/types -run 'DynamicAction|ResumeAdmission|LegacyToolResult' -count=1
  go test ./core/types -count=1
  ```

  Expected: all focused and existing core/types tests pass.

- [ ] **Step 5: Commit the contract slice**

  ```powershell
  git add core/types/dynamic_action_resume.go core/types/dynamic_action_resume_test.go core/types/types.go core/types/types_test.go core/types/protocol.go
  git commit -m "feat: add dynamic action resume contract types"
  ```

### Task 2: Implement Runner dynamic action pause and checkpoint creation

**Files:**
- Create: `core/runner/dynamic_action_resume.go`
- Modify: `core/runner/runner.go:StateDecideNext` and `dispatchReactToolCalls` boundary
- Modify: `core/runner/control.go` active-run state and cleanup
- Test: `core/runner/dynamic_action_resume_test.go`
- Test: `core/runner/runner_test.go`

- [ ] **Step 1: Write failing Runner pause tests**

  Add a fake dispatcher/model scenario where `local.application.prepare` returns `ToolResult.PendingAction` and a second model call would otherwise produce a final answer. Assert:

  ```go
  func TestDynamicActionPausesSameRunBeforeNextModelStep(t *testing.T)
  func TestDynamicActionSuppressesPendingExecutor(t *testing.T)
  func TestDynamicActionRecordsGateCheckAndInputRequired(t *testing.T)
  func TestToolWithoutDynamicActionRetainsExistingLoop(t *testing.T)
  ```

  The pause test must assert zero second model calls, zero executor calls after registration, `RunID` preservation, `TerminalOutcome.State == types.RunStateInputRequired`, and `gate_checks == 1` (or the documented dynamic-check count).

- [ ] **Step 2: Run the Runner tests and verify the current implementation reports completed**

  Run:

  ```powershell
  go test ./core/runner -run 'DynamicAction' -count=1 -v
  ```

  Expected: the new tests fail because current tool dispatch proceeds to the next model step and finalizes `completed`.

- [ ] **Step 3: Add normalization and checkpoint creation after tool dispatch**

  In `core/runner/dynamic_action_resume.go`, implement:

  ```go
  func normalizeDynamicAction(outcome types.ToolCallOutcome, runID, sessionID string, iteration int) (types.DynamicActionReference, error)
  func (e *Engine) pauseForDynamicAction(ctx context.Context, req types.RunRequest, ref types.DynamicActionReference, iteration int, state runnerResumeState) (*types.ClassifiedError, error)
  ```

  Validate correlation, call ID, bounds, and resumability. Save a checkpoint before mutating the active control to `input_required`. Use an injected in-memory store by default for single-process operation; return a deterministic fail-closed error when a required store is unavailable.

- [ ] **Step 4: Insert the pause boundary before `StateModelStep`**

  After `dispatchReactToolCalls` returns outcomes and before assigning `state = StateModelStep`, inspect outcomes for a dynamic registration. If a resumable registration exists, call `pauseForDynamicAction`, emit a pending action/runtime event and terminal projection, set the control state to `input_required`, and leave the loop suspended. Do not call the model or executor after that point.

- [ ] **Step 5: Run focused tests and existing action-gate tests**

  Run:

  ```powershell
  gofmt -w core/runner/dynamic_action_resume.go core/runner/dynamic_action_resume_test.go core/runner/runner.go core/runner/control.go
  go test ./core/runner -run 'DynamicAction|ActionGate|ClarificationRunLifecycleResume' -count=1
  ```

  Expected: new pause tests pass and existing pre-dispatch Action Gate/clarification tests remain green.

- [ ] **Step 6: Commit the pause slice**

  ```powershell
  git add core/runner/dynamic_action_resume.go core/runner/dynamic_action_resume_test.go core/runner/runner.go core/runner/control.go
  git commit -m "feat: pause runs on dynamic action registration"
  ```

### Task 3: Implement same-Run resume and negative admission paths

**Files:**
- Modify: `core/runner/dynamic_action_resume.go`
- Modify: `core/runner/control.go`
- Modify: `core/runner/runtime_input.go`
- Test: `core/runner/dynamic_action_resume_test.go`
- Test: `core/runner/control_test.go`

- [ ] **Step 1: Write failing resume tests**

  Add tests for:

  ```go
  func TestDynamicActionResumeContinuesOriginalRunIDAndIteration(t *testing.T)
  func TestDynamicActionResumeDoesNotRepeatCompletedTool(t *testing.T)
  func TestDynamicActionResumeDuplicateIsIdempotent(t *testing.T)
  func TestDynamicActionResumeRejectsStaleOrMismatchedCheckpoint(t *testing.T)
  func TestDynamicActionResumeRejectsTerminalRun(t *testing.T)
  func TestDynamicActionResumeRunStreamParity(t *testing.T)
  ```

  Assert that confirmation resumes the same `run_id`, uses the checkpointed tool result, increments only the resume attempt, and executes the continuation once. Rejection cases must keep the source Run `input_required` and record no provider/tool invocation.

- [ ] **Step 2: Run the tests and verify resume APIs are absent or incorrect**

  ```powershell
  go test ./core/runner -run 'DynamicActionResume' -count=1 -v
  ```

  Expected: failures until the same-Run resume admission exists.

- [ ] **Step 3: Implement `ResumeDynamicAction` with compare-and-set semantics**

  Add a public Engine method with a concrete signature:

  ```go
  func (e *Engine) ResumeDynamicAction(ctx context.Context, decision types.DynamicActionDecision, h types.EventHandler, stream bool) (types.RunResult, error)
  ```

  Validate checkpoint digest, token, session/Run, decision, and idempotency key. Atomically claim the resume identity, restore the request/iteration/tool-result references, apply the adapter decision reference, and invoke the existing Run/Stream continuation path without allocating a new Run ID. Duplicate equivalent decisions return the original admission/result projection; conflicting reuse fails closed.

- [ ] **Step 4: Keep lifecycle ownership distinct**

  Ensure `RetryRun`, `PromoteRuntimeFollowUp`, and Realtime cursor resume do not call `ResumeDynamicAction`. Keep an action-paused control alive until resume, denial, timeout, or explicit cancellation settles it; then run the existing cleanup path. Add comments and errors distinguishing `dynamic_action_resume` from `realtime_resume` and `follow_up_promotion`.

- [ ] **Step 5: Run focused and race tests**

  ```powershell
  gofmt -w core/runner/dynamic_action_resume.go core/runner/dynamic_action_resume_test.go core/runner/control.go core/runner/runtime_input.go
  go test ./core/runner -run 'DynamicActionResume|Realtime|RetryRun|RuntimeInput' -count=1
  go test -race ./core/runner -run 'DynamicActionResume' -count=1
  ```

- [ ] **Step 6: Commit the resume slice**

  ```powershell
  git add core/runner/dynamic_action_resume.go core/runner/dynamic_action_resume_test.go core/runner/control.go core/runner/runtime_input.go core/runner/control_test.go
  git commit -m "feat: resume dynamic actions within the same run"
  ```

### Task 4: Extend host command correlation and adapter bridge

**Files:**
- Modify: `core/types/host_contract.go`
- Modify: `host/host.go`
- Modify: `host/host_test.go`
- Create: `integration/dynamic_action_resume_contract_test.go`

- [ ] **Step 1: Write failing host admission tests**

  Add tests for accepted, rejected, duplicate, stale, mismatched, and disconnected action-resume commands. Assert command responses contain admission only and runtime events carry the later source-owned outcome.

- [ ] **Step 2: Run the host tests and verify no action-resume envelope exists**

  ```powershell
  go test ./host -run 'ActionResume|Pending|Disconnect' -count=1 -v
  ```

  Expected: compile or assertion failures until the envelope and bridge are added.

- [ ] **Step 3: Add versioned action-resume envelopes**

  Extend the host command vocabulary with an action-resume command containing `run_id`, `session_id`, opaque token, checkpoint version/digest, decision reference, and idempotency key. Reuse existing validation, authorization, bounded correlation, duplicate handling, and source ownership patterns. The host adapter must call `Engine.ResumeDynamicAction` and never mutate terminal state itself.

- [ ] **Step 4: Add the Application-style adapter fixture**

  In `integration/dynamic_action_resume_contract_test.go`, create a fake Application action owner that stores the business PendingAction outside Baymax, maps it to an opaque token, confirms it once, and counts executor calls. Verify:

  - unconfirmed action returns `input_required` and executor count stays zero;
  - confirmation resumes the original Run ID;
  - business payload never appears in checkpoint/diagnostic projection;
  - Run and Stream produce equivalent normalized outcomes.

- [ ] **Step 5: Run host and integration tests**

  ```powershell
  gofmt -w core/types/host_contract.go host/host.go host/host_test.go integration/dynamic_action_resume_contract_test.go
  go test ./host ./integration -run 'ActionResume|DynamicAction' -count=1
  ```

- [ ] **Step 6: Commit the host slice**

  ```powershell
  git add core/types/host_contract.go host/host.go host/host_test.go integration/dynamic_action_resume_contract_test.go
  git commit -m "feat: expose dynamic action resume through host correlation"
  ```

### Task 5: Add diagnostics, replay, and contract gates

**Files:**
- Modify: `runtime/diagnostics/store.go`
- Modify: `core/runner/runner.go` finish metadata
- Create: `tool/diagnosticsreplay/dynamic_action_resume.go`
- Create: `tool/diagnosticsreplay/dynamic_action_resume_test.go`
- Create: `tool/diagnosticsreplay/testdata/dynamic_action_resume.v1.json`
- Modify: `tool/contributioncheck` contract/gate tests and `docs/mainline-contract-test-index.md`

- [ ] **Step 1: Write failing replay and diagnostics tests**

  Add a fixture containing registration, pause, confirmation, denial, timeout, duplicate, stale, terminal race, and Run/Stream records. Assert replay output includes only bounded token/checkpoint digests and stable normalized classifications.

- [ ] **Step 2: Implement additive diagnostics fields**

  Add nullable/defaultable fields for dynamic action count, action reference digest, checkpoint digest/version, pause reason, resume attempt, and resume admission classification. Route all writes through the existing `RuntimeRecorder` path; do not add a second recorder or business payload field.

- [ ] **Step 3: Implement side-effect-free replay**

  Normalize fixture input, validate correlation/idempotency/checkpoint transitions, compare expected Run/Stream outcomes, and reject any fixture that would invoke a provider, tool, executor, or checkpoint mutation. Return stable replay classifications for drift and conflict.

- [ ] **Step 4: Add shell/PowerShell contract coverage**

  Extend the contribution/gate tests to check additive schema compatibility, opaque payload redaction, same-Run correlation, executor suppression, and shell/PowerShell parity. Register the new test mapping in `docs/mainline-contract-test-index.md`.

- [ ] **Step 5: Run replay and gate tests**

  ```powershell
  gofmt -w runtime/diagnostics/store.go core/runner/runner.go tool/diagnosticsreplay/dynamic_action_resume.go tool/diagnosticsreplay/dynamic_action_resume_test.go tool/contributioncheck/*dynamic*test.go
  go test ./tool/diagnosticsreplay ./tool/contributioncheck -run 'DynamicAction|ActionResume' -count=1
  ```

- [ ] **Step 6: Commit the evidence slice**

  ```powershell
  git add runtime/diagnostics/store.go core/runner/runner.go tool/diagnosticsreplay tool/contributioncheck docs/mainline-contract-test-index.md
  git commit -m "test: add dynamic action resume replay and gates"
  ```

### Task 6: Add documentation and the minimum integration example

**Files:**
- Create: `examples/dynamic-action-resume/main.go`
- Create: `examples/dynamic-action-resume/README.md`
- Modify: `docs/runtime-config-diagnostics.md`
- Modify: `docs/runtime-module-boundaries.md`
- Modify: `docs/external-adapter-template-index.md`
- Modify: `docs/development-roadmap.md`
- Modify: `README.md`

- [ ] **Step 1: Add the example contract before implementation claims**

  Document semantic anchor `dynamic_action.input_required_same_run`, runtime path `core/runner -> core/types -> host`, expected markers for registration/pause/resume/executor suppression, and rollback notes. The example must keep the fake Application PendingAction body outside Baymax and print only opaque reference/terminal facts.

- [ ] **Step 2: Implement the minimal example**

  Use a deterministic fake model/tool sequence: prepare tool returns a dynamic action reference, the first execution exposes `input_required`, a confirmation calls same-Run resume, and the final executor result completes. Add a Stream variant with equivalent normalized output.

- [ ] **Step 3: Update boundary and diagnostic documentation**

  Explain that dynamic action resume is distinct from Realtime resume, retry, and follow-up promotion; document additive fields, fail-closed errors, source ownership, and in-memory versus durable checkpoint capability.

- [ ] **Step 4: Run docs/example checks**

  ```powershell
  go run ./examples/dynamic-action-resume
  pwsh -File scripts/check-docs-consistency.ps1
  go test ./tool/contributioncheck -run 'DynamicAction|Docs|ReleaseStatus' -count=1
  ```

- [ ] **Step 5: Commit the documentation slice**

  ```powershell
  git add examples/dynamic-action-resume docs/runtime-config-diagnostics.md docs/runtime-module-boundaries.md docs/external-adapter-template-index.md docs/development-roadmap.md README.md
  git commit -m "docs: describe native dynamic action resume boundaries"
  ```

### Task 7: Full verification and OpenSpec completion

**Files:**
- Modify: `openspec/changes/introduce-dynamic-action-gate-and-native-run-resume/tasks.md`
- Verify: all implementation, test, replay, documentation, and gate files above

- [ ] **Step 1: Run affected package tests and race checks**

  ```powershell
  go test ./core/types ./core/runner ./host ./integration ./tool/diagnosticsreplay ./tool/contributioncheck -count=1
  go test -race ./core/types ./core/runner ./host ./integration ./tool/diagnosticsreplay -count=1
  ```

  Expected: all dynamic action tests, legacy Action Gate/Realtime/retry/follow-up tests, and Run/Stream parity tests pass.

- [ ] **Step 2: Run repository gates**

  ```powershell
  go test ./...
  go test -race ./...
  golangci-lint run --config .golangci.yml
  pwsh -File scripts/check-quality-gate.ps1
  pwsh -File scripts/check-docs-consistency.ps1
  openspec validate --all
  git diff --check
  ```

  Record unrelated pre-existing agent-mode shell/PowerShell or shared-wrapper failures separately; do not weaken this change's dynamic action tests to accommodate them.

- [ ] **Step 3: Mark the OpenSpec tasks with evidence**

  Check every item in `openspec/changes/introduce-dynamic-action-gate-and-native-run-resume/tasks.md` only after its stated command or observable behavior passes. Include the Example Impact Assessment evidence and the contract/replay/gate mappings.

- [ ] **Step 4: Review the final diff and prepare archival handoff**

  ```powershell
  git status --short
  git diff --stat master...HEAD
  openspec status --change "introduce-dynamic-action-gate-and-native-run-resume" --json
  ```

  Confirm no business PendingAction payload, provider SDK, hosted control plane, or unrelated refactor entered the change before requesting review and later archival.
