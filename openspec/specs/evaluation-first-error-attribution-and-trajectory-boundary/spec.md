# evaluation-first-error-attribution-and-trajectory-boundary Specification

## Purpose
Provides a deterministic, bounded, reference-only evaluation contract for locating the first trajectory deviation, explaining its source-owned cause, and recording the decision boundary that applied at that point without persisting reasoning or transcript bodies.

## Requirements

### Requirement: First-error attribution SHALL be versioned, bounded, and deterministic

The evaluation layer MUST accept a versioned first-error attribution associated with a stable corpus item or Badcase identity. The attribution MUST identify the first-error step and ordinal, first-error kind, root-cause owner, primary cause, bounded secondary causes, recoverability, integer confidence, and a stable trajectory-prefix digest. First-error kind MUST be one of `decision|tool_selection|tool_input|policy|memory_retrieval|memory_application|context|provider|termination|evidence`; root-cause owner MUST be one of `model|tool|policy|memory|context|provider|runtime|host|unknown`; recoverability MUST be one of `recoverable|non_recoverable|unknown`; confidence MUST use integer basis points in the inclusive range `0..10000`. An attribution MUST contain at most 16 ranked secondary causes and its canonical serialized representation MUST NOT exceed 64 KiB. Normalization MUST be deterministic, MUST reject unsupported versions, taxonomy values or bounds, and MUST produce a stable attribution identity for semantically equivalent inputs.

#### Scenario: Valid first-error attribution is normalized
- **WHEN** an attribution contains a supported version, stable evaluation correlation, valid first-error identity, bounded causes, recoverability, confidence, and prefix digest
- **THEN** normalization succeeds and emits the same canonical identity for semantically equivalent input ordering

#### Scenario: First-error identity is incomplete
- **WHEN** an attribution omits the corpus item or Badcase association, first-error step, ordinal, kind, root-cause owner, primary cause, or prefix digest
- **THEN** validation fails fast with `first_error_schema_drift` and emits no partial attribution

#### Scenario: Confidence or bounds are invalid
- **WHEN** confidence is outside its declared integer range or a bounded cause, evidence, action, or serialized-size limit is exceeded
- **THEN** validation fails deterministically with `first_error_schema_drift`

### Requirement: Trajectory-prefix decision boundary SHALL distinguish allowed and forbidden actions

The attribution MUST expose bounded reference-only sets for acceptable actions, forbidden actions, required evidence, and safety constraints at the first-error boundary. Each set MUST contain at most 32 entries. Each action and constraint MUST use a stable kind and identifier or digest; normalization MUST reject an action that is simultaneously acceptable and forbidden and MUST NOT infer an action boundary from free-form summaries.

#### Scenario: Decision boundary is canonicalized
- **WHEN** acceptable actions, forbidden actions, required evidence, and safety constraints are semantically equivalent but supplied in different set order
- **THEN** normalization emits identical ordered boundary output and the same attribution identity

#### Scenario: Action boundary conflicts
- **WHEN** the same stable action is declared both acceptable and forbidden at the first-error boundary
- **THEN** validation fails with `trajectory_action_boundary_conflict`

#### Scenario: Required evidence is missing
- **WHEN** an action boundary requires an evidence reference that is absent from the attribution evidence set
- **THEN** validation fails with `trajectory_required_evidence_missing`

### Requirement: Attribution evidence SHALL remain source-owned and reference-only

Evidence MUST be represented by at most 32 bounded references to existing event, checkpoint, artifact, policy, tool, memory, context, provider, host, or runtime owners. The contract MUST preserve owner, stable identifier, and available digest/version correlation, MUST reject duplicate conflicting references, and MUST NOT contain raw reasoning, transcript text, provider responses, tool output bodies, memory bodies, workspace bodies, credentials, or unbounded maps.

#### Scenario: Existing continuity evidence is reused
- **WHEN** first-error attribution cites an event, checkpoint, artifact, or continuity reference already owned by an existing contract
- **THEN** evaluation records only the bounded owner/identity/digest/version reference and does not copy or resolve the source body

#### Scenario: Body-bearing evidence is rejected
- **WHEN** attribution evidence contains reasoning, transcript, provider, tool, memory, workspace, credential, or other body content
- **THEN** validation fails with `first_error_privacy_violation` and leaves source state unchanged

#### Scenario: Conflicting duplicate evidence is rejected
- **WHEN** the same evidence owner and stable identifier appears with conflicting digest or version data
- **THEN** validation fails deterministically with `first_error_evidence_conflict`

### Requirement: Attribution comparison SHALL classify semantic drift canonically

Offline comparison MUST distinguish drift in first-error step, first-error kind, root-cause owner, primary or secondary cause, trajectory action boundary, required evidence, evidence references, recoverability, confidence, and prefix digest. Equivalent local/distributed and Run/Stream evaluation inputs MUST normalize identically, and comparison MUST NOT use last-write-wins behavior for conflicting records.

#### Scenario: First-error location changes
- **WHEN** candidate attribution identifies a different first-error step, ordinal, kind, or trajectory-prefix digest from the baseline
- **THEN** comparison emits the corresponding canonical step, kind, or prefix drift classification

#### Scenario: Cause or owner changes
- **WHEN** candidate root-cause owner, primary cause, or ranked secondary cause differs from baseline
- **THEN** comparison emits deterministic owner or cause drift without mutating either attribution

#### Scenario: Run and Stream attribution are equivalent
- **WHEN** equivalent Run and Stream evidence produces the same source-owned first-error facts and decision boundary
- **THEN** normalized attribution identity and comparison outcome are semantically equivalent

### Requirement: First-error attribution SHALL be offline and side-effect free

Attribution normalization and comparison MUST NOT call providers, tools, memory backends, artifact or transcript resolvers, Git, workspace mutation, runtime restore, or live execution paths. Memory retrieval/application failures MUST be represented as corpus scenarios and bounded references only, and attribution output MUST NOT become a runtime, memory, task, plan, or terminal source of truth.

#### Scenario: Memory application failure is evaluated offline
- **WHEN** a corpus fixture identifies a memory retrieval miss, scope error, or memory application error using existing diagnostic references
- **THEN** attribution is evaluated without reading or mutating memory content, indexes, lifecycle state, or runtime context

#### Scenario: Failed attribution leaves owners unchanged
- **WHEN** validation or comparison rejects malformed, conflicting, or drifting attribution input
- **THEN** no runtime, provider, tool, memory, context, checkpoint, artifact, workspace, experiment, or feedback owner is mutated
