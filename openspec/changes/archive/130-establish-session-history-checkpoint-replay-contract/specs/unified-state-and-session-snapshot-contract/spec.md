## MODIFIED Requirements

### Requirement: Restore Policy and Idempotency Contract

Restore flow MUST support `strict|compatible` policy modes and MUST remain idempotent across repeated imports of the same snapshot and operation identity. When session-history or checkpoint provenance is supplied, restore MUST validate those references before state mutation. `strict` MUST reject incompatible schema, broken lineage, or cross-session association; `compatible` MAY continue only within the configured compatibility window and MUST record a bounded downgrade action.

#### Scenario: Strict restore blocks incompatible payload
- **WHEN** restore mode is strict and payload contains incompatible schema/segment versions or invalid history/checkpoint lineage
- **THEN** restore MUST stop before state mutation and return canonical conflict code

#### Scenario: Compatible restore records downgrade action
- **WHEN** restore mode is compatible and payload is within the configured compatibility window with valid required lineage
- **THEN** restore MAY continue with bounded downgrade action and MUST record deterministic restore action metadata

#### Scenario: Repeated import is idempotent
- **WHEN** the same snapshot, history/checkpoint context, and operation identity are imported multiple times
- **THEN** resulting runtime state and diagnostics aggregates MUST remain stable without inflation

#### Scenario: Conflicting restore identity fails
- **WHEN** the same operation identity is reused with different normalized history or checkpoint data
- **THEN** restore MUST fail with deterministic replay conflict and MUST NOT mutate the snapshot fact source
