## Purpose

This capability gives externally authored extensions a deterministic, offline conformance contract that validates existing admission and lifecycle boundaries and returns actionable, replayable feedback without executing untrusted source or creating a second runtime.

## ADDED Requirements

### Requirement: Authoring conformance cases SHALL be versioned, declarative, and bounded

The conformance harness MUST accept a versioned case profile containing a bounded case identity, extension descriptor inputs, admission inputs, controlled lifecycle actions, and expected observable outcomes. Required fields, enum values, size limits, and ordering rules MUST be validated before evaluation; unknown additive fields MUST be ignored safely.

#### Scenario: Valid case is canonicalized deterministically
- **WHEN** a case contains valid descriptor, admission, lifecycle, and expected-result fields in any map or candidate enumeration order
- **THEN** the harness canonicalizes it into one stable ordering and evaluates the same result on repeated runs and on both supported shells

#### Scenario: Malformed case fails before lifecycle evaluation
- **WHEN** a required field is missing, an enum is unknown, a bound is exceeded, or the profile version is unsupported
- **THEN** the harness returns a deterministic schema-phase failure with a stable reason code and performs no admission or activation action

### Requirement: Conformance evaluation SHALL reuse extension governance owners

The harness MUST evaluate resource resolution, descriptor validation, capability negotiation, readiness/policy admission, bounded lifecycle execution, and generation reload through the existing owner contracts. A rejected or blocked candidate MUST NOT produce an activation side effect, and the harness MUST NOT create a parallel extension state machine.

#### Scenario: Required capability rejection is reported at admission
- **WHEN** a case requests a required capability unavailable in the supplied runtime capability set
- **THEN** the result is `failed` at the admission phase with the canonical missing-capability reason and activation remains false

#### Scenario: Optional capability degradation follows policy
- **WHEN** an optional capability is unavailable and the case selects best-effort admission
- **THEN** the result records a deterministic downgrade marker and canonical reason while preserving the existing extension admission policy

#### Scenario: Equal-precedence digest conflict is rejected
- **WHEN** two candidates for one extension identity have equal source precedence but different content digests
- **THEN** resolution fails with the canonical ambiguous-conflict reason and neither candidate is activated

### Requirement: Conformance results SHALL provide stable author feedback

Each evaluated case MUST return a bounded result containing `passed|failed` status, contract phase, stable reason code, optional field, expected and actual classifications, and a remediation hint selected from a versioned reason-to-guidance mapping. Results MUST NOT depend on platform-specific error strings or include source code, credentials, reasoning, workspace content, or unbounded payloads.

#### Scenario: Invalid manifest identifies the repair target
- **WHEN** a descriptor omits a required identity, compatibility, digest, or source field
- **THEN** the result identifies the manifest phase, canonical missing-field reason, offending field, and bounded guidance for adding that field

#### Scenario: Lifecycle fault identifies policy-safe remediation
- **WHEN** a controlled action times out, panics, returns an invalid result, or fails finalization
- **THEN** the result identifies the lifecycle phase and canonical failure reason, reports the configured skip/deny/degrade outcome, and confirms that unrelated runtime state was not mutated

### Requirement: Conformance SHALL verify reload isolation and Run/Stream parity

The harness MUST verify that successful reloads advance activation generation, stale-generation events are suppressed, failed reloads retain the previous generation, and equivalent Run and Stream inputs produce equivalent extension resolution, admission, activation, failure, and reload decisions. A conformance failure MUST NOT replace the authoritative runtime terminal outcome.

#### Scenario: Failed reload rolls back atomically
- **WHEN** a new generation fails descriptor validation or controlled activation during reload
- **THEN** the previous valid generation remains active, no partial new state is committed, and the result records a deterministic reload/rollback classification

#### Scenario: Stale generation event is suppressed
- **WHEN** an event from an older activation generation arrives after a successful reload
- **THEN** the event is ignored for the active extension and the result records stale-generation suppression without changing the current generation

#### Scenario: Run and Stream cases remain equivalent
- **WHEN** the same canonical case is evaluated through Run and Stream projections
- **THEN** both projections return equivalent selected resources, admission outcomes, lifecycle classifications, and terminal-preservation evidence

### Requirement: Replay and gates SHALL be deterministic and side-effect free

Conformance fixtures MUST be replayable offline and idempotently by the diagnostics replay tool. Dedicated shell and PowerShell gates MUST execute the same semantic profile, classify drift consistently, and be wired into the quality gate. Replay MUST not write runtime diagnostics, access a network, or mutate workspace/source state.

#### Scenario: Replaying a passing fixture is idempotent
- **WHEN** the same versioned fixture is replayed multiple times
- **THEN** canonical output is byte-for-byte stable and no additional side effect or duplicate diagnostic write is produced

#### Scenario: Drift is classified instead of silently rebased
- **WHEN** actual output differs in ordering, reason code, phase, remediation, idempotency, or Run/Stream parity
- **THEN** replay returns a stable drift classification and the gate fails without rewriting the expected fixture

#### Scenario: Shell and PowerShell gates agree
- **WHEN** the conformance profile is run through the POSIX and Windows gate wrappers
- **THEN** both wrappers invoke equivalent tests and produce the same pass/fail classification and profile identity

## Example Impact Assessment

`无需示例变更（附理由）`。The capability is exercised by offline fixtures and integration harnesses; no `examples/agent-modes` runtime behavior or mode inventory changes are required.
