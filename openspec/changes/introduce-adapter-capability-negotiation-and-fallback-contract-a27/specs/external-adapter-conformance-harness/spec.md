## ADDED Requirements

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
