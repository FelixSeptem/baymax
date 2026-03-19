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

