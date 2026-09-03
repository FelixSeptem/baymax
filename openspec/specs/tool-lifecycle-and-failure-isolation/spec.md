# tool-lifecycle-and-failure-isolation Specification

## Purpose
TBD - created by archiving change harden-tool-lifecycle-and-failure-isolation-contract. Update Purpose after archive.
## Requirements
### Requirement: Tool calls SHALL expose a deterministic logical lifecycle projection

For every tool call processed by the runtime, the normalized contract MUST represent the logical stages `prepare`, `validate`, `authorize`, `execute`, and `finalize`, while allowing an explicitly recorded `skipped` or `not_applicable` stage when the source path does not require that stage. The projection MUST retain the original `call_id`, tool name, source identity, run/step correlation when known, and input position. Stage projection MUST NOT create a second execution state machine or transfer ownership from the existing dispatcher, policy, sandbox, middleware, or terminal arbiter.

#### Scenario: Successful local call completes all stages
- **WHEN** a valid local tool call passes policy and executes successfully
- **THEN** replay and diagnostics expose the stages in deterministic order with the original `call_id` and input position preserved

#### Scenario: Missing or invalid input is rejected before execution
- **WHEN** a tool is unknown or its arguments fail schema validation
- **THEN** the result records `prepare`/`validate` failure, does not claim `execute`, and still emits an idempotent `finalize` projection

#### Scenario: Source path has no applicable policy substage
- **WHEN** an embedded source does not require a distinct policy or sandbox substage
- **THEN** the normalized result records an explicit `not_applicable` or `skipped` authorization stage rather than inventing an authorization decision

### Requirement: Lifecycle failures SHALL distinguish rejection from execution failure

The runtime MUST classify lifecycle failures using existing error families and reason taxonomies while exposing a bounded failure origin sufficient to distinguish lookup/schema rejection, policy or allowlist denial, sandbox denial/failure, middleware short-circuit/failure, panic, timeout, cancellation, retry exhaustion, and ordinary tool/provider execution failure. A pre-execution rejection MUST NOT be reported as a successful execution, and an execution failure MUST retain whether execution had started.

#### Scenario: Policy denial is not an execution error
- **WHEN** policy precedence, adapter allowlist, sandbox capability, or egress governance denies a tool call before side effects
- **THEN** the normalized result records an authorization rejection, preserves the canonical deny reason, and does not claim that tool execution started

#### Scenario: Panic is isolated and classified
- **WHEN** tool or middleware code panics during invocation
- **THEN** the dispatcher recovers the panic, records a runtime failure origin with the existing classified error semantics, and continues finalization without leaking the panic to unrelated calls

#### Scenario: Timeout and cancellation preserve start state
- **WHEN** a call times out or its context is canceled before or during execution
- **THEN** the result distinguishes pre-start rejection from started execution interruption and records the canonical timeout/cancel reason without retrying outside existing policy

#### Scenario: Retry exhaustion remains one call outcome
- **WHEN** configured tool retries are exhausted after one or more attempts
- **THEN** the normalized result records attempt metadata and one final failure for the original `call_id`, without emitting duplicate business outcomes

### Requirement: Finalization SHALL be idempotent and preserve source-owned facts

Every call that enters dispatch processing MUST produce at most one authoritative normalized finalization projection for its attempt scope. Repeated finalization observations MUST be idempotent; late conflicting observations MUST be diagnostics-only and MUST NOT overwrite the first accepted business result. Finalization MUST preserve valid partial output, completed tool-call facts, resource-release status, and existing terminal-outcome ownership without claiming rollback or exactly-once side effects.

#### Scenario: Repeated finalize is deduplicated
- **WHEN** middleware, dispatcher, recovery, or diagnostics replay observes the same completed call more than once
- **THEN** the normalized output contains one business finalization and marks repeats as idempotent duplicates

