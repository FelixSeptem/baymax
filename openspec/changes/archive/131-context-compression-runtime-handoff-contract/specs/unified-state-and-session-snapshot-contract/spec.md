## ADDED Requirements

### Requirement: Snapshot manifests SHALL remain the handoff source of truth
Handoff records SHALL reference existing snapshot/checkpoint/session identifiers and SHALL NOT duplicate their authoritative state; restore SHALL verify manifest compatibility before state mutation.

#### Scenario: Compatible manifest
- **WHEN** a handoff references a valid snapshot manifest and checkpoint
- **THEN** restore resolves the existing sources and preserves their identifiers

#### Scenario: Incompatible manifest
- **WHEN** the referenced manifest version or boundary is incompatible
- **THEN** restore fails deterministically before mutating source state
