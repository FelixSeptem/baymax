## ADDED Requirements

### Requirement: Repository SHALL define enforceable pull request template contract
The repository MUST define a machine-checkable pull request template contract that includes required sections for summary, change details, validation evidence, documentation impact, and change-impact declaration.

The contract MUST be language-agnostic for content values while keeping Chinese-first section labels.

#### Scenario: Pull request body uses English content
- **WHEN** a contributor fills required sections in English
- **THEN** contract validation passes as long as required structure and checklist semantics are preserved

#### Scenario: Pull request removes required section headers
- **WHEN** a contributor submits a pull request body without required section headers
- **THEN** contract validation fails with explicit missing-section diagnostics

### Requirement: Repository SHALL define enforceable issue intake template contract
The repository MUST provide machine-checkable issue intake templates for bug and feature submissions with required problem context, reproduction-or-use-case details, and environment metadata where applicable.

#### Scenario: Bug report misses environment metadata
- **WHEN** a bug issue omits required environment fields
- **THEN** issue template contract validation marks the submission as incomplete

#### Scenario: Feature request lacks problem statement
- **WHEN** a feature issue omits required problem context
- **THEN** issue template contract validation marks the submission as incomplete

### Requirement: Template enforcement tooling SHALL provide deterministic failure reasons
Contribution-template enforcement tooling MUST return deterministic, human-readable failure reasons with machine-readable reason codes for each missing or malformed required item.

#### Scenario: Multiple required items are missing
- **WHEN** enforcement tooling evaluates content missing multiple required items
- **THEN** output lists all violations in a deterministic order with stable reason codes

#### Scenario: Template contract is fully satisfied
- **WHEN** enforcement tooling evaluates compliant content
- **THEN** output indicates success with zero violations
