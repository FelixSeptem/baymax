## ADDED Requirements

### Requirement: Scheduler SHALL preserve compatibility when QoS is disabled
When QoS mode is not enabled, scheduler MUST preserve existing FIFO-compatible claim behavior and retry semantics.

#### Scenario: Existing scheduler integration runs under default config
- **WHEN** host uses scheduler without qos-specific config
- **THEN** scheduler behavior remains compatible with pre-A10 FIFO baseline

### Requirement: Scheduler terminal path SHALL include dead-letter terminal classification
Scheduler terminal outcomes MUST include explicit dead-letter classification when tasks are moved out of normal retry lifecycle.

#### Scenario: Retry-exhausted task enters dead-letter
- **WHEN** dead-letter policy is enabled and retry budget is exhausted
- **THEN** task terminal classification includes dead-letter reason and no further standard queue claims occur
