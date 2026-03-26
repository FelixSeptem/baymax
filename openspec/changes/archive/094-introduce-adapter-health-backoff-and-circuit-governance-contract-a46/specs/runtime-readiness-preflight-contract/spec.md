## ADDED Requirements

### Requirement: Runtime readiness preflight SHALL classify adapter-governance outcomes deterministically
Runtime readiness preflight MUST classify adapter-health governance outcomes using canonical findings and MUST preserve strict/non-strict escalation semantics.

At minimum, readiness findings MUST cover:
- circuit-open sustained unavailable path
- half-open degraded probe path
- governance recovery path after successful half-open probes

Canonical finding codes for governance paths MUST remain in `adapter.health.*` namespace.

#### Scenario: Strict mode escalates sustained circuit-open required adapter
- **WHEN** required adapter remains unavailable with circuit held in `open` under strict mode
- **THEN** preflight returns `blocked` with canonical `adapter.health.*` finding code

#### Scenario: Non-strict mode degrades optional adapter during circuit-open window
- **WHEN** optional adapter is unavailable while governance keeps circuit `open`
- **THEN** preflight returns `degraded` and preserves runnable runtime behavior with recorded findings
