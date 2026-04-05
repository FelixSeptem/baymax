## ADDED Requirements

### Requirement: Quality Gate SHALL Include A67-CTX JIT Context Contract Checks
Standard quality gate MUST execute A67-CTX contract checks as blocking validations in both shell and PowerShell flows.

Repository MUST provide:
- `scripts/check-context-jit-organization-contract.sh`
- `scripts/check-context-jit-organization-contract.ps1`

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** A67-CTX contract checks MUST run as required blocking steps

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent A67-CTX contract checks MUST run as required blocking steps

### Requirement: A67-CTX Gate SHALL Fail Fast on Context Semantics Drift
When A67-CTX suites detect canonical semantic drift, quality gate MUST exit non-zero and block merge.

Semantic drift for this milestone MUST include at minimum:
- reference resolution drift
- isolate handoff drift
- edit-gate threshold drift
- swap-back relevance drift
- lifecycle tiering drift
- recap semantic drift
- replay drift classification mismatch

#### Scenario: Context fixture suite detects edit-gate threshold drift
- **WHEN** equivalent fixture or integration inputs produce non-canonical edit-gate decisions
- **THEN** quality gate MUST fail fast and block validation completion

#### Scenario: Replay suite detects recap semantic drift
- **WHEN** A67-CTX replay validation returns non-canonical recap semantics
- **THEN** quality gate MUST fail fast and block validation completion

### Requirement: A67-CTX Gate SHALL Enforce Context Provider Boundary
A67-CTX gate MUST assert boundary condition `context_provider_sdk_absent` and fail when `context/*` introduces direct dependency on provider official SDK packages.

#### Scenario: Gate detects direct provider SDK import in context package
- **WHEN** A67-CTX scope introduces direct provider SDK import under `context/*`
- **THEN** gate MUST fail with deterministic boundary-violation classification

#### Scenario: Context organization remains within abstraction boundary
- **WHEN** context implementation only uses existing model abstraction interfaces
- **THEN** boundary assertion MUST pass

### Requirement: A67-CTX Impacted Contract Suites Enforcement
Gate execution MUST enforce impacted suites for A67-CTX scope changes and MUST reject merges when required suites are missing or failing.

#### Scenario: ReAct scope requires parity suites
- **WHEN** A67-CTX changes touch ReAct context assembly boundaries
- **THEN** gate MUST require Run/Stream parity suites before merge

#### Scenario: Replay scope requires replay suites
- **WHEN** A67-CTX changes touch fixture parser or drift classification logic
- **THEN** gate MUST require replay contract suites before merge
