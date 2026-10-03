## Purpose

This capability gives an embedded Baymax runtime a bounded, native way to pause a Run when a tool produces a resumable business action and to continue the same Run after an external decision without taking ownership of the action's business payload.

## ADDED Requirements

### Requirement: Runtime SHALL accept bounded dynamic action registration after tool execution

The runtime SHALL accept an additive dynamic action registration from a tool result or tool dispatcher at the tool-result boundary. A registration MUST contain a non-empty opaque token or reference, a bounded action kind, a resumability declaration, and the producing Run, iteration, and tool-call correlation. The runtime MUST NOT infer a pending action from arbitrary tool content or copy business action payloads into Runner-owned state.

#### Scenario: Tool returns a resumable action reference
- **WHEN** a tool completes successfully with a valid opaque action reference for the current Run and iteration
- **THEN** runtime records the bounded reference and evaluates the Run for a dynamic action pause before starting the next model step

#### Scenario: Tool result has no dynamic action
- **WHEN** a tool completes without a dynamic action registration
- **THEN** runtime preserves the existing tool-result-to-next-model-step behavior and does not synthesize `input_required`

#### Scenario: Malformed dynamic action registration is rejected
- **WHEN** a registration omits its token, correlation, action kind, or resumability declaration, or exceeds declared bounds
- **THEN** runtime rejects the registration fail-closed, does not invoke a subsequent executor or model step, and emits a deterministic validation classification

### Requirement: A valid resumable action SHALL pause the same Run as input_required

When a valid resumable action is registered, runtime MUST transition the source Run from `working` to `input_required` at the tool-result boundary, persist a resumable checkpoint reference, and prevent subsequent model generation and executor dispatch until a matching decision is admitted. The terminal/runtime projection MUST preserve the original `run_id`, `session_id`, iteration, producing tool correlation, and an additive action reference.

#### Scenario: Dynamic action pauses before the next model step
- **WHEN** a tool registers a valid resumable action after execution
- **THEN** the Run becomes `input_required`, `gate_checks` records the dynamic action evaluation, and no next model call occurs

#### Scenario: Dynamic action blocks executor continuation
- **WHEN** a dynamic action is registered while later executor work remains in the current workflow
- **THEN** runtime does not invoke that executor until a valid resume decision has been accepted

#### Scenario: Non-resumable action does not create a resumable Run
- **WHEN** a tool registers an action explicitly marked non-resumable
- **THEN** runtime records the bounded action reference according to its declared outcome but MUST NOT expose it as a same-Run resumable `input_required` checkpoint

### Requirement: Native resume SHALL restore the same Runner execution correlation

Runtime SHALL expose a versioned checkpoint/resume contract for a Run paused by a dynamic action. A valid resume MUST restore the same `run_id`, `session_id`, iteration boundary, tool-result references, pending action reference, and Run/Stream mode, then continue through the existing loop. Resume MUST NOT be implemented as retry, follow-up promotion, or an implicitly new causally related Run.

#### Scenario: Valid confirmation resumes the original Run
- **WHEN** the source action owner admits one valid confirmation for an active `input_required` checkpoint
- **THEN** runtime resumes the checkpoint with the original `run_id`, applies the confirmation reference, and continues from the saved iteration boundary

#### Scenario: Resume preserves prior tool results
- **WHEN** a paused Run resumes after one or more tool results have been checkpointed
- **THEN** the restored model request contains the same normalized tool-result references without re-executing completed tools

#### Scenario: Resume mode remains equivalent
- **WHEN** equivalent Run and Stream checkpoints receive equivalent decisions
- **THEN** both paths continue with semantically equivalent iteration, action, and terminal correlation

### Requirement: Dynamic action resume admission SHALL be idempotent and fail closed

Resume admission MUST require matching Run, session, checkpoint version/digest, opaque action token, and a bounded idempotency key. Duplicate equivalent decisions MUST be idempotent. Missing, stale, expired, mismatched, conflicting, or terminal checkpoints MUST be rejected without mutating Runner state or invoking a tool, executor, or provider.

#### Scenario: Duplicate resume is idempotent
- **WHEN** the same resume identity and normalized decision are submitted more than once
- **THEN** runtime returns a duplicate/idempotent admission and does not execute the continuation twice

#### Scenario: Resume token does not match checkpoint
- **WHEN** a resume request carries a token or checkpoint digest belonging to another action or Run
- **THEN** runtime rejects the request with a deterministic correlation classification and preserves `input_required` state

#### Scenario: Terminal checkpoint cannot be resumed
- **WHEN** a resume request targets a checkpoint whose source Run is already completed, failed, or canceled
- **THEN** runtime rejects the request without synthesizing a new terminal transition

### Requirement: Dynamic action checkpoints SHALL preserve source ownership and bounded observability

Checkpoint storage MUST be injectable and bounded. Baymax SHALL own only Runner recovery metadata and opaque action references; the Application/adapter SHALL remain the owner of action payload, authorization, confirmation policy, executor, and business state. Dynamic pause, resume, rejection, duplicate, stale, and terminal outcomes MUST be emitted through existing event and `RuntimeRecorder` paths with additive nullable/defaultable fields.

#### Scenario: Business payload remains outside Runner checkpoint
- **WHEN** an Application adapter registers a PendingAction with sensitive or domain-specific fields
- **THEN** the Runner checkpoint stores only the opaque reference and bounded metadata needed for correlation and recovery

#### Scenario: Pause and resume diagnostics are recorded canonically
- **WHEN** a dynamic action pauses or resumes a Run
- **THEN** diagnostics expose bounded action/checkpoint/resume facts through the canonical writer without creating a parallel business-state store

#### Scenario: Replay reproduces dynamic pause/resume facts
- **WHEN** offline replay consumes a versioned dynamic action pause/resume fixture
- **THEN** replay reproduces normalized admission, checkpoint, Run/Stream parity, and terminal facts without invoking providers, tools, or executors
