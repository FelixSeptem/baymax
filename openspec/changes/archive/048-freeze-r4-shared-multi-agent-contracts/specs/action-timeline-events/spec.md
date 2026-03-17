## MODIFIED Requirements

### Requirement: Action timeline status enum SHALL be stable and include cancellation semantics
The runtime MUST use normalized timeline status enums: `pending`, `running`, `succeeded`, `failed`, `skipped`, and `canceled`.

For multi-agent domains, domain-specific lifecycle statuses MAY exist internally, but they MUST map deterministically to the normalized timeline status enums before timeline emission and aggregate diagnostics ingestion.

Mandatory mapping for this milestone:
- A2A `submitted` MUST map to normalized timeline status `pending`.

#### Scenario: A2A submitted status is emitted through timeline
- **WHEN** A2A lifecycle transitions to `submitted`
- **THEN** timeline and aggregate diagnostics use normalized status `pending`

## ADDED Requirements

### Requirement: Action timeline SHALL enforce multi-agent reason namespace consistency
Multi-agent timeline reason codes MUST be namespace-qualified by domain and MUST use one of the approved prefixes:
- `team.*`
- `workflow.*`
- `a2a.*`

Unqualified or cross-domain ambiguous reason codes MUST be rejected by contract validation for related changes.

#### Scenario: Teams collect reason is emitted
- **WHEN** Teams orchestration emits collect-path timeline reason
- **THEN** reason code uses `team.collect` namespace

#### Scenario: Workflow retry reason is emitted
- **WHEN** workflow step enters retry path
- **THEN** reason code uses `workflow.retry` namespace

#### Scenario: A2A callback retry reason is emitted
- **WHEN** A2A callback delivery enters retry path
- **THEN** reason code uses `a2a.callback_retry` namespace

### Requirement: Action timeline SHALL use canonical peer field naming for A2A correlation
When A2A timeline events include remote peer context, payload field naming MUST use `peer_id` as canonical key.

#### Scenario: A2A submit event includes remote peer correlation
- **WHEN** runtime emits timeline event for A2A submit path
- **THEN** event payload uses `peer_id` for remote peer identification
