# external-adapter-template-and-migration-mapping Specification

## Purpose
TBD - created by archiving change introduce-external-adapter-template-and-migration-mapping-a21. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL provide external adapter templates with documented priority order
The repository MUST provide external adapter onboarding templates and MUST publish priority order as:
- MCP adapter template first,
- Model provider adapter template second,
- Tool adapter template third.

Templates MAY be minimal skeletons but MUST be executable as reference snippets for integrators.

#### Scenario: Integrator browses adapter templates
- **WHEN** integrator opens template index
- **THEN** MCP, Model, and Tool templates are available with documented priority and scope

#### Scenario: Integrator applies MCP template first
- **WHEN** integrator follows recommended onboarding path
- **THEN** MCP template guidance appears as first-class starting point

### Requirement: Migration mapping SHALL use capability-domain and code-snippet dual structure
Migration documentation MUST organize guidance by both capability domain and representative code snippets.

Each mapping entry MUST include at least:
- previous pattern,
- recommended pattern,
- compatibility notes.

#### Scenario: Contributor migrates model adapter integration
- **WHEN** contributor checks migration mapping for model capability domain
- **THEN** contributor can find old/new code snippet mapping with explicit compatibility notes

#### Scenario: Contributor migrates MCP adapter integration
- **WHEN** contributor checks migration mapping for MCP capability domain
- **THEN** contributor can apply provided snippet mapping without source-code archaeology

### Requirement: Migration guidance SHALL include common errors and alternative patterns
Migration documentation MUST include common integration mistakes and corresponding replacement patterns for each adapter category.

#### Scenario: Integrator hits common adapter registration mistake
- **WHEN** integrator encounters documented anti-pattern during migration
- **THEN** documentation provides explicit replacement path and corrected snippet

#### Scenario: Integrator validates fallback behavior mapping
- **WHEN** integrator checks error handling section
- **THEN** documentation shows fail-fast and fallback alternatives with selection criteria

### Requirement: Compatibility semantics SHALL be stated uniformly for adapter migration
Adapter migration docs MUST describe compatibility semantics consistently using `additive + nullable + default + fail-fast` boundary terms.

#### Scenario: Integrator reviews diagnostics field migration
- **WHEN** integrator reads compatibility section for new optional fields
- **THEN** integrator can determine additive/nullable/default handling and fail-fast boundaries deterministically

### Requirement: Repository SHALL provide sandbox adapter onboarding templates for mainstream backends
Adapter template index MUST provide onboarding templates for mainstream sandbox backends:
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job`

Each template MUST include profile declaration snippet, manifest snippet, and conformance command reference.

#### Scenario: Integrator opens sandbox adapter template index
- **WHEN** integrator reads adapter template documentation
- **THEN** mainstream sandbox backend templates are discoverable with executable onboarding snippets

#### Scenario: Integrator follows template for OCI backend
- **WHEN** integrator uses `oci_runtime` onboarding template
- **THEN** template includes complete profile/manifest/conformance snippet chain

### Requirement: Migration mapping SHALL cover legacy wrapper to profile-pack adapter transitions
Migration documentation MUST provide deterministic mapping for legacy sandbox wrapper integrations to canonical profile-pack adapters.

Each mapping entry MUST include:
- previous wrapper pattern,
- target profile-pack pattern,
- compatibility and rollback notes.

#### Scenario: Contributor migrates legacy process wrapper
- **WHEN** contributor follows migration mapping from wrapper-based sandbox integration
- **THEN** resulting configuration aligns with canonical profile-pack adapter contract semantics

#### Scenario: Migration mapping misses rollback guidance
- **WHEN** mapping entry omits rollback notes
- **THEN** documentation contract validation fails

### Requirement: Template and migration docs SHALL remain conformance-linked
Each sandbox adapter template and migration mapping entry MUST link to at least one conformance suite identifier so documentation stays executable and verifiable.

#### Scenario: Maintainer updates sandbox template snippet
- **WHEN** maintainer changes template example
- **THEN** corresponding conformance suite link remains present and valid

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

