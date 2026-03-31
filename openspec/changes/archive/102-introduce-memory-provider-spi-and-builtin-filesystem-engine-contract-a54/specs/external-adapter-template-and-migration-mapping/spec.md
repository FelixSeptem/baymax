## ADDED Requirements

### Requirement: Repository SHALL provide memory adapter onboarding templates for mainstream profiles
Repository MUST provide executable onboarding templates for memory adapters covering:
- `mem0`
- `zep`
- `openviking`
- `generic`

Templates MUST also include builtin filesystem mode enablement example and dual-mode switch example.

#### Scenario: Integrator selects zep onboarding template
- **WHEN** integrator opens memory adapter template index for `zep`
- **THEN** integrator can apply executable template with canonical memory SPI fields and profile wiring

#### Scenario: Integrator enables builtin filesystem mode from template
- **WHEN** integrator follows builtin template path
- **THEN** runtime config example includes deterministic `builtin_filesystem` mode controls and fallback semantics

### Requirement: Migration mapping SHALL cover existing file-based memory path to unified memory SPI
Migration documentation MUST provide mapping from existing file-based memory path to:
- builtin filesystem engine contract path,
- external SPI adapter path.

Each mapping entry MUST include previous pattern, target pattern, compatibility notes, and rollback guidance.

#### Scenario: Contributor migrates legacy file-based memory usage
- **WHEN** contributor follows migration mapping from legacy file path to builtin engine contract
- **THEN** contributor can complete migration without changing external behavior contract unexpectedly

#### Scenario: Contributor migrates to external SPI profile
- **WHEN** contributor follows migration mapping to `external_spi` profile path
- **THEN** required config, manifest, and conformance steps are explicitly documented

### Requirement: Memory templates and migration mapping SHALL bind to observability and conformance checklist
Each memory onboarding template and migration mapping entry MUST include:
- required diagnostics fields checklist,
- required readiness finding checklist,
- linked conformance case identifiers.

#### Scenario: Template update omits conformance case mapping
- **WHEN** maintainer updates memory template without conformance case id reference
- **THEN** docs consistency validation fails with deterministic template-contract drift classification

#### Scenario: Integrator validates observability checklist
- **WHEN** integrator reviews template acceptance criteria
- **THEN** integrator can verify memory observability and readiness coverage before merge
