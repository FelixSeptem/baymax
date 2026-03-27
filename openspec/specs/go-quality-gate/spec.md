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

For both shell and PowerShell gate implementations, each required check MUST propagate failure deterministically (non-zero exit). Quality gate MUST NOT continue with success reporting after an unhandled required-check failure.

#### Scenario: Validation in CI or local pre-merge checks
- **WHEN** a change is validated before merge
- **THEN** linter execution, unit tests, race tests, vulnerability scan, and required mainline contract tests are all required checks and failures block completion

#### Scenario: govulncheck finds vulnerabilities in strict mode
- **WHEN** validation runs with default strict scan mode and vulnerabilities are reported
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Required native command fails in PowerShell gate
- **WHEN** a required command in `check-quality-gate.ps1` exits non-zero
- **THEN** the script exits non-zero deterministically and does not report overall gate success

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

Cross-platform parity MUST include deterministic failure propagation semantics: equivalent check failures in shell and PowerShell MUST produce equivalent blocking outcomes.

#### Scenario: Linux and PowerShell scripts are executed
- **WHEN** contributors run quality-gate scripts on different platforms
- **THEN** both flows execute equivalent test/lint/race/vuln checks and produce consistent pass/fail semantics

#### Scenario: Docs consistency check fails under PowerShell flow
- **WHEN** `check-docs-consistency.ps1` detects status-parity drift or contract-doc mismatch
- **THEN** PowerShell gate returns non-zero and quality gate treats it as blocking failure

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

### Requirement: Shared quality gate SHALL include unified query contract suites
The shared multi-agent quality gate MUST include blocking contract suites for unified diagnostics query behavior.

The suites MUST run in repository gate scripts for both shell and PowerShell flows.

#### Scenario: CI runs shared quality gate after unified query rollout
- **WHEN** CI executes shared multi-agent contract gate scripts
- **THEN** unified query contract suites run as required blocking checks

