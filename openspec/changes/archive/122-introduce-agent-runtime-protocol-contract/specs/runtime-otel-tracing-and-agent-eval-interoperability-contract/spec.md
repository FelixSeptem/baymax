## ADDED Requirements

### Requirement: Trace and eval outputs SHALL preserve Agent Runtime Protocol correlation
Tracing and evaluation outputs MUST preserve canonical protocol `run_id`, `step_id`, source, and available artifact/checkpoint lineage without replacing existing semantic-convention topology or evaluation metric contracts.

#### Scenario: Tool step trace carries protocol correlation
- **WHEN** a mapped protocol tool Step emits a canonical tool span
- **THEN** the span retains the protocol Run and Step correlation while preserving existing OTel semantic attributes
