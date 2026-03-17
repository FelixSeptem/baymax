## ADDED Requirements

### Requirement: Quality gate SHALL include S3 security-event contract checks
The standard CI validation flow MUST include S3 security-event contract checks that validate deny-only alert triggering, callback dispatch semantics, severity normalization, and Run/Stream semantic equivalence.

Failures in S3 security-event contract checks MUST block merge.

#### Scenario: S3 security-event contract check fails
- **WHEN** CI runs S3 security-event contracts and observed taxonomy/alert semantics diverge from fixtures
- **THEN** security-event gate exits non-zero and pull request cannot pass required validation

#### Scenario: S3 security-event contract check passes
- **WHEN** CI runs S3 security-event contracts and all expected behaviors match fixtures
- **THEN** security-event gate reports success and does not block merge

### Requirement: Security-event gate SHALL be exposed as independent required-check candidate
The CI workflow MUST expose S3 security-event validation in an independent job suitable for branch-protection required status checks.

#### Scenario: Maintainer configures branch protection for S3
- **WHEN** maintainer reviews available CI checks
- **THEN** security-event gate appears as a distinct check that can be configured as required