#### Scenario: Late conflict does not overwrite result
- **WHEN** a later observer reports a conflicting result for a call whose finalization was already accepted
- **THEN** the runtime records a conflict classification and preserves the first business result and terminal projection

#### Scenario: Partial facts survive abnormal completion
- **WHEN** a call emits valid content or completes a tool sub-operation before panic, timeout, cancellation, or provider failure
- **THEN** finalization retains those source-owned facts with their original correlation and marks the interruption separately

### Requirement: Parallel tool feedback SHALL preserve deterministic correlation and order

For a batch of tool calls, completion timing MAY be concurrent, but normalized outcomes and model feedback MUST align with the original input order and `call_id`. The lifecycle projection MUST preserve per-call stage and attempt metadata without introducing a global queue or serializing otherwise independent calls.

#### Scenario: Fast call completes before slow call
- **WHEN** parallel calls complete in an order different from their input order
- **THEN** the returned outcomes and ReAct feedback remain in original input order with matching `call_id` values

#### Scenario: Duplicate call IDs are rejected or classified deterministically
- **WHEN** a batch contains duplicate or blank call identifiers
- **THEN** the runtime applies the existing validation/error policy and emits a stable classification without silently merging unrelated calls

### Requirement: Run and Stream SHALL expose equivalent lifecycle semantics

For equivalent request input, effective configuration, dependency state, policy decision, sandbox capability, middleware behavior, and tool outcome, Run and Stream MUST produce semantically equivalent lifecycle stage order, failure origin, attempt classification, finalization idempotency, retained facts, terminal classification, and input-order tool feedback. Differences in transport event timing are permitted only when they do not change normalized semantics.

#### Scenario: Equivalent successful tool loop
- **WHEN** Run and Stream execute the same valid multi-call ReAct tool loop
- **THEN** both paths expose equivalent stage projections, ordered tool results, and terminal completion semantics

#### Scenario: Equivalent denial and timeout
- **WHEN** Run and Stream encounter the same policy denial or middleware/tool timeout
- **THEN** both paths expose the same rejection/interruption origin, canonical reason, finalization state, and terminal mapping

#### Scenario: Provider failure after partial tool facts
- **WHEN** equivalent Run and Stream executions fail after one or more tool calls have completed
- **THEN** both paths retain the same completed facts and classify the remaining failure equivalently

### Requirement: Lifecycle diagnostics and replay SHALL be additive, bounded, and library-first

Lifecycle diagnostics MUST be written through `observability/event.RuntimeRecorder` and exposed with additive, nullable, defaultable fields. Replay MUST provide a versioned canonical fixture plus malformed, unsupported-version, ordering-drift, failure-origin-drift, duplicate-finalize, and hosted-ownership negative cases. The contract gate MUST keep shell and PowerShell classifications equivalent and MUST fail if lifecycle work introduces a transport listener, hosted execution/session store, global invocation queue, external event store, or second terminal state machine. High-cardinality arguments, payloads, and causation values MUST NOT become OTel metric dimensions.

#### Scenario: Canonical lifecycle fixture replays deterministically
- **WHEN** replay processes a valid lifecycle fixture covering success, rejection, panic, timeout, cancellation, retry exhaustion, parallel ordering, and partial facts
- **THEN** normalized output is stable across repeated runs and matches the expected stage and failure projections

#### Scenario: Replay drift blocks the gate
- **WHEN** normalized stage order, failure origin, finalization idempotency, retained facts, or Run/Stream parity differs from the fixture
- **THEN** replay returns a stable drift classification and the lifecycle gate fails fast

#### Scenario: Hosted ownership is rejected
- **WHEN** implementation adds a transport server, hosted tool/session store, global lifecycle queue, or alternate terminal state machine
- **THEN** the gate fails with a deterministic library-first boundary classification

#### Scenario: Historical consumers remain compatible
- **WHEN** an older diagnostics or replay consumer reads a lifecycle record without the new fields
- **THEN** parsing succeeds with nullable/default behavior and existing error/result fields retain their previous meaning
