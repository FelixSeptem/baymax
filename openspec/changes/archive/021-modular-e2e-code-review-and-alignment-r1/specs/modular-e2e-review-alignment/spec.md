## ADDED Requirements

### Requirement: Review process SHALL cover both module boundaries and end-to-end chains
The repository MUST define a review matrix that simultaneously covers module-level responsibilities and end-to-end mainline execution chains before a governance refactor is considered complete.

#### Scenario: Module review is executed
- **WHEN** a governance review change starts
- **THEN** reviewers use an explicit module checklist covering at least `core`, `context`, `model`, `runtime`, and `observability`

#### Scenario: Chain review is executed
- **WHEN** the same review change validates behavior consistency
- **THEN** reviewers use an explicit chain checklist covering at least `Run`, `Stream`, `tool-loop`, `CA2 stage2 retrieval`, and `CA3 pressure/recovery`

### Requirement: Governance review SHALL close all identified severities in one change
For review-governance scoped changes, all identified findings labeled P0, P1, and P2 MUST be resolved in the same change before archive.

#### Scenario: Findings list includes mixed severities
- **WHEN** the change reaches implementation and verification phase
- **THEN** no open P0, P1, or P2 finding remains in the review checklist

### Requirement: Mainline contract tests SHALL be mandatory completion criteria
The system MUST include contract tests for all mainline flows and treat them as required quality-gate checks for review-governance scoped changes.

#### Scenario: Mainline contract suite runs
- **WHEN** CI or local quality validation executes
- **THEN** every defined mainline flow has contract coverage and failures block completion

### Requirement: Repository hygiene SHALL reject temporary backup artifacts
The repository MUST prevent temporary backup artifacts from being merged, including random-suffix copies such as `*.go.<random>`.

#### Scenario: Hygiene check detects backup artifact
- **WHEN** validation scans tracked files and detects temporary backup patterns
- **THEN** the quality gate fails and requires cleanup before merge

### Requirement: Documentation alignment SHALL be required for governance refactors
Any governance review change MUST update README and all affected docs pages so documented behavior matches implemented behavior at merge time.

#### Scenario: Code behavior changed in governance review
- **WHEN** the change updates behavior, boundaries, or quality-gate semantics
- **THEN** README and affected `docs/*` pages are updated in the same change
