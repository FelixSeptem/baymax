## ADDED Requirements

### Requirement: Runtime SHALL emit a versioned bounded handoff
The runtime SHALL emit a `handoff.v1` record at a legal stable cut containing objective, completed work, pending work, failed attempts, file changes, tool results, policy/sandbox/admission state, references, next actions, facts, inferences, and needs-confirmation items. The record SHALL be bounded and SHALL use nullable/defaultable additive fields.

#### Scenario: Stable cut produces handoff
- **WHEN** compression is requested after a finalized event, tool call, checkpoint, or flushed stream boundary
- **THEN** the runtime emits a schema-valid bounded handoff with the source run and checkpoint identifiers

#### Scenario: Invalid cut is rejected
- **WHEN** compression is requested during an unfinalized tool call, uncommitted artifact, unresolved policy decision, or unflushed stream delta
- **THEN** the runtime rejects the cut and records `handoff_cut_invalid`

### Requirement: Handoff SHALL preserve protected evidence and source references
The runtime SHALL preserve critical/immutable evidence, terminal events, file write outcomes, failed attempts, and reference metadata, and SHALL resolve references through existing Artifact, Checkpoint, Session History, and snapshot owners.

#### Scenario: Protected evidence survives compression
- **WHEN** pressure-driven compression removes non-critical context
- **THEN** all protected evidence and canonical references remain present and verifiable

#### Scenario: Missing source reference blocks recoverability
- **WHEN** a referenced artifact, checkpoint, or history leaf cannot be resolved
- **THEN** the handoff is marked non-recoverable and records `handoff_reference_loss`

#### Scenario: Reference owner dispatch and durable restore identity
- **WHEN** a handoff contains artifact, checkpoint, session-history, or snapshot references and a durable restore-operation store is supplied
- **THEN** restore delegates each reference to its existing owner, validates all references before mutation, and repeated restore from another assembler instance returns the same stable operation result without re-reading the owners

### Requirement: Handoff SHALL separate facts, inferences, and confirmation needs
Facts SHALL be backed by observed events or source metadata; inferences SHALL include provenance and confidence; uncertain claims SHALL be placed in needs-confirmation. The runtime MUST NOT promote unsupported claims into facts.

#### Scenario: Unsupported completion claim
- **WHEN** the compressor cannot find a source event proving an item completed
- **THEN** the item is not emitted as a completed fact and is represented as pending or needs-confirmation

### Requirement: Compression quality SHALL have deterministic fallback
The runtime SHALL evaluate schema validity, required-fact coverage, protected evidence retention, reference resolution, and next-action recoverability. Below-threshold or failed compression SHALL use the existing deterministic fallback chain and SHALL leave the primary Run executable.

#### Scenario: Quality below threshold
- **WHEN** a generated handoff fails the configured quality threshold
- **THEN** the runtime records `handoff_quality_below_threshold`, applies deterministic fallback, and does not silently expose an unrecoverable handoff

#### Scenario: Fail-fast mode
- **WHEN** handoff generation fails and fail-fast mode is configured
- **THEN** the compression operation returns an error without mutating source state

### Requirement: Restore from handoff SHALL be idempotent and Run/Stream equivalent
Restore SHALL validate schema and source boundaries before mutation, inject through existing reference-first/context assembler paths, and preserve equivalent terminal, policy, and diagnostic semantics for Run and Stream.

#### Scenario: Repeated restore
- **WHEN** the same valid handoff is restored more than once
- **THEN** the resulting state and diagnostics are equivalent and no duplicate logical events are created

#### Scenario: Run and Stream parity
- **WHEN** equivalent Run and Stream executions restore from equivalent handoffs
- **THEN** next-action eligibility, terminal outcome, and reference projections are semantically equal

#### Scenario: Explicit finalized stream boundary
- **WHEN** a model emits `checkpoint_id` or `stream_flushed` metadata on a finalized completion event
- **THEN** the next assembler request may select the corresponding checkpoint or flushed-stream cut; metadata on an incremental delta is ignored

### Requirement: Handoff diagnostics SHALL be single-writer and replayable
Handoff generation, fallback, validation, and restore outcomes SHALL be recorded only through `RuntimeRecorder`; replay SHALL classify fact loss, reference loss, invalid cuts, schema/quality drift, non-idempotent restore, and Run/Stream mismatch.

#### Scenario: Replay detects drift
- **WHEN** a handoff fixture is replayed against current behavior and a protected fact is missing
- **THEN** replay fails with canonical `handoff_fact_loss` classification
