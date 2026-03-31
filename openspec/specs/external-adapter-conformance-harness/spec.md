# external-adapter-conformance-harness Specification

## Purpose
TBD - created by archiving change introduce-external-adapter-conformance-harness-and-gate-a22. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL provide adapter conformance harness for MCP, Model, and Tool paths
The repository MUST provide a conformance harness that validates external adapter behavior across minimal MCP, Model, and Tool integration matrices.

The minimum implementation priority MUST be `MCP > Model > Tool`, while all three categories MUST have at least one conformance scenario in initial rollout.

#### Scenario: Contributor runs conformance harness
- **WHEN** contributor executes adapter conformance command
- **THEN** harness evaluates MCP, Model, and Tool minimum conformance scenarios in one consistent flow

#### Scenario: Maintainer audits category coverage
- **WHEN** maintainer reviews conformance suite entries
- **THEN** MCP category appears as primary coverage tier and Model/Tool minimum scenarios are present

### Requirement: Adapter conformance harness SHALL run in offline deterministic mode by default
Conformance execution MUST use offline fixtures, stubs, or fakes by default and MUST NOT require network connectivity or external provider credentials.

#### Scenario: CI runs without external credentials
- **WHEN** CI executes adapter conformance suite in clean environment
- **THEN** suite runs successfully without external API credentials

#### Scenario: Local run in disconnected environment
- **WHEN** contributor runs conformance suite without network access
- **THEN** suite remains executable and deterministic

### Requirement: Harness SHALL validate semantic contract boundaries for adapter integrations
The conformance harness MUST validate adapter semantic boundaries, including:
- Run/Stream semantic equivalence where applicable,
- normalized error layer and reason taxonomy mappings,
- downgrade behavior for unsupported optional capabilities,
- fail-fast behavior for invalid mandatory input.

#### Scenario: Model adapter lacks optional capability
- **WHEN** adapter does not support an optional capability such as token counting
- **THEN** harness verifies deterministic downgrade behavior without semantic drift

#### Scenario: Adapter returns malformed mandatory response
- **WHEN** adapter provides invalid mandatory contract fields
- **THEN** harness asserts fail-fast outcome with normalized error classification

### Requirement: A21 adapter templates SHALL be verifiable by conformance harness
Adapter templates introduced by A21 MUST map to conformance harness cases so template guidance stays executable and contract-aligned.

#### Scenario: Template path is updated in docs
- **WHEN** maintainer updates adapter template snippets in A21 docs
- **THEN** corresponding conformance cases remain present and pass for the documented template path

#### Scenario: Template drifts from contract expectations
- **WHEN** template behavior diverges from conformance expectations
- **THEN** conformance suite fails and signals template-contract drift

### Requirement: Adapter conformance harness SHALL validate manifest-profile alignment
Adapter conformance harness MUST validate that adapter manifest declarations and executed conformance profile are semantically aligned.

This validation MUST include:
- declared adapter category vs executed category suite,
- declared required capabilities vs executed required contract assertions,
- declared optional capabilities vs downgrade-path assertions where applicable.

#### Scenario: Declared category mismatches conformance suite
- **WHEN** harness detects manifest category differs from executed conformance category
- **THEN** conformance run fails with manifest-profile mismatch classification

#### Scenario: Required capability declaration is not covered by contract assertions
- **WHEN** harness detects required capability declaration without corresponding contract assertion path
- **THEN** conformance run fails and reports missing required-capability coverage

### Requirement: Adapter conformance harness SHALL include capability negotiation matrix
Adapter conformance harness MUST include negotiation matrix coverage for:
- required capability missing fail-fast,
- optional capability downgrade behavior,
- strategy override (`fail_fast` vs `best_effort`) behavior,
- Run/Stream negotiation semantic equivalence.

#### Scenario: Harness executes required-missing matrix
- **WHEN** conformance harness runs required capability missing scenario
- **THEN** harness observes deterministic fail-fast classification with canonical reason taxonomy

#### Scenario: Harness executes optional-downgrade matrix
- **WHEN** conformance harness runs optional capability missing scenario under downgrade-allowed strategy
- **THEN** harness verifies deterministic downgrade behavior and canonical downgrade reason

### Requirement: Conformance harness SHALL validate negotiation-profile alignment with adapter declarations
Conformance harness MUST verify that negotiation test profile aligns with adapter declaration shape and strategy inputs.

#### Scenario: Strategy profile mismatches declared adapter negotiation configuration
- **WHEN** harness detects mismatch between negotiation profile and declared adapter configuration
- **THEN** harness fails with explicit profile-mismatch classification

### Requirement: Adapter conformance harness SHALL include runtime health-probe matrix
Adapter conformance harness MUST include runtime health-probe coverage for required and optional adapter paths.

The matrix MUST include at minimum:
- required adapter unavailable,
- optional adapter unavailable with deterministic downgrade,
- degraded adapter classification visibility.

#### Scenario: Required adapter probe returns unavailable
- **WHEN** conformance harness executes required adapter health case and probe returns unavailable
- **THEN** harness asserts fail-fast classification with canonical adapter-health reason code

#### Scenario: Optional adapter probe returns unavailable
- **WHEN** conformance harness executes optional adapter health case and probe returns unavailable
- **THEN** harness asserts deterministic downgrade path with observable adapter-health finding

