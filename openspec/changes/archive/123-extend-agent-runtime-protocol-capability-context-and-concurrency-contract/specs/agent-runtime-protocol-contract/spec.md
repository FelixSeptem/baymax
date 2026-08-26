## ADDED Requirements

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
