## ADDED Requirements

### Requirement: Quality gate SHALL include sandbox adapter conformance contract checks
Standard quality gate flow MUST include sandbox adapter conformance contract checks validating:
- backend profile-pack matrix behavior,
- manifest compatibility enforcement,
- capability negotiation and session lifecycle conformance,
- replay drift assertions for sandbox adapter fixtures.

Sandbox adapter conformance check failures MUST block merge.

#### Scenario: Sandbox adapter conformance check fails
- **WHEN** quality gate detects backend/profile/session/manifest semantic mismatch
- **THEN** sandbox adapter gate exits non-zero and blocks merge

#### Scenario: Sandbox adapter conformance check passes
- **WHEN** quality gate validates sandbox adapter contracts against fixtures
- **THEN** sandbox adapter gate reports success and does not block merge

### Requirement: Sandbox adapter gate SHALL preserve shell and PowerShell parity
Repository MUST provide shell and PowerShell sandbox adapter gate scripts with equivalent blocking semantics.

#### Scenario: Equivalent contract failure on shell and PowerShell gate
- **WHEN** sandbox adapter contract failure occurs in either shell or PowerShell path
- **THEN** both scripts return non-zero and produce equivalent blocking outcome

### Requirement: CI SHALL expose sandbox adapter gate as independent required-check candidate
CI workflow MUST expose sandbox adapter conformance validation as an independent status check suitable for branch-protection required-check configuration.

#### Scenario: Maintainer configures branch protection for sandbox adapter gate
- **WHEN** maintainer reviews available CI status checks
- **THEN** sandbox adapter gate appears as a distinct required-check candidate
