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

