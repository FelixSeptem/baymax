# agent-runtime-protocol-contract Specification

## Purpose
TBD - created by archiving change introduce-agent-runtime-protocol-contract. Update Purpose after archive.
## Requirements
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

### Requirement: Protocol descriptor SHALL separate runtime discovery from capability negotiation

The Agent Runtime Protocol MUST expose an additive `ProtocolDescriptor` projection containing a stable protocol name, effective profile version, runtime identity, declared capabilities, and host-visible actions. Capability declarations MUST distinguish required and optional requests and MUST reuse the adapter capability negotiation vocabulary: `fail_fast` as the default strategy and explicit `best_effort` override when downgrade is allowed.

Capability absence MUST produce deterministic classifications: required absence is `protocol.capability.missing_required`; optional absence under an explicit best-effort request is `protocol.capability.optional_downgraded`. The descriptor MUST NOT imply authorization or create a registry.

#### Scenario: Descriptor advertises a supported required capability

- **WHEN** a host requests a capability declared by the effective protocol profile as supported
- **THEN** descriptor negotiation succeeds with the effective profile version and no downgrade classification

#### Scenario: Missing required capability fails fast

- **WHEN** a host requests a required capability that is not declared or is incompatible with the effective profile
- **THEN** negotiation rejects the request with `protocol.capability.missing_required` and does not mutate Runtime state

#### Scenario: Missing optional capability downgrades only under explicit best effort

- **WHEN** a host requests an optional capability that is unavailable and explicitly selects `best_effort`
- **THEN** negotiation returns a deterministic downgrade with `protocol.capability.optional_downgraded`

#### Scenario: Run and Stream negotiation remains equivalent

- **WHEN** equivalent requests use Run and Stream with the same descriptor, capability set, and strategy
- **THEN** both paths produce semantically equivalent accept, downgrade, or reject classifications and reason taxonomy

### Requirement: Session context projection SHALL be bounded and reference-oriented

The protocol MUST expose an additive Session context projection associated with `SessionRef`. The projection MUST support bounded participant/agent references, a declared context scope, and bounded scalar metadata. Participant references MUST preserve source and role when known. The projection MUST NOT embed message history, prompts, provider SDK objects, file contents, or unbounded arbitrary payloads.

Metadata keys, values, participant count, and serialized size MUST be validated against deterministic profile limits. Invalid or over-limit context MUST fail protocol validation before source Runtime state mutation.

#### Scenario: Session context carries participant and scope references

- **WHEN** a source Runtime provides a Session with participant references, context scope, and valid bounded metadata
- **THEN** the protocol projection preserves those references and metadata without copying conversation or provider content

#### Scenario: Unsupported context metadata is rejected

- **WHEN** a caller supplies an unallowlisted, oversized, or structurally invalid context metadata entry
- **THEN** validation returns a deterministic protocol classification and leaves the source Session unchanged

#### Scenario: Empty context remains backward compatible

- **WHEN** a source Runtime only provides the existing `SessionRef`
- **THEN** the protocol emits an empty optional context projection and existing Session/Run mappings remain valid

### Requirement: Host-visible actions SHALL be explicit and distinct from authorization

The protocol MUST expose a finite canonical action set for lifecycle operations including `cancel`, `resume`, and `retry`, plus versioned capability-specific actions when declared. An action listed by a descriptor indicates availability under the current profile, not permission to execute it. Unsupported actions MUST fail protocol validation before source state mutation; authorization MUST continue through existing policy, readiness, sandbox, and HITL owners.

#### Scenario: Descriptor lists available lifecycle actions

- **WHEN** a Runtime profile supports cancel and retry but not resume
- **THEN** the descriptor lists exactly those available actions and does not infer resume support from the Run state machine

#### Scenario: Unsupported action is rejected

- **WHEN** a host requests an action absent from the effective descriptor
- **THEN** the protocol returns a deterministic unsupported-action classification without mutating the Run or Session

