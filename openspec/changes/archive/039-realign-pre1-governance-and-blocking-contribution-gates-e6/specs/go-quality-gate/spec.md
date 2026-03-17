## ADDED Requirements

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
