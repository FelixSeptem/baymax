## ADDED Requirements

### Requirement: Repository SHALL define versioning and compatibility policy
The repository MUST provide a versioning and compatibility policy that defines semantic versioning rules, supported Go version window, provider support levels, and breaking-change communication requirements.

#### Scenario: Contributor prepares a release
- **WHEN** a maintainer prepares a new release
- **THEN** the maintainer can follow documented semantic versioning and breaking-change rules without relying on tribal knowledge

#### Scenario: User evaluates upgrade risk
- **WHEN** a user reads project documentation before upgrading
- **THEN** the user can determine compatibility scope and expected migration risk from a single documented policy source

### Requirement: Repository SHALL provide security disclosure process
The repository MUST include a security disclosure document that uses GitHub Security Advisory as the primary private reporting channel and defines response timeline and disclosure flow.

#### Scenario: Security researcher reports a vulnerability
- **WHEN** a reporter finds a potential vulnerability
- **THEN** the reporter is directed to GitHub Security Advisory rather than public issue channels

#### Scenario: Maintainer handles a reported vulnerability
- **WHEN** a vulnerability report is received
- **THEN** maintainers can execute a documented response workflow including triage, fix, and coordinated disclosure

### Requirement: Repository SHALL provide contribution and review workflow baseline
The repository MUST include contribution guidance and standardized GitHub templates for issue reporting and pull request submission, including a minimum review checklist for tests, docs synchronization, compatibility impact, and breaking-change marking.

#### Scenario: External contributor submits first pull request
- **WHEN** a new contributor opens a pull request
- **THEN** pull request template prompts required quality and compatibility checks before review

#### Scenario: Maintainer triages incoming issue
- **WHEN** a user opens a bug or feature request
- **THEN** issue templates collect minimum structured information for reproducible triage

### Requirement: Repository SHALL publish a community conduct guideline
The repository MUST include a code-of-conduct document and reference it from contributor-facing documentation.

#### Scenario: Community member reviews participation expectations
- **WHEN** a contributor checks collaboration norms
- **THEN** the contributor can find behavior expectations and escalation path in a version-controlled guideline
