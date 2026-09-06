## ADDED Requirements

### Requirement: Session history replay SHALL validate handoff continuity
Session history and checkpoint replay SHALL verify that a handoff cut, referenced immutable leaves, and restored next actions preserve lineage and continuity without creating a parallel history source.

#### Scenario: Continuity-preserving replay
- **WHEN** a valid handoff is replayed from its source checkpoint
- **THEN** history lineage and immutable leaves match the source projection and restore is idempotent
