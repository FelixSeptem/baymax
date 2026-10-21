## ADDED Requirements

### Requirement: Composite source owners SHALL extend durable continuation without weakening direct guarantees

Composite continuation implementations MUST preserve the existing durable
checkpoint envelope, atomic claim, fail-closed lifecycle, Run/Stream parity,
same-host single-writer sequential-reopen boundary, and explicit unsupported
matrix established by the direct and workflow profiles.

#### Scenario: Composite restore preserves deployment scope
- **WHEN** a child continuation is reopened by another Engine on the same
  configured FileStore
- **THEN** restore is allowed only within the declared single-writer sequential
  scope and distributed/multi-writer requests remain unavailable
