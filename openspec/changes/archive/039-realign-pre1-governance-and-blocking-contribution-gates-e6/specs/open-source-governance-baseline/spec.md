## MODIFIED Requirements

### Requirement: Repository SHALL define versioning and compatibility policy
The repository MUST publish a versioning policy using Semantic Versioning notation and MUST explicitly declare that, before `1.0.0`, the project does not provide API/runtime/config compatibility guarantees.

The policy MUST also document that maintenance support is limited to the latest minor line.

#### Scenario: Maintainer prepares a pre-1.x release
- **WHEN** a maintainer prepares a release with major version `0`
- **THEN** release documentation states version semantics and explicitly states no compatibility commitment before `1.0.0`

#### Scenario: User evaluates upgrade expectation in pre-1.x phase
- **WHEN** a user checks repository governance documentation before upgrading
- **THEN** the user can identify that compatibility is best-effort and support scope is the latest minor line only

### Requirement: Repository SHALL provide security disclosure process
The repository MUST include a security disclosure document that uses a maintainer email address as the private reporting channel and defines a best-effort triage/fix/disclosure workflow without response-time SLA commitments.

#### Scenario: Security researcher reports a vulnerability
- **WHEN** a reporter finds a potential vulnerability
- **THEN** the reporter is directed to the documented security email channel instead of public issue channels

#### Scenario: Maintainer handles a reported vulnerability
- **WHEN** a vulnerability report is received via email
- **THEN** maintainers can execute a documented best-effort workflow including triage, fix validation, and disclosure decision

### Requirement: Repository SHALL provide contribution and review workflow baseline
The repository MUST include contribution guidance and standardized GitHub templates for issue reporting and pull request submission with Chinese-first prompts and English-acceptable inputs.

Templates MUST define required checklist and context fields for tests, docs synchronization, and change impact disclosure.

#### Scenario: External contributor submits first pull request
- **WHEN** a new contributor opens a pull request
- **THEN** the template presents required structured fields and checklist items before review can proceed

#### Scenario: Maintainer triages incoming issue
- **WHEN** a user opens a bug or feature request
- **THEN** the template captures minimum structured information for reproducible triage in Chinese-first form while accepting English content