### Requirement: Adapter health conformance suites SHALL remain offline deterministic
Adapter health conformance execution MUST run using offline fixtures or fakes and MUST NOT require external network access.

#### Scenario: CI runs adapter-health conformance without network
- **WHEN** CI executes adapter conformance harness in disconnected environment
- **THEN** adapter-health suites run deterministically without external credentials

#### Scenario: Local contributor runs adapter-health suites offline
- **WHEN** contributor runs conformance harness locally without network access
- **THEN** adapter-health assertions remain executable and deterministic

### Requirement: Adapter conformance harness SHALL include health-governance matrix suites
External adapter conformance harness MUST include adapter-health governance matrix suites as blocking validations.

The matrix MUST cover:
- backoff throttling behavior under repeated failure,
- circuit transition determinism (`closed|open|half_open`),
- half-open recovery and reopen behavior,
- strict/non-strict readiness mapping parity for required/optional adapters,
- replay-idempotent governance diagnostics aggregates.

#### Scenario: Harness executes health-governance matrix for one adapter fixture
- **WHEN** conformance harness runs adapter health suites
- **THEN** backoff/circuit/readiness/governance-observability assertions execute as required checks

#### Scenario: Governance semantics drift from canonical matrix
- **WHEN** harness detects state-transition or readiness-classification drift
- **THEN** conformance validation fails and returns non-zero status

### Requirement: Conformance harness SHALL include mainstream sandbox backend matrix suites
External adapter conformance harness MUST include sandbox backend matrix suites for:
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job` (when Windows runner is available)

Platform-unavailable backend suites MAY be skipped only with deterministic skip classification.

#### Scenario: Linux runner executes sandbox backend matrix
- **WHEN** conformance harness runs on Linux environment
- **THEN** harness executes `linux_nsjail`, `linux_bwrap`, and `oci_runtime` suites with deterministic results

#### Scenario: Windows runner executes windows-job suite
- **WHEN** conformance harness runs on Windows environment
- **THEN** harness executes `windows_job` suite with deterministic contract assertions

### Requirement: Harness SHALL validate sandbox capability negotiation and session lifecycle semantics
Sandbox adapter conformance harness MUST validate:
- required capability missing fail-fast behavior,
- optional capability downgrade behavior,
- `per_call|per_session` lifecycle semantics,
- crash/reconnect/close-idempotent semantics for session lifecycle.

#### Scenario: Required capability is missing for selected backend profile
- **WHEN** harness executes adapter with unsatisfied required capability
- **THEN** suite fails with deterministic missing-required-capability classification

#### Scenario: Per-session lifecycle close is repeated
- **WHEN** harness invokes close repeatedly on same sandbox session
- **THEN** suite verifies idempotent close semantics without duplicate terminal side effects

### Requirement: Harness SHALL classify sandbox adapter drift using canonical classes
Sandbox adapter conformance harness MUST emit deterministic drift classes at minimum:
- `sandbox_backend_profile_drift`
- `sandbox_capability_claim_drift`
- `sandbox_session_lifecycle_drift`
- `sandbox_reason_taxonomy_drift`

#### Scenario: Backend profile mapping drifts from canonical fixture
- **WHEN** adapter backend/profile mapping output diverges from fixture expectation
- **THEN** harness fails with deterministic `sandbox_backend_profile_drift` classification

### Requirement: Conformance harness SHALL include mainstream memory adapter matrix suites
External adapter conformance harness MUST include memory adapter matrix suites for:
- `mem0`
- `zep`
- `openviking`
- `generic`

Memory matrix execution MUST be offline deterministic by default and MUST NOT require external network or provider credentials.

#### Scenario: CI executes memory matrix in disconnected environment
- **WHEN** conformance harness runs memory adapter suites without network access
- **THEN** suites execute deterministically using fixtures or fakes and produce stable pass fail semantics

#### Scenario: Contributor declares unsupported memory profile in matrix
- **WHEN** memory conformance matrix input references unsupported profile id
- **THEN** harness fails fast with deterministic profile-unknown classification

### Requirement: Memory conformance suites SHALL validate canonical operation and error taxonomy semantics
Memory adapter conformance suites MUST validate canonical `Query/Upsert/Delete` semantics, required and optional capability behavior, and normalized error taxonomy mapping.

#### Scenario: Adapter misses required memory operation
- **WHEN** adapter manifest declares required memory operation but implementation does not satisfy it
- **THEN** conformance fails with deterministic required-capability-missing classification

#### Scenario: Adapter misses optional memory capability
- **WHEN** adapter lacks optional capability such as TTL or metadata filter
- **THEN** conformance verifies deterministic downgrade path with canonical downgrade reason

### Requirement: Memory conformance SHALL assert fallback and Run Stream semantic equivalence
Memory conformance harness MUST assert:
- fallback policy behavior (`fail_fast|degrade_to_builtin|degrade_without_memory`),
- Run and Stream semantic equivalence under equivalent memory inputs.

#### Scenario: Equivalent Run and Stream memory sequence
- **WHEN** equivalent memory request sequence runs through Run and Stream suites
- **THEN** conformance output remains semantically equivalent for operation outcomes and reason taxonomy

#### Scenario: Fallback policy behavior drifts from canonical expectation
- **WHEN** memory fallback execution diverges from canonical fixture expectation
- **THEN** conformance fails with deterministic fallback-drift classification

