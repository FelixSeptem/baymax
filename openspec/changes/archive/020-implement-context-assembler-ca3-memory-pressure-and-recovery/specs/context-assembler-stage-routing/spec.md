## ADDED Requirements

### Requirement: CA3 pressure decisions SHALL remain semantically equivalent between Run and Stream
For equivalent inputs and configuration, Run and Stream paths MUST produce semantically equivalent CA3 pressure-zone decisions, allowing implementation-level event order differences.

#### Scenario: Equivalent pressure path in Run and Stream
- **WHEN** Run and Stream process equivalent requests under identical CA3 pressure config
- **THEN** both paths report equivalent pressure-zone outcomes in diagnostics

#### Scenario: Equivalent emergency downgrade in Run and Stream
- **WHEN** Run and Stream both enter emergency pressure zone
- **THEN** both paths apply equivalent low-priority rejection semantics and record equivalent downgrade reason classes
