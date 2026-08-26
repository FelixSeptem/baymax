## ADDED Requirements

### Requirement: Quality gate SHALL block Agent Runtime Protocol contract drift
The quality gate MUST execute an Agent Runtime Protocol contract check with shell and PowerShell parity. The check MUST validate replay fixtures, canonical correlation and state transitions, source mapping stability, and `control_plane_absent` behavior.

#### Scenario: Protocol gate detects invalid mapping fixture
- **WHEN** the protocol fixture suite contains an invalid lifecycle, correlation, ordering, lineage, or control-plane assertion
- **THEN** both shell and PowerShell gate paths fail with the same deterministic classification
