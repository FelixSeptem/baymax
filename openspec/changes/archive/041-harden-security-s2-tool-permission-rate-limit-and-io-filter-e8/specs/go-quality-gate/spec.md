## ADDED Requirements

### Requirement: Quality gate SHALL include S2 security policy contract checks
The standard CI validation flow MUST include S2 security policy contract checks that validate:
- `namespace+tool` permission deny/allow semantics,
- process-scoped rate-limit deny semantics,
- model input/output filtering deny semantics,
- hot-reload invalid-update rollback semantics.

Failures in S2 security policy contract checks MUST block merge.

#### Scenario: S2 security contract check fails in pull request
- **WHEN** CI runs S2 security contract checks and expected permission/rate-limit/filter/reload behavior diverges from fixtures
- **THEN** security policy gate exits non-zero and pull request cannot pass required validation

#### Scenario: S2 security contract check passes in pull request
- **WHEN** CI runs S2 security contract checks and all expected behaviors match fixtures
- **THEN** security policy gate reports success and does not block merge

### Requirement: Security policy gate SHALL be exposed as independent required-check candidate
The CI workflow MUST expose S2 security policy validation in an independent job suitable for branch-protection required status checks.

#### Scenario: Maintainer configures branch protection for S2
- **WHEN** maintainer reviews available CI status checks
- **THEN** security policy gate appears as a distinct check that can be configured as required