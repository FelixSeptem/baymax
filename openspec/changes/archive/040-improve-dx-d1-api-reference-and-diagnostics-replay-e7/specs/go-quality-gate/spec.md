## ADDED Requirements

### Requirement: Quality gate SHALL include diagnostics replay contract check
The standard CI validation flow MUST include a diagnostics replay contract check that validates replay behavior against version-controlled fixtures.

Failures in replay contract validation MUST block merge.

#### Scenario: Replay contract check fails in pull request
- **WHEN** CI runs replay gate and output or reason-code expectations diverge from fixtures
- **THEN** replay gate exits non-zero and pull request cannot pass required validation

#### Scenario: Replay contract check passes in pull request
- **WHEN** CI runs replay gate and fixtures match expected output and reason codes
- **THEN** replay gate reports success and does not block merge

### Requirement: Replay gate SHALL be exposed as independent required-check candidate
The CI workflow MUST expose replay validation in an independent job suitable for branch-protection required status checks.

#### Scenario: Maintainer configures branch protection
- **WHEN** maintainer reviews available status checks
- **THEN** replay gate appears as a distinct check that can be configured as required
