## ADDED Requirements

### Requirement: Quality gate SHALL include S4 security delivery contract checks
CI validation flow MUST include S4 security delivery contract checks that verify async delivery behavior, `drop_old` queue policy, retry budget enforcement, Hystrix-style circuit state transitions, and Run/Stream semantic equivalence.
Failures in S4 delivery contract checks MUST block merge.

#### Scenario: S4 delivery contract check fails
- **WHEN** CI runs S4 delivery contracts and observed delivery semantics diverge from contract fixtures
- **THEN** security delivery gate exits non-zero and pull request cannot pass required validation

#### Scenario: S4 delivery contract check passes
- **WHEN** CI runs S4 delivery contracts and all expected semantics are satisfied
- **THEN** security delivery gate reports success and does not block merge

### Requirement: Security delivery gate SHALL be exposed as independent required-check candidate
CI workflow MUST expose S4 delivery validation as an independent job named `security-delivery-gate` that can be configured as branch-protection required check.

#### Scenario: Maintainer configures branch protection for S4
- **WHEN** maintainer reviews available CI checks
- **THEN** `security-delivery-gate` appears as a distinct status check
