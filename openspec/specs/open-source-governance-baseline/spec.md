# open-source-governance-baseline Specification

## Purpose
TBD - created by archiving change establish-open-source-p0-governance-baseline. Update Purpose after archive.
## Requirements
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

### Requirement: Repository SHALL publish a community conduct guideline
The repository MUST include a code-of-conduct document and reference it from contributor-facing documentation.

#### Scenario: Community member reviews participation expectations
- **WHEN** a contributor checks collaboration norms
- **THEN** the contributor can find behavior expectations and escalation path in a version-controlled guideline

### Requirement: Governance documentation SHALL avoid implicit stable-release claims during pre-1 phase
When repository release stage remains `0.x`, contributor-facing governance documents MUST NOT imply `1.0`, `stable`, or `prod-ready` commitments unless a dedicated release-governance change explicitly upgrades the stage.

This requirement applies to roadmap and versioning policy as source-of-truth entries.

#### Scenario: Maintainer edits roadmap milestones
- **WHEN** roadmap milestone text is updated while repository is still in `0.x`
- **THEN** roadmap language remains compatible with pre-1 compatibility posture and does not imply stable-release guarantees

#### Scenario: Maintainer proposes explicit stage upgrade
- **WHEN** maintainers decide to move from `0.x` to explicit stable-release commitment
- **THEN** governance change is required before source-of-truth docs can claim `1.0/prod-ready` posture

### Requirement: Pre-1 governance SHALL define long-term direction as non-execution scope
Governance documentation in pre-1 phase MUST keep long-term platformization directions as non-execution scope and MUST not place them in near-term active milestones.

#### Scenario: Roadmap includes long-term platformization themes
- **WHEN** maintainer documents platformization themes such as control plane or multi-tenant governance
- **THEN** those themes are marked as long-term directions and excluded from near-term execution scope

