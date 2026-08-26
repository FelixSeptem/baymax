## ADDED Requirements

### Requirement: Runtime SHALL expose a library-first canonical lifecycle object model
Runtime MUST expose canonical, embeddable protocol references for Session, Run, Step, Event, Artifact, and Checkpoint. The model MUST use stable correlation identifiers where applicable: `session_id`, `run_id`, `step_id`, `parent_step_id`, `event_id`, `causation_id`, `artifact_id`, and `checkpoint_id`.

The model MUST remain additive and reference-oriented; it MUST NOT require a hosted session service, hosted artifact store, or control-plane dependency.

#### Scenario: Host correlates a tool step with its run and session
- **WHEN** a host receives a mapped tool-execution protocol step for a Run with a known Session
- **THEN** the step includes its `run_id`, `session_id`, `step_id`, and source metadata, and exposes `parent_step_id` only when a parent exists

#### Scenario: Optional source correlation remains compatible
- **WHEN** a source subsystem does not own an artifact or parent-step identifier
- **THEN** its canonical protocol object omits that optional identifier without synthesizing a conflicting value

### Requirement: Runtime SHALL provide a deterministic minimal Run lifecycle state machine
Runtime MUST expose the canonical Run states `submitted`, `working`, `input_required`, `completed`, `failed`, and `canceled`. A Run MUST enter terminal state only through `completed`, `failed`, or `canceled`; cancel MUST be idempotent; resume MUST be accepted only from `input_required`; retry MUST preserve causal association without mutating a terminal Run into `working`.

#### Scenario: Realtime interrupt maps to input-required
- **WHEN** an active Run accepts a valid realtime interrupt under the existing realtime contract
- **THEN** its protocol lifecycle maps to `input_required` and includes the existing resumable-cursor correlation

#### Scenario: Invalid terminal resume is rejected
- **WHEN** a caller requests protocol resume for a completed, failed, or canceled Run
- **THEN** validation rejects the transition with deterministic protocol classification and does not modify source Runtime state

### Requirement: Runtime SHALL map existing execution semantics without replacing their owners
Runtime MUST provide deterministic one-way mappings from Runner Run/Stream, Workflow Step, Teams Task, Scheduler attempt, A2A Task, and realtime interrupt/resume semantics to canonical protocol objects. Mappings MUST retain source module identity and MUST NOT introduce a parallel execution state machine, scheduler, or source-of-truth store.

#### Scenario: Equivalent Run and Stream normalize equivalently
- **WHEN** equivalent work is executed via Runner Run and Stream under unchanged effective configuration
- **THEN** normalized protocol Run terminal state, Step class, error class, and causal relationships are semantically equivalent after permitted event-order normalization

#### Scenario: A2A task retains orchestration correlation
- **WHEN** an A2A Task is dispatched with workflow, team, task, and step correlation
- **THEN** the mapped protocol Run/Step/Event retains the supplied correlation without redefining A2A lifecycle ownership

### Requirement: Runtime SHALL expose canonical protocol event mapping with source-scoped ordering
Runtime MUST map existing standard events, timeline events, realtime envelopes, and A2A lifecycle events to canonical protocol Event envelopes. Every mapped Event MUST include `event_id`, `run_id` when known, source metadata, event kind, timestamp, and payload. Realtime-mapped Events MUST preserve the existing sequence and idempotency semantics; non-realtime sources MUST NOT claim realtime ordering guarantees.

#### Scenario: Repeated realtime event remains idempotent after mapping
- **WHEN** the same valid realtime event is ingested repeatedly and mapped to protocol Event envelopes
- **THEN** the protocol mapping preserves existing deduplication semantics and does not inflate logical interrupt, resume, or event counters

#### Scenario: Source event lacks realtime sequence
- **WHEN** a Workflow or Teams timeline event is mapped without a realtime sequence
- **THEN** the canonical Event records its source and does not assign a synthetic realtime sequence

### Requirement: Runtime SHALL expose artifact and checkpoint lineage as references
Runtime MUST expose `ArtifactRef` and `CheckpointRef` as reference-only protocol objects. An Artifact reference MUST retain `id`, `type`, `locator`, optional digest, and producing Run/Step correlation when known. A Checkpoint reference MUST retain identifier, schema version, source component, optional Run/Session correlation, and integrity reference. Existing snapshot manifest segment ownership and restore semantics MUST remain unchanged.

#### Scenario: Isolate handoff artifact is mapped without copying body content
- **WHEN** an isolate handoff artifact is projected to an Artifact reference
- **THEN** the reference preserves its ID, type, locator, and producing correlation while avoiding duplication of deferred artifact body content

#### Scenario: Snapshot import remains module-owned
- **WHEN** a caller imports a checkpoint reference backed by an existing snapshot manifest
- **THEN** `strict|compatible` policy, compatibility window, digest validation, and idempotent import remain enforced by the snapshot contract

### Requirement: Runtime SHALL preserve fail-fast boundaries while representing recoverable outcomes
Protocol mapping MAY represent a recoverable tool or business failure as a failed Step outcome or error Event only when the owning Runtime path already allows recovery. Configuration validation, security/permission denial, protocol validation, snapshot compatibility conflict, and module-boundary violations MUST retain their existing fail-fast behavior and deterministic classifications.

#### Scenario: Recoverable tool failure is visible to a Run consumer
- **WHEN** a tool failure is classified as recoverable by the owning Runner policy
- **THEN** the mapped Step exposes failed outcome data without suppressing the owner-defined retry or continuation semantics

#### Scenario: Invalid configuration remains fail-fast
- **WHEN** protocol mapping is requested under an invalid effective configuration
- **THEN** runtime rejects the operation through existing configuration validation and MUST NOT convert the violation into a recoverable Step outcome

### Requirement: Runtime Protocol SHALL remain contract-replayable and free of hosted control-plane dependencies
Runtime MUST provide replayable `agent_runtime_protocol.v1` fixtures for valid mapping and deterministic drift cases, including invalid state transition, missing correlation, event ordering/deduplication, artifact/checkpoint lineage, and source mapping divergence. The contract gate MUST preserve shell/PowerShell semantic parity and assert that protocol support does not introduce a hosted control-plane dependency.

#### Scenario: Mapping drift is detected by replay
- **WHEN** a fixture changes canonical state, correlation, ordering, or lineage semantics from the expected normalized output
- **THEN** replay returns a deterministic protocol drift classification and the contract gate fails

#### Scenario: Control-plane dependency is introduced
- **WHEN** protocol implementation introduces a hosted runtime gateway, session service, or artifact store dependency
- **THEN** the protocol contract gate fails its `control_plane_absent` assertion
