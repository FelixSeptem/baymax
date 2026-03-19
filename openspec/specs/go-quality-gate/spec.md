# go-quality-gate Specification

## Purpose
TBD - created by archiving change upgrade-openai-native-stream-mapping. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL provide golangci-lint baseline configuration
The repository MUST include a version-controlled `golangci-lint` configuration file that defines enabled linters, runtime limits, and issue handling defaults for this codebase.

#### Scenario: Developer runs linter locally
- **WHEN** a developer executes `golangci-lint run`
- **THEN** lint behavior follows the shared repository configuration without requiring ad hoc local flags

### Requirement: Quality gate SHALL include golangci-lint in standard verification flow
The standard validation flow MUST include `golangci-lint`, `go test ./...`, `go test -race ./...`, `govulncheck`, and mainline contract test suites so style, correctness, concurrency regressions, dependency vulnerability risks, and cross-module semantic regressions are detected before merge.

`govulncheck` MUST run in strict mode by default, and vulnerability findings MUST fail validation unless explicitly downgraded by controlled configuration.

#### Scenario: Validation in CI or local pre-merge checks
- **WHEN** a change is validated before merge
- **THEN** linter execution, unit tests, race tests, vulnerability scan, and required mainline contract tests are all required checks and failures block completion

#### Scenario: govulncheck finds vulnerabilities in strict mode
- **WHEN** validation runs with default strict scan mode and vulnerabilities are reported
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: Lint profile SHALL align with Go style and safety priorities
The configured quality profile MUST enforce formatting/import conventions, detect common correctness risks, and include concurrency safety auditing practices.

#### Scenario: Code violates configured style or safety rules
- **WHEN** code introduces issues covered by enabled linters or race detection
- **THEN** validation output reports actionable diagnostics tied to file and line locations

### Requirement: Lint configuration changes SHALL be documented
Any newly introduced lint policy and recommended invocation commands MUST be documented under `docs/` for contributor onboarding.

#### Scenario: New contributor sets up development environment
- **WHEN** the contributor reads project documentation
- **THEN** they can run the documented lint and test commands with expected outcomes

### Requirement: Performance regression gate SHALL use relative percentage thresholds
Performance validation MUST evaluate benchmark outcomes using relative percentage change against a documented baseline.

#### Scenario: Benchmark comparison is executed
- **WHEN** benchmark results are compared for a candidate change
- **THEN** acceptance is decided by relative percentage thresholds for throughput and latency metrics

### Requirement: Concurrency safety SHALL be treated as a baseline quality requirement
Concurrency safety checks MUST be mandatory and cannot be bypassed in standard merge flow, including race detection and targeted concurrent diagnostics tests.

#### Scenario: Concurrency safety check fails
- **WHEN** race detection or required concurrent diagnostics tests fail
- **THEN** the change is rejected from merge until safety checks pass

### Requirement: Quality gate SHALL include diagnostics concurrency test coverage
The repository MUST maintain explicit tests for concurrent diagnostics writes, duplicate event replay, and idempotent persistence behavior.

#### Scenario: Diagnostics concurrency suite is executed
- **WHEN** diagnostics-focused concurrent tests run
- **THEN** write deduplication and data integrity guarantees are verified under parallel workloads

### Requirement: Quality-gate scripts SHALL provide cross-platform security scan parity
Repository-provided quality-gate scripts for Linux and PowerShell MUST both execute the same vulnerability scan semantics as CI.

#### Scenario: Linux and PowerShell scripts are executed
- **WHEN** contributors run quality-gate scripts on different platforms
- **THEN** both flows execute equivalent test/lint/race/vuln checks and produce consistent pass/fail semantics

### Requirement: Quality gate SHALL enforce repository hygiene checks
The standard validation flow MUST include repository hygiene checks that reject temporary backup artifacts and stale generated-by-accident files that are outside committed source-of-truth conventions.

#### Scenario: Temporary backup file is tracked
- **WHEN** repository hygiene checks detect files matching banned temporary patterns
- **THEN** validation fails and requires cleanup before merge

### Requirement: Mainline contract coverage SHALL be explicitly traceable
The repository MUST maintain a traceable mapping between required mainline flows and their corresponding contract test cases.

