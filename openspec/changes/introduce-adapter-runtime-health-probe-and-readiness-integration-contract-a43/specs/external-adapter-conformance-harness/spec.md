## ADDED Requirements

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