#### Scenario: Available action does not bypass authorization

- **WHEN** a descriptor lists `cancel` but existing policy or readiness denies the caller's operation
- **THEN** the owning policy path denies the operation and the descriptor does not grant an override

### Requirement: Concurrent Run admission SHALL be an explicit source-owned outcome projection

The protocol MUST represent same-Session concurrent-Run policy using the finite vocabulary `reject`, `serialize`, `branch`, `optimistic`, or `unknown`. It MUST represent admission outcomes using `admitted`, `queued`, `branched`, `rejected`, or `unresolved`, with Session/Run correlation, a stable reason code, and conflicting Run references when available.

The projection MUST validate policy/outcome compatibility: `reject` MUST NOT produce `queued` or `branched`; `serialize` MAY produce `admitted` or `queued`; `branch` MAY produce `admitted` or `branched`; `optimistic` MAY produce `admitted` or `rejected`; `unknown` MUST produce `unresolved` unless the source explicitly supplies a compatible decision. Protocol mapping MUST NOT synthesize queueing, cancellation, locking, or branching and MUST NOT mutate source scheduler or Session state.

#### Scenario: Reject policy reports a conflict deterministically

- **WHEN** a source Runtime declares `reject` and receives a conflicting Run for the same Session
- **THEN** the projection returns `rejected` with the source reason and conflict Run references when available

#### Scenario: Serialize policy reports queued admission

- **WHEN** a source Runtime declares `serialize` and a conflicting Run is active
- **THEN** the projection returns `queued` without claiming that the protocol itself owns the queue

#### Scenario: Branch policy preserves causal relationship

- **WHEN** a source Runtime declares `branch` and creates a branch for a conflicting Session Run
- **THEN** the projection returns `branched` with the new Run correlation and existing causal relationship

#### Scenario: Unknown policy is not silently defaulted

- **WHEN** a source Runtime does not expose its same-Session concurrency policy
- **THEN** the projection returns `unknown`/`unresolved` semantics or an explicit source decision and does not infer reject, serialize, branch, or cancel behavior

#### Scenario: Incompatible policy outcome is rejected

- **WHEN** a mapping attempts to emit `queued` for `reject` or `branched` for `serialize`
- **THEN** protocol validation fails with deterministic admission classification and source state remains unchanged

### Requirement: Capability, context, and admission projections SHALL preserve compatibility and observability contracts

All new protocol fields MUST be additive, nullable or defaultable, and associated with an explicit profile version. Existing `agent_runtime_protocol.v1` fixtures and six-object mappings MUST remain valid. New descriptor, context, action, and admission events MUST preserve `run_id`, `session_id`, source, causation, and available trace correlation; diagnostics writes MUST continue through `observability/event.RuntimeRecorder`.

Equivalent Run and Stream inputs MUST normalize to semantically equivalent descriptor negotiation, context validation, action availability, and admission outcomes after permitted event-order normalization.

#### Scenario: Old v1 mapping remains valid without new fields

- **WHEN** an existing `agent_runtime_protocol.v1` fixture contains only the original lifecycle references
- **THEN** replay succeeds and optional descriptor/context/admission fields are absent or defaulted

#### Scenario: Profile drift is detected by replay

- **WHEN** a descriptor, capability reason, context limit, or admission outcome changes for the same declared profile version
- **THEN** protocol replay classifies deterministic drift and the contract gate fails

#### Scenario: Diagnostics correlation uses the single write path

- **WHEN** a descriptor or admission projection emits an observable decision
- **THEN** its additive fields are recorded through `RuntimeRecorder` with protocol correlation and no parallel diagnostics writer

#### Scenario: Run and Stream preserve admission parity

- **WHEN** equivalent Run and Stream requests receive the same source descriptor and concurrent-Run outcome
- **THEN** normalized protocol output preserves equivalent decision, reason, capability, and context semantics