#### Scenario: Local contributor runs PowerShell shared gate
- **WHEN** contributor executes `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
- **THEN** unified query contract suites are executed with equivalent blocking semantics

### Requirement: Unified query contract suites SHALL enforce canonical query semantics
Contract tests MUST cover at least:
- multi-filter `AND` semantics,
- default pagination `page_size=50`,
- maximum page size `200` with fail-fast on invalid values,
- default sort `time desc`,
- opaque cursor pagination behavior and invalid cursor fail-fast behavior,
- non-existent `task_id` returns empty result set without error.

#### Scenario: Regression changes filter semantics to OR
- **WHEN** implementation returns records matching any filter instead of all filters
- **THEN** contract suite fails and blocks merge

#### Scenario: Regression changes missing task behavior to error
- **WHEN** implementation returns error for unmatched but syntactically valid `task_id`
- **THEN** contract suite fails and blocks merge

### Requirement: Mainline contract index SHALL trace unified query coverage
Repository documentation and contract index MUST include traceable mapping from unified query semantic rows to concrete test cases and gate script entries.

#### Scenario: Contributor audits unified query coverage
- **WHEN** contributor inspects mainline contract index after A18
- **THEN** each required unified query semantic row maps to concrete test and gate path

### Requirement: Quality gate SHALL include multi-agent performance regression checks
The standard repository quality gate MUST execute multi-agent mainline benchmark regression checks as blocking validation.

This check MUST run in both shell and PowerShell quality-gate scripts to preserve cross-platform parity.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** multi-agent performance regression check is executed as a required step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent multi-agent performance regression check is executed as a required step

### Requirement: Quality gate SHALL block merge on multi-agent performance threshold regression
When multi-agent benchmark regression check reports degradation beyond configured thresholds, quality gate MUST fail and block merge.

#### Scenario: Candidate regression exceeds configured threshold
- **WHEN** one or more multi-agent benchmark metrics exceed configured degradation limits
- **THEN** quality gate exits non-zero and validation is blocked

#### Scenario: Candidate regression stays within configured thresholds
- **WHEN** all required multi-agent benchmark metrics remain within configured limits
- **THEN** quality gate proceeds without performance-regression failure

### Requirement: CI quality workflow SHALL preserve local parity for multi-agent performance gate
Default CI workflow MUST invoke quality gate steps that include the same multi-agent performance regression semantics used in local scripts.

#### Scenario: CI executes test-and-lint quality path
- **WHEN** CI runs the default quality-gate job
- **THEN** multi-agent performance regression check behavior matches local quality-gate scripts

### Requirement: Quality gate SHALL include full-chain example smoke validation
The standard quality gate MUST execute smoke validation for the full-chain multi-agent reference example as a blocking step.

This validation MUST be included in both shell and PowerShell quality-gate scripts.

#### Scenario: Shell quality gate runs
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** full-chain example smoke validation runs as a required blocking step

#### Scenario: PowerShell quality gate runs
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent full-chain example smoke validation runs as a required blocking step

### Requirement: Example smoke gate SHALL fail fast on execution drift
If full-chain example smoke command fails, times out, or misses required success markers, quality gate MUST fail and return non-zero status.

#### Scenario: Example execution fails
- **WHEN** full-chain example smoke command exits with non-zero status
- **THEN** quality gate fails and blocks merge

#### Scenario: Example output misses required convergence markers
- **WHEN** smoke validation cannot find required success/checkpoint markers
- **THEN** quality gate fails with explicit example-smoke classification

### Requirement: Mainline index SHALL trace full-chain example smoke coverage
The repository MUST update mainline contract/index documentation to include traceability between full-chain example smoke checks and corresponding gate paths.

#### Scenario: Contributor audits full-chain example validation mapping
- **WHEN** contributor inspects mainline index after A20
- **THEN** full-chain example smoke check has explicit mapping to quality-gate execution path

### Requirement: Quality gate SHALL validate adapter template and migration-doc consistency
The repository quality validation flow MUST verify that external adapter template documentation and migration mapping indexes are synchronized with declared navigation entries.

Validation MUST run through existing docs consistency and contribution check paths.

#### Scenario: Docs index misses adapter mapping entry
- **WHEN** adapter template docs are added or renamed without index synchronization
- **THEN** docs consistency or contribution checks fail and block validation

#### Scenario: Migration mapping link is stale
- **WHEN** migration mapping reference points to missing or moved document path
- **THEN** validation fails with explicit documentation consistency error

### Requirement: Quality gate SHALL keep traceability for adapter migration guidance
Mainline documentation checks MUST preserve traceability between adapter templates, migration mapping docs, and repository entry points.

#### Scenario: Maintainer audits adapter onboarding coverage
- **WHEN** maintainer reviews contribution check outputs and docs index
- **THEN** template and migration mapping paths are traceable from repository documentation entry points

### Requirement: Quality gate SHALL include adapter conformance validation as blocking step
The standard quality gate MUST execute adapter conformance validation and treat failures as blocking.

This validation MUST be integrated into both shell and PowerShell quality-gate paths.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter conformance validation is executed as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter conformance validation is executed as required blocking step

### Requirement: Adapter conformance gate SHALL fail fast and return deterministic non-zero status
If any conformance case fails, quality gate MUST fail fast and return deterministic non-zero status without continuing as success.

#### Scenario: Conformance case fails during validation
- **WHEN** one adapter conformance scenario reports semantic mismatch
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: All conformance cases pass
- **WHEN** all required adapter conformance scenarios pass
- **THEN** quality gate proceeds without adapter conformance failure

### Requirement: Mainline contract index SHALL map adapter conformance coverage and gate paths
Repository documentation MUST map adapter conformance scenarios to concrete test entries and gate scripts for traceability.

#### Scenario: Maintainer audits adapter contract coverage
- **WHEN** maintainer inspects mainline contract index after A22
- **THEN** adapter conformance rows map to concrete harness test entries and quality-gate script paths

### Requirement: Quality gate SHALL include adapter scaffold drift validation as blocking step
The repository quality gate MUST execute adapter scaffold drift validation and MUST treat failures as blocking.

This validation MUST be integrated into both `scripts/check-quality-gate.sh` and `scripts/check-quality-gate.ps1` with equivalent semantics.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter scaffold drift validation is executed as a required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** adapter scaffold drift validation is executed with equivalent blocking behavior

### Requirement: Scaffold drift validation SHALL fail fast with deterministic non-zero status
If generated scaffold output diverges from repository source-of-truth templates or expected fixture mapping, drift validation MUST fail fast and return non-zero status.

#### Scenario: Template drift is detected
- **WHEN** drift validation detects mismatch between generated scaffold and committed expectation
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Scaffold output matches source-of-truth
- **WHEN** drift validation confirms all required scaffold outputs are aligned
- **THEN** quality gate continues without scaffold drift failure

### Requirement: Quality gate SHALL preserve traceability between scaffold generation and conformance bootstrap checks
Repository validation flow MUST keep traceable linkage between scaffold generation outputs and adapter conformance bootstrap coverage.

#### Scenario: Maintainer audits scaffold-conformance traceability
- **WHEN** maintainer reviews quality-gate scripts and contract index
- **THEN** maintainer can identify how scaffold drift checks and conformance bootstrap checks map to concrete validation entries

### Requirement: Quality gate SHALL include pre-1 governance consistency checks
The standard quality gate MUST validate pre-1 governance consistency across roadmap and versioning documentation when repository remains in `0.x` phase.

This validation MUST run through repository docs consistency paths for both shell and PowerShell workflows.

#### Scenario: Contributor runs docs consistency in shell path
- **WHEN** contributor executes `bash scripts/check-docs-consistency.sh`
- **THEN** pre-1 governance consistency checks are executed as required validation

#### Scenario: Contributor runs docs consistency in PowerShell path
- **WHEN** contributor executes `pwsh -File scripts/check-docs-consistency.ps1`
- **THEN** equivalent pre-1 governance consistency checks are executed

### Requirement: Governance consistency check SHALL fail fast on stage-conflict drift
If governance docs contain semantic conflicts between pre-1 posture and stable-release claims, the docs consistency check MUST fail fast and return non-zero status.

#### Scenario: Roadmap claims stable-release posture while versioning remains pre-1
- **WHEN** docs consistency check detects conflicting release-stage semantics
- **THEN** validation exits non-zero and blocks merge

#### Scenario: Governance docs remain semantically aligned
- **WHEN** roadmap and versioning docs consistently express pre-1 posture
- **THEN** docs consistency validation passes without governance-stage failure

### Requirement: Quality gate SHALL include release status parity validation for progress docs
Repository docs consistency checks MUST validate status parity between OpenSpec authority sources and contributor-facing progress docs.

This validation MUST be executed in both shell and PowerShell documentation consistency paths and treated as blocking in quality gate.

#### Scenario: Shell docs consistency path executes
- **WHEN** contributor runs `bash scripts/check-docs-consistency.sh`
- **THEN** release status parity validation runs and failures return non-zero

#### Scenario: PowerShell docs consistency path executes
- **WHEN** contributor runs `pwsh -File scripts/check-docs-consistency.ps1`
- **THEN** equivalent release status parity validation runs with same blocking semantics

### Requirement: Quality gate SHALL include core module README richness validation
Repository docs consistency checks MUST validate required section baseline for covered core module README files.

Failures in module README richness validation MUST fail quality gate.

#### Scenario: Covered module README misses required section
- **WHEN** docs consistency checks detect missing required section marker in covered module README
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Covered module READMEs satisfy richness baseline
- **WHEN** all covered module READMEs include required sections or explicit N/A markers
- **THEN** docs consistency checks pass without module-readme-richness failure

### Requirement: Mainline contract index SHALL map status parity and module README gates
Mainline contract documentation MUST map status parity and module README richness checks to concrete tests or script entries.

#### Scenario: Maintainer audits governance gate traceability
- **WHEN** maintainer inspects `docs/mainline-contract-test-index.md`
- **THEN** maintainer can identify status parity and module README richness gate paths and corresponding check entries

### Requirement: Quality gate SHALL include adapter manifest contract validation as blocking step
The standard quality gate MUST execute adapter manifest contract validation and MUST treat failures as blocking.

This validation MUST be integrated into both shell and PowerShell quality-gate paths.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter manifest contract validation runs as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter manifest contract validation runs as required blocking step

### Requirement: Manifest contract gate SHALL fail fast with deterministic non-zero status
If manifest schema, compatibility range, or required capability checks fail, validation MUST fail fast and return deterministic non-zero status.

#### Scenario: Manifest compatibility check fails
- **WHEN** manifest contract validation detects incompatible `baymax_compat` or invalid semver expression
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Manifest contract checks pass
- **WHEN** all required manifest schema and compatibility checks pass
- **THEN** quality gate proceeds without manifest-gate failure

### Requirement: Quality gate SHALL include adapter capability negotiation contract checks
The standard quality gate MUST execute adapter capability negotiation contract checks and treat failures as blocking.

This check MUST run in both shell and PowerShell quality-gate flows.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter capability negotiation contract checks run as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter capability negotiation contract checks run as required blocking step

### Requirement: Capability negotiation gate SHALL fail fast on semantic drift
If required-capability fail-fast behavior, optional-downgrade behavior, strategy override semantics, or Run/Stream equivalence regresses, capability negotiation validation MUST fail fast and return deterministic non-zero status.

#### Scenario: Regression changes required-capability failure semantics
- **WHEN** contract checks detect required capability missing no longer fails deterministically
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Regression changes Run/Stream negotiation equivalence
- **WHEN** contract checks detect negotiation outcome divergence between Run and Stream
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: Quality gate SHALL include adapter contract replay validation
The standard quality gate MUST execute adapter contract replay validation and treat failures as blocking.

This validation MUST run in both shell and PowerShell gate paths.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter contract replay validation executes as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter contract replay validation executes as required blocking step

### Requirement: Contract replay gate SHALL fail fast on profile drift
If replay fixtures diverge from runtime outputs for supported profile versions, validation MUST fail fast and return deterministic non-zero status.

#### Scenario: Replay detects taxonomy drift
- **WHEN** replay validation detects reason taxonomy output differs from fixture baseline
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Replay detects profile compatibility window mismatch
- **WHEN** replay validation detects unsupported profile handling diverges from contract expectations
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: Shared multi-agent gate SHALL include Task Board query contract suites
The shared multi-agent contract gate MUST execute Task Board query contract suites as blocking checks.

The gate MUST cover at least filter semantics, pagination/cursor determinism, invalid-input fail-fast behavior, and memory/file backend parity.

#### Scenario: Contributor runs shared multi-agent gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** Task Board query contract suites are executed as required blocking checks

#### Scenario: Contributor runs shared multi-agent gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent Task Board query contract suites are executed as required blocking checks

### Requirement: Shared multi-agent gate SHALL include mailbox contract suites
The shared multi-agent contract gate MUST execute mailbox contract suites as blocking checks.

The mailbox suites MUST cover:
- envelope validation and idempotency,
- ack/nack/retry/ttl/dlq lifecycle semantics,
- sync/async/delayed convergence through mailbox,
- mailbox query pagination/cursor deterministic behavior,
- memory/file backend parity,
- mailbox worker lifecycle execution semantics,
- mailbox worker default policy semantics (`enabled=false`, `poll_interval=100ms`, `handler_error_policy=requeue`),
- mailbox worker lease/reclaim semantics (`inflight_timeout=30s`, `heartbeat_interval=5s`, `reclaim_on_consume=true`, `panic_policy=follow_handler_error_policy`),
- mailbox lifecycle canonical reason taxonomy drift detection (including `lease_expired`).

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** mailbox contract suites are executed as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent mailbox contract suites are executed as required blocking checks

#### Scenario: Worker crash or panic recovery semantics regress
- **WHEN** contract suites detect stale in-flight reclaim or panic-recover behavior drift from contract
- **THEN** shared quality gate fails and blocks merge

#### Scenario: Regression introduces non-canonical mailbox lifecycle reason
- **WHEN** contract suites detect mailbox lifecycle reason code outside canonical taxonomy without synchronized contract update
- **THEN** shared quality gate fails and blocks merge

### Requirement: Quality gate SHALL track mailbox migration as canonical multi-agent path
Quality gate and contract index mapping MUST treat mailbox path as canonical for sync/async/delayed coordination flows after migration.

#### Scenario: Maintainer audits shared contract index after mailbox rollout
- **WHEN** maintainer reviews gate scripts and mainline contract index
- **THEN** mailbox-based rows are canonical and legacy path mapping is marked deprecated

### Requirement: Shared multi-agent gate SHALL include async-await lifecycle contract suites
The shared multi-agent quality gate MUST execute async-await lifecycle contract suites as blocking checks in both shell and PowerShell gate paths.

Required coverage MUST include:
- accepted-to-awaiting-report lifecycle transition,
- timeout terminalization behavior,
- late-report drop-and-record behavior,
- duplicate/replay idempotency behavior,
- Run/Stream semantic equivalence,
- memory/file backend parity.

#### Scenario: Shell shared gate executes async-await suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** async-await lifecycle suites are executed as blocking checks

#### Scenario: PowerShell shared gate executes async-await suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent async-await lifecycle suites are executed as blocking checks

### Requirement: Async-await gate SHALL block lifecycle semantic regressions
If lifecycle state transition, timeout convergence, late-report policy, or replay-idempotency semantics drift from contract, shared gate MUST fail fast and return non-zero status.

#### Scenario: Regression changes late-report policy behavior
- **WHEN** contract suite detects late report mutates an already terminal business outcome
- **THEN** shared quality gate fails and blocks merge

### Requirement: Shared multi-agent gate SHALL include async-await reconcile contract suites
The shared multi-agent quality gate MUST execute async-await reconcile contract suites as blocking checks in both shell and PowerShell shared-gate paths.

Required coverage MUST include:
- callback-loss reconcile fallback convergence,
- first-terminal-wins arbitration and conflict recording,
- `not_found -> keep_until_timeout` behavior,
- Run/Stream semantic equivalence,
- memory/file backend parity,
- replay idempotency for callback/poll mixed events.

#### Scenario: Shell shared gate executes reconcile suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** async-await reconcile suites run as required blocking checks

#### Scenario: PowerShell shared gate executes reconcile suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent async-await reconcile suites run as required blocking checks

### Requirement: Reconcile gate SHALL fail fast on terminal-arbitration or fallback semantic drift
If contract suites detect regression in first-terminal-wins arbitration, conflict recording, not-found timeout behavior, or replay-idempotency semantics, shared gate MUST fail fast and return non-zero status.

#### Scenario: Regression allows second terminal to overwrite first terminal
- **WHEN** contract suite observes later callback/poll terminal overwriting first committed terminal state
- **THEN** shared quality gate fails and blocks merge

### Requirement: Shared multi-agent gate SHALL include collaboration retry contract suites
The shared multi-agent quality gate MUST execute collaboration retry contract suites as blocking checks in both shell and PowerShell shared-gate paths.

Required coverage MUST include:
- retry-disabled default behavior,
- bounded retry with exponential backoff+jitter under enabled policy,
- `retry_on=transport_only` classification behavior,
- scheduler-managed single-owner retry behavior (no compounded retries),
- Run/Stream semantic equivalence and replay-idempotent aggregate behavior.

#### Scenario: Shell shared gate executes collaboration retry suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** collaboration retry suites are executed as required blocking checks

#### Scenario: PowerShell shared gate executes collaboration retry suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent collaboration retry suites are executed as required blocking checks

### Requirement: Collaboration retry gate SHALL fail fast on retry-policy semantic drift
If contract suites detect retry-boundary drift, retry-classification drift, or compounded retry behavior drift, shared gate MUST fail fast and return non-zero status.

#### Scenario: Regression introduces compounded primitive+scheduler retries
- **WHEN** contract suite observes one logical failure triggering both primitive retry and scheduler retry loops simultaneously
- **THEN** shared quality gate fails and blocks merge

### Requirement: Quality gate SHALL block legacy direct invoke API reintroduction
The shared multi-agent quality gate and default quality gate MUST include canonical-only checks that block:
- re-exposing legacy direct invoke public APIs for sync/async orchestration paths,
- reintroducing cross-module usage that bypasses mailbox canonical entrypoints.

Canonical-only checks MUST be treated as blocking validation in both shell and PowerShell quality workflows.

#### Scenario: Change reintroduces direct invoke public API surface
- **WHEN** validation detects legacy direct invoke APIs are reintroduced as supported public entrypoints
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Change keeps mailbox canonical entrypoints only
- **WHEN** validation confirms sync/async/delayed orchestration calls route through mailbox canonical entrypoints
- **THEN** canonical-only checks pass without introducing additional failures

### Requirement: Shared gate SHALL include task-board manual-control contract suites
The shared multi-agent contract gate MUST execute task-board manual-control suites as blocking checks.

The suites MUST cover at minimum:
- action validation and state-matrix fail-fast behavior,
- `operation_id` idempotent dedup and replay stability,
- manual retry budget enforcement (`max_manual_retry_per_task`),
- canonical reason taxonomy coverage (`scheduler.manual_cancel`, `scheduler.manual_retry`),
- memory/file backend parity and Run/Stream semantic equivalence.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** task-board manual-control suites run as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent task-board manual-control suites run as required blocking checks

#### Scenario: Regression introduces non-canonical manual-control reason
- **WHEN** contract suites detect manual-control reason drift outside canonical scheduler namespace without synchronized contract update
- **THEN** shared quality gate fails and blocks merge

### Requirement: Quality gate SHALL include runtime readiness contract suites
Quality gate MUST execute runtime readiness contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover at minimum:
- readiness status classification (`ready|degraded|blocked`),
- strict policy escalation (`degraded -> blocked` when strict enabled),
- canonical finding schema and code stability,
- diagnostics additive readiness fields and replay idempotency,
- composer readiness passthrough parity with runtime readiness.

#### Scenario: Contributor runs quality gate in shell
- **WHEN** contributor executes `scripts/check-quality-gate.sh`
- **THEN** readiness contract suites run as required blocking checks

#### Scenario: Contributor runs quality gate in PowerShell
- **WHEN** contributor executes `scripts/check-quality-gate.ps1`
- **THEN** equivalent readiness contract suites run as required blocking checks

#### Scenario: Regression breaks readiness code taxonomy or strict escalation
- **WHEN** readiness contract suite detects non-canonical finding code or strict escalation mismatch
- **THEN** quality gate fails and blocks merge

### Requirement: Shared multi-agent gate SHALL include cross-domain timeout-resolution contract suites
Shared multi-agent gate MUST execute cross-domain timeout-resolution suites as blocking checks.

The suites MUST cover at minimum:
- operation-profile selection validation,
- layered precedence resolution (`profile -> domain -> request`),
- parent-child timeout clamp and exhausted-budget reject behavior,
- replay idempotency of timeout-resolution aggregates,
- Run/Stream equivalence and memory/file backend parity.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** cross-domain timeout-resolution suites run as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent cross-domain timeout-resolution suites run as required blocking checks

#### Scenario: Regression introduces precedence or clamp drift
- **WHEN** contract suites detect divergence in layered timeout precedence or parent-budget convergence semantics
- **THEN** shared gate fails fast and blocks merge

### Requirement: Quality gate SHALL preserve docs-consistency traceability for operation-profile timeout fields
Repository quality gate MUST ensure docs/config/spec alignment for newly introduced operation-profile timeout fields and diagnostics mappings.

#### Scenario: Config field introduced without docs mapping
- **WHEN** operation-profile timeout field exists in runtime config but docs mapping is missing or stale
- **THEN** docs consistency validation fails and quality gate returns non-zero status

#### Scenario: Docs and contract index are synchronized
- **WHEN** operation-profile timeout fields, diagnostics keys, and contract index mappings are aligned
- **THEN** quality gate proceeds without docs-consistency failure

### Requirement: Quality gate SHALL include diagnostics-query performance regression checks
The standard repository quality gate MUST execute diagnostics-query benchmark regression checks as blocking validation.

This check MUST run in both shell and PowerShell quality-gate scripts to preserve cross-platform parity.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** diagnostics-query performance regression check is executed as a required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent diagnostics-query performance regression check is executed as a required blocking step

### Requirement: Quality gate SHALL block merge on diagnostics-query threshold regression
When diagnostics-query benchmark regression check reports degradation beyond configured thresholds, quality gate MUST fail and block merge.

#### Scenario: Diagnostics-query regression exceeds configured threshold
- **WHEN** one or more diagnostics-query benchmark metrics exceed configured degradation limits
- **THEN** quality gate exits non-zero and validation is blocked

#### Scenario: Diagnostics-query regression remains within configured threshold
- **WHEN** all diagnostics-query benchmark metrics remain within configured limits
- **THEN** quality gate proceeds without diagnostics-query performance failure

### Requirement: Quality gate SHALL include adapter-health contract suites
The standard quality gate MUST execute adapter-health contract suites as blocking validation in both shell and PowerShell paths.

The suites MUST cover:
- adapter-health configuration validation,
- readiness mapping strict/non-strict behavior,
- diagnostics additive schema and replay idempotency,
- adapter conformance health matrix.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter-health contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter-health contract suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on adapter-health semantic drift
When adapter-health suites detect readiness mapping drift, non-canonical reason taxonomy, or replay-idempotency regression, quality gate MUST fail and block merge.

#### Scenario: Adapter-health mapping drifts from contract
- **WHEN** contract suites detect divergence in required/optional mapping or strict escalation behavior
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Adapter-health semantics remain within contract
- **WHEN** adapter-health suites pass all canonical semantic assertions
- **THEN** quality gate proceeds without adapter-health failure

### Requirement: Quality gate SHALL include readiness-admission contract suites
Quality gate MUST execute readiness-admission contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- admission config validation fail-fast and rollback behavior,
- blocked/degraded policy mapping semantics,
- deny-path side-effect-free assertions,
- Run/Stream admission equivalence,
- diagnostics additive schema and replay idempotency.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** readiness-admission contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent readiness-admission contract suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on readiness-admission semantic drift
When readiness-admission suites detect mapping drift, non-canonical admission reason taxonomy, or deny-path side-effect regressions, quality gate MUST fail and block merge.

#### Scenario: Admission deny path mutates scheduler state
- **WHEN** contract suites detect task lifecycle mutation after admission deny
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Readiness-admission semantics remain aligned
- **WHEN** readiness-admission suites pass canonical semantic assertions
- **THEN** quality gate proceeds without readiness-admission failure

### Requirement: Quality gate SHALL include diagnostics-cardinality contract suites
The standard quality gate MUST execute diagnostics-cardinality contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover at minimum:
- diagnostics cardinality config validation fail-fast and hot-reload rollback behavior,
- overflow policy semantics (`truncate_and_record` and `fail_fast`),
- deterministic truncation output semantics,
- Run/Stream truncation equivalence,
- replay-idempotent cardinality aggregate behavior.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** diagnostics-cardinality contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent diagnostics-cardinality contract suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on diagnostics-cardinality semantic drift
When diagnostics-cardinality suites detect non-deterministic truncation output, non-canonical overflow policy behavior, or replay-idempotency regressions, quality gate MUST fail and block merge.

#### Scenario: Regression changes truncation output ordering
- **WHEN** contract suites detect equivalent payloads produce different truncated field summaries
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Diagnostics-cardinality semantics remain aligned
- **WHEN** diagnostics-cardinality suites pass canonical semantic assertions
- **THEN** quality gate proceeds without diagnostics-cardinality failure

### Requirement: Quality gate SHALL include adapter-health governance contract suites
Quality gate MUST execute adapter-health governance contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- adapter-health backoff/circuit config validation fail-fast and rollback behavior,
- circuit transition determinism and half-open budget semantics,
- readiness strict/non-strict mapping stability under governance paths,
- diagnostics additive schema stability and replay idempotency,
- adapter conformance governance matrix parity.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter-health governance suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter-health governance suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on adapter-health governance semantic drift
When governance suites detect transition drift, canonical reason-code drift, or replay-idempotency regressions, quality gate MUST fail and block merge.

#### Scenario: Regression alters half-open transition semantics
- **WHEN** governance suites detect `half_open` no longer reopens on failed probe
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Governance semantics remain aligned
- **WHEN** governance suites pass canonical assertions
- **THEN** quality gate proceeds without adapter-health-governance failures

### Requirement: Quality gate SHALL include readiness-timeout-health replay fixture suites
Quality gate MUST execute readiness-timeout-health composite replay fixture suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- composite fixture matrix coverage,
- canonical taxonomy drift detection,
- Run/Stream parity assertions,
- replay idempotency assertions.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** A47 composite replay fixture suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent A47 composite replay fixture suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on composite semantic drift
When composite replay fixture suites detect canonical semantic drift across readiness, timeout-resolution, or adapter-health domains, quality gate MUST fail fast and block merge.

#### Scenario: Composite fixture detects timeout-source drift
- **WHEN** fixture assertion detects non-canonical timeout-resolution source mapping
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Composite fixture semantics remain aligned
- **WHEN** composite replay fixture assertions pass canonical semantic checks
- **THEN** quality gate proceeds without A47 replay fixture failure

### Requirement: Quality gate SHALL include cross-domain primary-reason arbitration contract suites
Quality gate MUST execute cross-domain primary-reason arbitration suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- precedence-order assertions,
- tie-break determinism assertions,
- Run/Stream parity assertions,
- replay idempotency assertions,
- taxonomy drift assertions.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** arbitration contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent arbitration contract suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on primary-reason arbitration semantic drift
When arbitration suites detect precedence drift, tie-break drift, or canonical taxonomy drift, quality gate MUST fail fast and block merge.

#### Scenario: Drift changes top-level timeout precedence
- **WHEN** arbitration suite detects timeout reject no longer outranks blocked readiness
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Arbitration semantics remain aligned
- **WHEN** arbitration suites pass canonical semantic assertions
- **THEN** quality gate proceeds without arbitration-related failure

### Requirement: Quality gate SHALL include arbitration explainability contract suites
Quality gate MUST execute arbitration explainability contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- secondary reason boundedness and deterministic ordering,
- remediation hint taxonomy stability,
- rule-version stability,
- Run/Stream explainability parity,
- replay idempotency for explainability aggregates.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** arbitration explainability suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent arbitration explainability suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on explainability semantic drift
When explainability suites detect secondary ordering drift, hint taxonomy drift, or rule-version drift, quality gate MUST fail fast and block merge.

#### Scenario: Secondary ordering drifts from canonical rule
- **WHEN** explainability suite detects non-deterministic secondary ordering for equivalent input
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Explainability semantics remain aligned
- **WHEN** explainability suites pass canonical assertions
- **THEN** quality gate proceeds without explainability-related failure

