## ADDED Requirements

### Requirement: Tutorial catalog SHALL include full-chain multi-agent reference example
Tutorial examples MUST include a dedicated full-chain reference example that demonstrates composition across `team + workflow + a2a + scheduler + recovery`.

#### Scenario: User browses tutorial directories
- **WHEN** user checks tutorial index and examples directory
- **THEN** a full-chain multi-agent reference example is listed and discoverable

#### Scenario: User executes full-chain tutorial command
- **WHEN** user runs the documented command for the full-chain example
- **THEN** tutorial runs successfully and demonstrates composed multi-agent path

### Requirement: Full-chain tutorial docs SHALL provide dual-path run guidance
The full-chain tutorial documentation MUST provide both `Run` and `Stream` execution guidance and expected observable outputs.

#### Scenario: User follows Run path documentation
- **WHEN** user executes tutorial in Run mode
- **THEN** documentation-aligned terminal summary output is observable

#### Scenario: User follows Stream path documentation
- **WHEN** user executes tutorial in Stream mode
- **THEN** documentation-aligned streaming output and terminal convergence are observable

### Requirement: Full-chain tutorial SHALL document async-delayed-recovery composition checkpoints
The tutorial documentation MUST call out where async reporting, delayed dispatch, and recovery semantics appear in the reference flow and what markers to verify.

#### Scenario: User validates async/delayed/recovery checkpoints
- **WHEN** user follows tutorial verification steps
- **THEN** user can locate explicit async, delayed, and recovery checkpoints in output or logs

