## ADDED Requirements

### Requirement: Quality gate SHALL include sandbox rollout-governance contract checks
Standard quality gate flow MUST include sandbox rollout-governance contract checks that validate:
- rollout phase transition semantics,
- health budget breach/freeze semantics,
- capacity action mapping (`allow|throttle|deny`),
- Run/Stream semantic equivalence for rollout-governed paths,
- replay fixture drift assertions for `a52.v1`.

Failures in rollout-governance contract checks MUST block merge.

#### Scenario: Rollout-governance contract check fails
- **WHEN** CI or local validation detects mismatch in rollout/freeze/capacity contract behavior
- **THEN** rollout-governance gate exits non-zero and blocks merge

#### Scenario: Rollout-governance contract check passes
- **WHEN** CI or local validation confirms rollout-governance contract behavior matches fixtures
- **THEN** rollout-governance gate reports success and does not block merge

### Requirement: Rollout-governance gate SHALL preserve shell and PowerShell parity
Repository MUST provide shell and PowerShell gate scripts with equivalent blocking semantics for rollout-governance checks.

#### Scenario: Equivalent rollout failure on shell and PowerShell flows
- **WHEN** rollout-governance contract failure is triggered under either shell or PowerShell gate
- **THEN** both scripts return non-zero and produce equivalent blocking outcome

### Requirement: CI SHALL expose rollout-governance gate as independent required-check candidate
CI workflow MUST expose rollout-governance validation as an independent job suitable for branch-protection required status checks.

#### Scenario: Maintainer configures branch protection for rollout-governance gate
- **WHEN** maintainer reviews available CI status checks
- **THEN** rollout-governance gate appears as a distinct required-check candidate