#### Scenario: Contributor reviews test coverage for a critical chain
- **WHEN** contributor inspects quality-gate documentation or test index
- **THEN** contributor can identify which contract test covers each required mainline flow

### Requirement: Quality gate SHALL include CA4 benchmark regression checks
The standard validation flow MUST include CA4-related benchmark checks evaluated by relative percentage thresholds, including P95 latency constraints.

#### Scenario: CA4 benchmark regression exceeds threshold
- **WHEN** candidate benchmark result exceeds configured relative degradation or P95 threshold
- **THEN** validation fails and change cannot be completed until regression is mitigated or explicitly re-baselined

### Requirement: CA4 benchmark policy SHALL align with documented performance rules
CA4 benchmark acceptance criteria MUST align with repository performance policy and remain documented for local and CI execution parity.

#### Scenario: Contributor runs CA4 performance validation
- **WHEN** contributor follows documented commands
- **THEN** contributor can reproduce the same pass/fail semantics locally and in CI

### Requirement: CI workflow SHALL pin critical lint tooling version
The default CI workflow MUST use a pinned `golangci-lint` version rather than floating `latest` to ensure reproducible validation behavior across time.

#### Scenario: CI executes lint job on different dates
- **WHEN** the same commit is validated at different times
- **THEN** lint pass/fail semantics remain stable unless repository-owned version pin is intentionally changed

### Requirement: CI workflow SHALL avoid duplicated quality-gate stages
The default CI workflow MUST avoid duplicate execution of repository hygiene checks when the quality-gate script already includes the same check.

#### Scenario: Workflow runs standard validation
- **WHEN** CI executes the default quality gate
- **THEN** repository hygiene is executed exactly once in the canonical validation path

### Requirement: CI workflow SHALL declare least-privilege permissions and timeout
The default CI workflow MUST explicitly declare minimum required GitHub Actions permissions and define job timeout to prevent unbounded execution.

#### Scenario: Workflow job starts on pull request
- **WHEN** CI job initializes
- **THEN** the job runs under explicit least-privilege permissions and bounded timeout settings

### Requirement: Quality gate SHALL enforce contribution template completeness checks
The standard validation flow MUST execute contribution-template completeness checks for pull requests, and missing required sections or unchecked mandatory checklist items MUST fail validation.

This check MUST be configured as a required CI status check in the default merge flow.

#### Scenario: Pull request misses required template section
- **WHEN** CI validates a pull request body missing one or more required sections
- **THEN** the contribution-template check exits non-zero and blocks merge

#### Scenario: Pull request omits mandatory checklist confirmation
- **WHEN** CI validates a pull request with required checklist items left unchecked or removed
- **THEN** the contribution-template check exits non-zero and blocks merge

#### Scenario: Pull request satisfies template requirements
- **WHEN** CI validates a pull request containing all required sections and mandatory checklist confirmations
- **THEN** the contribution-template check passes and does not block merge

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

### Requirement: CI quality gate SHALL include scheduler crash-recovery and takeover contract suite
CI MUST include a dedicated scheduler crash-recovery/takeover contract suite for A6 closure.

The suite MUST cover:
- worker crash + lease expiry takeover,
- duplicate submit/commit idempotency,
- Run/Stream semantic equivalence under scheduler-managed flows.

#### Scenario: Scheduler closure gate runs in CI
- **WHEN** scheduler closure gate executes
- **THEN** recovery/idempotency/equivalence regressions fail the gate before merge

### Requirement: Quality gate SHALL include composer contract suite in shared multi-agent gate
The quality gate MUST include composer integration contract tests in the existing shared multi-agent gate pipeline, rather than introducing a disconnected parallel gate.

#### Scenario: CI executes multi-agent shared-contract gate after A8
- **WHEN** CI runs shared multi-agent contract gate scripts
- **THEN** composer contract suites run as blocking checks within the same gate path

### Requirement: Composer contract suite SHALL cover fallback and semantic equivalence
Composer contract suites MUST cover scheduler fallback-to-memory behavior, Run/Stream semantic equivalence, and replay/idempotency behavior for scheduler-managed child execution.

