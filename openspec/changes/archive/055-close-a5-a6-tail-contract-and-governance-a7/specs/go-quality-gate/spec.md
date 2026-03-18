## ADDED Requirements

### Requirement: CI quality gate SHALL include scheduler crash-recovery and takeover contract suite
CI MUST include a dedicated scheduler crash-recovery/takeover contract suite for A6 closure.

The suite MUST cover:
- worker crash + lease expiry takeover,
- duplicate submit/commit idempotency,
- Run/Stream semantic equivalence under scheduler-managed flows.

#### Scenario: Scheduler closure gate runs in CI
- **WHEN** scheduler closure gate executes
- **THEN** recovery/idempotency/equivalence regressions fail the gate before merge
