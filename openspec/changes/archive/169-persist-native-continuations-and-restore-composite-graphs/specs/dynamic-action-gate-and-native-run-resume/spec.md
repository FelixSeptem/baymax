## ADDED Requirements

### Requirement: Native dynamic-action checkpoints SHALL have a durable restore profile

In addition to same-Engine behavior, a declared durable profile SHALL persist the normalized request boundary and bounded tool-result references needed for direct continuation. The profile MUST reject opaque callbacks or payloads that cannot be reconstructed by the new Engine and MUST retain the existing input_required action ownership rules.

#### Scenario: Durable profile preserves the action boundary
- **WHEN** a resumable action is paused under the durable profile
- **THEN** a reopened source contains the same run/session/action identity and no completed tool is dispatched again

#### Scenario: Legacy in-memory profile is not cross-process support
- **WHEN** only the Engine-local checkpoint map is configured
- **THEN** capability discovery reports process-local support and a new Engine rejects restore as unavailable
