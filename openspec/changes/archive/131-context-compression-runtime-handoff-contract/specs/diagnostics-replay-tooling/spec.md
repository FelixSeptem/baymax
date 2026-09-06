## ADDED Requirements

### Requirement: Replay tooling SHALL support context handoff fixtures
Replay tooling SHALL accept `context_handoff.v1` fixtures containing source state, handoff record, references, restore result, and Run/Stream comparison, while preserving validation of older fixture versions.

#### Scenario: Valid handoff fixture
- **WHEN** a valid `context_handoff.v1` fixture is supplied
- **THEN** replay validates schema, protected facts, references, fallback semantics, and restore idempotency

### Requirement: Replay drift SHALL use canonical handoff classifications
Replay SHALL classify at least `handoff_fact_loss`, `handoff_reference_loss`, `handoff_cut_invalid`, `handoff_quality_below_threshold`, `handoff_schema_drift`, `handoff_restore_non_idempotent`, and `handoff_run_stream_mismatch`.

#### Scenario: Schema drift
- **WHEN** a fixture requires a field or enum not accepted by the current schema
- **THEN** replay reports `handoff_schema_drift` without mutating runtime state

#### Scenario: Restore lifecycle remains additive
- **WHEN** RuntimeRecorder receives generated, validated, and restored handoff events
- **THEN** the final RunRecord preserves the latest bounded lifecycle plus nullable restore status, operation identity, reason, and source checkpoint fields without storing the handoff body
