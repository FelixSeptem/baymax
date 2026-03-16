## ADDED Requirements

### Requirement: Action timeline SHALL include drop_low_priority backpressure reason
Action timeline MUST include reason `backpressure.drop_low_priority` when low-priority dropping is applied under configured backpressure mode.

#### Scenario: Timeline records drop_low_priority reason
- **WHEN** runtime drops low-priority local tool calls under queue pressure
- **THEN** corresponding timeline events include reason `backpressure.drop_low_priority`
