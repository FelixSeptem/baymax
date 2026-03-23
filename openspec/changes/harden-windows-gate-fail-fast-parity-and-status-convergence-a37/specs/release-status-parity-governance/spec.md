## MODIFIED Requirements

### Requirement: Status parity checks SHALL fail fast on semantic drift
If release/progress docs contain semantic conflict with OpenSpec authority status, repository validation MUST fail fast with deterministic non-zero exit status.

Status parity checks MUST propagate native test failure deterministically in both direct `go test` invocation and script-wrapped invocation paths (for example docs consistency gate).

#### Scenario: Roadmap marks archived change as in-progress
- **WHEN** status parity check detects conflict between roadmap text and OpenSpec archive status
- **THEN** validation exits non-zero and blocks completion

#### Scenario: README snapshot omits current active change status updates
- **WHEN** status parity check detects README milestone snapshot does not reflect active change posture
- **THEN** validation exits non-zero and reports status-parity classification

#### Scenario: Docs consistency wrapper executes status parity tests
- **WHEN** docs consistency script runs status-parity tests and the tests fail
- **THEN** docs consistency script exits non-zero deterministically instead of reporting pass
