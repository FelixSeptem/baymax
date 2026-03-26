## ADDED Requirements

### Requirement: Adapter-health findings SHALL participate in cross-domain arbitration with canonical required/optional semantics
Adapter-health findings MUST participate in cross-domain primary-reason arbitration while preserving required/optional semantic distinction.

Required-unavailable findings MUST outrank optional-unavailable/degraded findings within non-timeout buckets.

#### Scenario: Required and optional adapter findings co-exist
- **WHEN** one required adapter is unavailable and one optional adapter is unavailable
- **THEN** arbitration selects required-unavailable branch as higher-priority candidate

#### Scenario: Optional adapter unavailable co-exists with degraded readiness
- **WHEN** optional adapter unavailable and degraded readiness findings co-exist
- **THEN** arbitration applies deterministic same-level tie-break and records conflict when needed