#### Scenario: Regression introduces Run/Stream summary divergence
- **WHEN** equivalent composer-managed Run and Stream requests produce non-equivalent aggregate summaries
- **THEN** composer contract suite fails and blocks merge

### Requirement: Shared multi-agent gate SHALL include session recovery contract suite
Quality gate contract MUST include session recovery and deterministic replay tests in the existing shared multi-agent gate scripts.

Required gate path:
- `go test ./integration -run '^TestComposerRecovery' -count=1`

#### Scenario: CI runs shared multi-agent gate after recovery rollout
- **WHEN** CI executes `check-multi-agent-shared-contract.*`
- **THEN** recovery/replay contract suites run as blocking checks in the same gate path

### Requirement: Recovery gate SHALL block semantic divergence and conflict-policy regressions
Recovery contract suite MUST fail on Run/Stream semantic divergence, replay counter inflation, or non-fail-fast conflict handling.

#### Scenario: Regression changes conflict handling away from fail-fast
- **WHEN** recovery conflict handling regresses to non-fail-fast behavior
- **THEN** recovery contract tests fail and block merge

### Requirement: Quality gate SHALL include scheduler QoS and dead-letter contract suites
The shared multi-agent quality gate MUST include scheduler qos/fairness/dead-letter contract tests as blocking checks.

#### Scenario: CI executes shared multi-agent gate after A10 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** scheduler QoS and dead-letter suites execute as blocking checks in that gate path

### Requirement: QoS gate SHALL block fairness and dead-letter regressions
QoS contract suites MUST fail on fairness-window violations, dead-letter transfer regressions, or retry-backoff policy drift.

#### Scenario: Regression bypasses fairness threshold
- **WHEN** high-priority claims exceed configured fairness window without yielding
- **THEN** QoS contract tests fail and block merge

### Requirement: Quality gate SHALL include shared synchronous invocation contract tests
The shared multi-agent quality gate MUST include contract suites validating shared synchronous invocation behavior across orchestration integration paths.

#### Scenario: CI executes shared multi-agent contract gate for A11
- **WHEN** CI runs shared multi-agent contract scripts after A11 rollout
- **THEN** shared synchronous invocation contract tests are executed as blocking checks

### Requirement: Synchronous invocation gate SHALL block semantic divergence
Shared synchronous invocation contract suite MUST fail on timeout/cancellation precedence regressions, error-layer normalization drift, or Run/Stream semantic divergence.

#### Scenario: Regression changes cancellation precedence in one module path
- **WHEN** one orchestration path diverges from shared synchronous invocation cancellation semantics
- **THEN** contract suite fails and blocks merge

### Requirement: Quality gate SHALL include async reporting contract suites
Shared multi-agent quality gate MUST include async reporting contract suites as blocking checks.

#### Scenario: CI executes shared gate after A12 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** async reporting contract tests are executed as blocking checks in the same gate path

### Requirement: Async reporting gate SHALL block retry-idempotency regressions
Async reporting contract suites MUST fail on delivery retry drift, dedup regression, or replay-idempotency violations.

#### Scenario: Regression causes duplicate async reports to inflate aggregates
- **WHEN** duplicate async reports increase logical counters
- **THEN** contract suite fails and blocks merge

### Requirement: Quality gate SHALL include delayed-dispatch contract suites
Shared multi-agent quality gate MUST include delayed-dispatch contract suites as blocking checks.

#### Scenario: CI executes shared gate after delayed-dispatch rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** delayed-dispatch suites execute as blocking checks in the same gate path

### Requirement: Delayed-dispatch gate SHALL block early-claim and recovery-drift regressions
Delayed-dispatch contract suites MUST fail on early-claim regressions, delayed-ready ordering drift, or restore-time semantic drift.

#### Scenario: Regression claims task before not_before
- **WHEN** scheduler claims a delayed task before `not_before`
- **THEN** delayed-dispatch contract suite fails and blocks merge

### Requirement: Shared multi-agent gate SHALL include A12/A13 cross-mode closure matrix suites
Quality gate MUST include blocking cross-mode contract suites that cover sync/async/delayed communication semantics under Run/Stream and required qos/recovery key paths.

