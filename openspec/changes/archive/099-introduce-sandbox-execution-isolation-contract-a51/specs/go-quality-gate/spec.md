## ADDED Requirements

### Requirement: Quality gate SHALL include sandbox execution isolation contract checks
The standard quality gate flow MUST include sandbox execution isolation contract checks for:
- policy resolution semantics (`host|sandbox|deny`),
- fallback behavior (`allow_and_record|deny`),
- capability negotiation semantics (`required_capabilities`),
- session lifecycle semantics (`per_call|per_session`),
- Run/Stream semantic equivalence,
- readiness/admission integration assertions.

Sandbox contract check failures MUST block merge.

#### Scenario: Sandbox contract check fails
- **WHEN** CI or local gate execution observes sandbox semantic mismatch against fixtures/tests
- **THEN** sandbox gate exits non-zero and quality gate reports blocking failure

#### Scenario: Sandbox contract check passes
- **WHEN** CI or local gate execution confirms sandbox semantics align with contract suites
- **THEN** sandbox gate reports success and does not block merge

### Requirement: Sandbox quality gate SHALL include backend compatibility matrix smoke suites
Sandbox gate MUST include backend compatibility matrix smoke coverage for:
- at least one Linux-style sandbox backend path,
- at least one container/job-style backend path,
- `windows_job` path on Windows runners when platform is available.

#### Scenario: Backend matrix suite detects backend-specific semantic drift
- **WHEN** one backend path diverges from canonical sandbox contract semantics
- **THEN** sandbox gate exits non-zero and reports backend-scoped failure classification

### Requirement: Repository SHALL provide sandbox executor conformance harness suites
Repository MUST provide sandbox executor conformance harness suites that validate canonical ExecSpec/ExecResult interoperability and capability negotiation semantics across supported backend adapters.

Conformance harness MUST run in offline deterministic mode and MUST be invocable from quality gate scripts.

#### Scenario: Contributor runs sandbox conformance harness offline
- **WHEN** contributor executes sandbox conformance harness in offline environment
- **THEN** harness validates canonical executor semantics without external network dependency

#### Scenario: Conformance harness detects capability negotiation drift
- **WHEN** backend adapter reports capability semantics inconsistent with canonical contract
- **THEN** harness fails with deterministic conformance classification and blocks gate

### Requirement: Sandbox gate SHALL be available as independent required-check candidate
CI workflow MUST expose sandbox contract validation as an independent status check suitable for branch protection required-check configuration.

#### Scenario: Maintainer configures branch protection for sandbox contract
- **WHEN** maintainer inspects available CI status checks
- **THEN** sandbox contract gate appears as a distinct check candidate
