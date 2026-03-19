## ADDED Requirements

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
