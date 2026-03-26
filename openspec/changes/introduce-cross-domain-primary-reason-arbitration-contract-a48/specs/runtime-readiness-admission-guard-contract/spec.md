## ADDED Requirements

### Requirement: Admission guard SHALL consume arbitration-aligned primary reason without reclassification drift
Runtime readiness admission guard MUST consume primary reason output from cross-domain arbitration without introducing per-path reclassification drift.

Admission decision explanation fields MUST preserve:
- primary domain,
- primary code,
- primary source.

#### Scenario: Admission deny consumes blocked primary reason
- **WHEN** admission guard receives blocked-class primary reason from arbitration
- **THEN** deny decision explanation preserves the same primary domain/code/source semantics

#### Scenario: Admission allow-and-record consumes degraded primary reason
- **WHEN** admission guard receives degraded-class primary reason under allow-and-record policy
- **THEN** allow decision explanation preserves arbitration primary reason without remapping
