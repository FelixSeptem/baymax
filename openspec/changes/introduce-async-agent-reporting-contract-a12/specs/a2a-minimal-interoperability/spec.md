## ADDED Requirements

### Requirement: A2A SHALL support async submit and independent reporting lifecycle
A2A interoperability MUST support async submit lifecycle where terminal reporting can be delivered independently of `WaitResult`.

#### Scenario: A2A task is submitted asynchronously
- **WHEN** caller submits an A2A task in async mode
- **THEN** A2A returns submission acknowledgment and terminal outcome is later delivered via configured report sink

### Requirement: A2A async reporting SHALL preserve existing lifecycle query compatibility
Async reporting support MUST NOT remove or break existing `status/result` query semantics.

#### Scenario: Caller mixes async reporting and status polling
- **WHEN** async reporting is enabled and caller still polls status/result
- **THEN** status/result queries remain available with consistent terminal semantics

### Requirement: A2A async reporting SHALL preserve canonical correlation metadata
A2A async report payloads MUST preserve canonical correlation fields including `workflow_id`, `team_id`, `step_id`, `task_id`, `agent_id`, and `peer_id` when available.

#### Scenario: Async report is emitted for composed A2A call
- **WHEN** A2A async report is delivered from composed orchestration path
- **THEN** report payload includes canonical cross-domain correlation metadata
