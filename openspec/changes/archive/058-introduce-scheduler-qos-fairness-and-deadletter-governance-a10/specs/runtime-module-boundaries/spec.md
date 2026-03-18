## ADDED Requirements

### Requirement: Scheduler QoS governance SHALL remain inside scheduler/runtime boundaries
QoS/fairness/dead-letter governance MUST remain implemented within scheduler/runtime boundaries and MUST NOT introduce forbidden dependencies or direct diagnostics store writes.

#### Scenario: QoS features are added to scheduler module
- **WHEN** scheduler QoS and dead-letter logic is implemented
- **THEN** dependency direction remains compliant with module boundary contract

### Requirement: QoS observability SHALL keep RuntimeRecorder single-writer path
Scheduler QoS and dead-letter observability MUST be emitted as standard events and persisted through RuntimeRecorder single-writer ingestion.

#### Scenario: QoS transitions update diagnostics
- **WHEN** qos/fairness/dlq events are emitted
- **THEN** diagnostics persistence uses RuntimeRecorder mapping and not direct writes from scheduler state logic
