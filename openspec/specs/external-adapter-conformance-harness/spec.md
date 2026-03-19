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

