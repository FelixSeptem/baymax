## ADDED Requirements

### Requirement: Production compaction quality SHALL include handoff recoverability
Production semantic compaction SHALL include handoff schema validity, protected fact coverage, reference resolution, and restore-readiness in its quality result without changing existing compaction modes.

#### Scenario: Recoverability gate
- **WHEN** semantic compaction completes
- **THEN** the quality result contains a deterministic handoff recoverability decision and reason
