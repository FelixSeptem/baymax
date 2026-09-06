## ADDED Requirements

### Requirement: Pressure compaction SHALL expose handoff eligibility and fallback provenance
Pressure-driven compaction SHALL reuse the existing protected-evidence and provenance rules, expose the selected legal cut and handoff eligibility, and record deterministic fallback provenance when a recoverable handoff cannot be formed.

#### Scenario: Emergency pressure
- **WHEN** pressure enters the emergency zone
- **THEN** compaction preserves the minimum evidence set, reports handoff eligibility, and records the fallback reason if eligibility is false