#### Scenario: CI executes shared gate after A14 closure
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** required A12/A13 cross-mode suites execute as blocking checks in the shared gate path

### Requirement: Shared gate SHALL block async and delayed reason-taxonomy drift
Quality gate contract checks MUST fail when required A12/A13 canonical reason taxonomy is incomplete or renamed without synchronized contract updates.

#### Scenario: Contract drift removes delayed reason from shared snapshot
- **WHEN** delayed canonical reason is missing from shared contract snapshot validation
- **THEN** gate fails with explicit taxonomy-drift classification and blocks merge

### Requirement: Mainline contract index SHALL trace A12/A13 closure matrix coverage
Repository contract index MUST map A12/A13 closure matrix flows to concrete test entries and remain synchronized with gate suites.

#### Scenario: Contributor audits A12/A13 closure coverage
- **WHEN** contributor reviews mainline contract index and shared-gate suites
- **THEN** each required cross-mode matrix row has traceable test mapping and no missing index entry

### Requirement: Quality gate SHALL include workflow graph composability contract suites
Shared multi-agent quality gate MUST include workflow graph composability contract suites as blocking checks.

#### Scenario: CI executes shared contract gate after A15 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** workflow graph composability suites run as blocking checks in the same shared gate path

### Requirement: Workflow graph composability gate SHALL block compile-boundary regressions
Workflow graph composability contract suites MUST fail on depth-limit regression, alias/id collision acceptance, template-scope violation acceptance, or forbidden kind-override acceptance.

#### Scenario: Regression accepts kind override in subgraph instance
- **WHEN** test suite detects forbidden `kind` override no longer fails validation
- **THEN** quality gate fails and blocks merge

### Requirement: Mainline contract index SHALL map workflow graph composability coverage
Mainline contract index MUST include traceable mapping for A15 core scenarios: expansion determinism, compile fail-fast, Run/Stream equivalence, and resume consistency.

#### Scenario: Contributor audits A15 coverage
- **WHEN** contributor checks mainline contract index
- **THEN** each required A15 contract row maps to concrete test cases

### Requirement: Shared multi-agent quality gate SHALL include collaboration primitive contract suites
Shared multi-agent quality gate MUST include collaboration primitive contract suites as blocking checks.

#### Scenario: CI executes shared gate after collaboration rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** collaboration primitive suites run as blocking checks in the same shared gate path

### Requirement: Collaboration primitive gate SHALL block semantic drift across modes
Collaboration primitive contract suites MUST fail on semantic drift for sync/async/delayed mode composition, Run/Stream equivalence, or replay-idempotency behavior.

#### Scenario: Regression causes mode-dependent terminal divergence
- **WHEN** same collaboration primitive request produces divergent terminal semantics across modes
- **THEN** contract gate fails and blocks merge

### Requirement: Mainline contract index SHALL map collaboration primitive coverage
Mainline contract index MUST provide traceable mapping for collaboration primitive coverage including handoff, delegation, aggregation strategy semantics, and failure-policy behavior.

#### Scenario: Contributor audits collaboration primitive coverage
- **WHEN** contributor checks contract index and integration suites
- **THEN** each required collaboration primitive scenario has a concrete test mapping

### Requirement: Shared quality gate SHALL include long-running recovery-boundary contract suites
Shared multi-agent quality gate MUST include long-running recovery-boundary suites as blocking checks.

#### Scenario: CI executes shared gate after A17 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** recovery-boundary suites execute as blocking checks in the shared gate path

### Requirement: Recovery-boundary gate SHALL block rewind and unbounded-reentry regressions
Recovery-boundary contract suites MUST fail on rewind of terminal tasks, unbounded timeout reentry, or boundary policy drift.

#### Scenario: Regression re-executes terminal task after restore
- **WHEN** contract suite detects restored terminal task being dispatched again
- **THEN** quality gate fails and blocks merge

### Requirement: Mainline contract index SHALL map recovery-boundary matrix coverage
Mainline contract index MUST include traceable coverage rows for crash/restart/replay/timeout boundary scenarios.

#### Scenario: Contributor audits A17 coverage mappings
- **WHEN** contributor inspects contract index for recovery-boundary entries
- **THEN** each required boundary scenario maps to concrete test cases

