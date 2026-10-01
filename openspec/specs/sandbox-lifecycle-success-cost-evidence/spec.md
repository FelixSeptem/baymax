# sandbox-lifecycle-success-cost-evidence Specification

## Purpose
Defines a versioned, bounded, offline contract for evaluating supplied sandbox lifecycle facts and successful-task cost evidence without creating or controlling sandbox execution.

## Requirements

### Requirement: Sandbox lifecycle evidence SHALL use a versioned, bounded, privacy-safe contract
The system SHALL accept `sandbox_lifecycle_success_cost_evidence.v1` only as an offline, host-supplied evidence contract. Its normalized input and canonical output SHALL contain only opaque invocation and session references, catalog-like backend/profile identity, session mode, lifecycle phase, bounded duration and resource buckets, retry ordinal, stable reason code, terminal outcome, success classification, and optional Run/Stream evidence snapshots.

The contract SHALL reject unsupported versions, missing required facts, invalid enum values, oversized collections or buckets, and prohibited fields with deterministic reason codes. The canonical output SHALL include a stable digest derived only from normalized permitted facts.

The contract SHALL NOT accept or emit command text, arguments, environment, workdir, mounts, paths, endpoints, credentials, stdout, stderr, raw violation payloads, prompts, reasoning, or unbounded bodies.

#### Scenario: Valid bounded evidence is normalized
- **WHEN** the evaluator receives a valid `sandbox_lifecycle_success_cost_evidence.v1` payload containing permitted bounded facts
- **THEN** it returns a canonical projection and stable digest without contacting a sandbox, host, network service, or provider

#### Scenario: Prohibited command-derived field is supplied
- **WHEN** a lifecycle evidence payload contains command text, output, a path, credential material, or another prohibited field
- **THEN** the evaluator fails closed with a deterministic privacy or schema reason and produces no partial cost result

### Requirement: Lifecycle evidence SHALL preserve canonical lifecycle continuity
The system SHALL evaluate lifecycle facts in canonical order: `acquire`, `launch`, `execute`, optional `retry`, `release` or `recover`, and terminal outcome. It SHALL distinguish cold launch, per-session reuse, and recovery or resume using supplied facts without becoming a production session owner.

Invalid continuity, including execution before acquisition, recovery without a prior recoverable failure, duplicate release, reuse after a terminal outcome, contradictory terminal outcomes, or an invalid retry ordinal, SHALL produce deterministic classifications and SHALL NOT be silently repaired.

#### Scenario: Per-session reuse follows a successful acquisition
- **WHEN** evidence records a per-session invocation that reuses an already acquired session and has no new launch phase
- **THEN** the canonical projection classifies the invocation as reuse rather than cold launch

#### Scenario: Recovery is claimed without prior failure
- **WHEN** evidence contains a recovery phase without a preceding recoverable failure or interrupted execution fact
- **THEN** the evaluator returns a deterministic lifecycle-continuity classification and no successful-task cost

### Requirement: Successful-task cost SHALL be derived only from complete terminal evidence
The system SHALL derive a unit successful-task cost summary only when one terminal task is explicitly classified successful and its required bounded lifecycle and resource facts are complete and internally consistent. The summary SHALL use only normalized duration and resource buckets, never live wall-clock observation, live probes, raw consumption values, or inferred data.

Failed, unknown, missing, or contradictory terminal outcomes SHALL result in `insufficient-evidence` or a stable lifecycle-gap reason as applicable, and SHALL NOT fabricate a successful-task cost.

#### Scenario: Complete terminal success contributes bounded cost
- **WHEN** evidence has a continuous lifecycle, a successful terminal outcome, and all required bounded cost facts
- **THEN** the evaluator emits a deterministic successful-task cost summary for that task

#### Scenario: Failed terminal outcome lacks successful-task cost
- **WHEN** evidence terminates in failure after bounded lifecycle facts have been recorded
- **THEN** the evaluator omits successful-task cost and returns the canonical non-success verdict or reason

### Requirement: Evidence evaluation SHALL use stable verdicts and baseline comparison
The system SHALL return exactly one evidence verdict: `within-baseline`, `lifecycle-gap-confirmed`, or `insufficient-evidence`.

`within-baseline` SHALL mean complete permitted evidence matches the supplied bounded baseline expectation. `lifecycle-gap-confirmed` SHALL mean complete permitted evidence proves a declared lifecycle or successful-cost representation gap. `insufficient-evidence` SHALL mean the evaluator cannot establish either conclusion because required permitted facts are absent, invalid, or contradictory.

Baseline comparison SHALL use only supplied bounded lifecycle and resource buckets. It SHALL NOT operate a sandbox, inspect platform state, observe clock time, change admission, or alter execution behavior.

#### Scenario: Bounded evidence matches the supplied baseline
- **WHEN** complete normalized lifecycle and successful-task cost buckets equal the supplied baseline expectation
- **THEN** the evaluator returns `within-baseline` with a stable digest and reason ordering

#### Scenario: Required facts are incomplete
- **WHEN** a payload cannot establish terminal success or a required lifecycle bucket
- **THEN** the evaluator returns `insufficient-evidence` rather than treating the missing fact as a confirmed gap

### Requirement: Evidence replay SHALL be immutable, offline, and Run/Stream equivalent
The system SHALL evaluate recorded lifecycle facts without mutating them and SHALL produce semantically equivalent canonical projections, verdicts, reasons, and successful-task cost summaries for equivalent Run and Stream evidence.

Repeated evaluation of identical evidence SHALL be idempotent. Unknown additive fields SHALL be ignored when they do not alter required v1 facts; incompatible version, type, or required-field violations SHALL fail fast with deterministic classification.

#### Scenario: Equivalent Run and Stream evidence is replayed
- **WHEN** equivalent Run and Stream lifecycle evidence is evaluated
- **THEN** the resulting projection, verdict, and successful-task cost summary are semantically equivalent

#### Scenario: A replayed payload includes an unknown additive field
- **WHEN** valid v1 evidence includes an unknown field outside the normalized contract
- **THEN** replay ignores that field and produces the same canonical output on repeated evaluation

### Requirement: Evidence contract SHALL remain reference-only
The lifecycle-success-cost evidence contract SHALL be reference-only. It SHALL NOT introduce a sandbox executor, backend, platform driver, proxy, service account, credential handler, filesystem or network policy implementation, global mutable session store, runtime configuration key, diagnostics persistence field, model selection behavior, or a second readiness, admission, or terminal-state machine.

#### Scenario: Evidence is evaluated in an offline test process
- **WHEN** a test or replay gate evaluates lifecycle evidence
- **THEN** evaluation has no sandbox execution, configuration activation, network access, persistent state mutation, or runtime decision side effect
