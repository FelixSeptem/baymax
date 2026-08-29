## MODIFIED Requirements

### Requirement: Runtime SHALL expose artifact and checkpoint lineage as references

Runtime MUST expose `ArtifactRef` and `CheckpointRef` as reference-only protocol objects. An Artifact reference MUST retain `id`, `type`, `locator`, optional digest, and producing Run/Step correlation when known. A Checkpoint reference MUST retain identifier, schema version, source component, optional Run/Session correlation, and integrity reference, and MAY include additive parent, branch, history, restore-source, replay, and workspace provenance references. Existing snapshot manifest segment ownership and restore semantics MUST remain unchanged.

#### Scenario: Snapshot checkpoint exposes optional lineage
- **WHEN** a valid snapshot manifest is projected with root, derived, branch, or replay context
- **THEN** the checkpoint reference preserves manifest fields and validates optional lineage without changing the manifest

#### Scenario: Missing lineage parent is rejected
- **WHEN** a derived or branch checkpoint lacks a valid parent reference
- **THEN** protocol validation returns deterministic lineage classification and does not mutate source state

#### Scenario: Existing reference remains backward compatible
- **WHEN** an existing `agent_runtime_protocol.v1` mapping contains only the original checkpoint fields
- **THEN** validation and replay succeed with all new fields omitted or defaulted

### Requirement: Capability, context, and admission projections SHALL preserve compatibility and observability contracts

All new protocol fields MUST be additive, nullable or defaultable, and associated with an explicit profile version. Checkpoint history and workspace provenance fields MUST preserve `run_id`, `session_id`, source, causation, and available trace correlation; diagnostics writes MUST continue through `observability/event.RuntimeRecorder`. Equivalent Run and Stream inputs MUST normalize to semantically equivalent provenance outcomes after permitted event-order normalization.

#### Scenario: Provenance diagnostics use the single write path
- **WHEN** a lineage, replay, or workspace drift projection emits an observable decision
- **THEN** its bounded additive fields are recorded through `RuntimeRecorder` without a parallel diagnostics writer

#### Scenario: Run and Stream preserve provenance parity
- **WHEN** equivalent Run and Stream requests receive the same checkpoint and workspace context
- **THEN** normalized decision, reason, lineage, and restore-source fields remain equivalent
