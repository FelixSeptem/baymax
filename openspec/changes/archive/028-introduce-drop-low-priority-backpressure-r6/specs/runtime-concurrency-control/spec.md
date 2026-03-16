## ADDED Requirements

### Requirement: Runtime concurrency config SHALL accept drop_low_priority enum
Runtime MUST treat `drop_low_priority` as a valid enum value for concurrency backpressure configuration in addition to existing modes.

#### Scenario: Config validation accepts drop_low_priority
- **WHEN** configuration sets `concurrency.backpressure=drop_low_priority`
- **THEN** runtime validation passes for enum check
