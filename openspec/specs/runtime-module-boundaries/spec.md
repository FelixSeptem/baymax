# runtime-module-boundaries Specification

## Purpose
TBD - created by archiving change refactor-runtime-responsibility-boundaries-and-enrich-docs. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL define explicit module boundaries
The system MUST define explicit boundaries between global runtime platform capabilities and MCP-specific capabilities, with documented ownership and allowed dependency directions.

#### Scenario: Developer evaluates module responsibility
- **WHEN** developer checks runtime architecture documentation
- **THEN** documentation clearly identifies which package owns global config lifecycle versus MCP policy semantics

### Requirement: Runtime modules SHALL enforce one-way dependency direction
Global runtime packages MUST NOT depend on MCP transport packages, and MCP packages MUST consume runtime configuration and diagnostics APIs via stable interfaces.

#### Scenario: Build-time dependency check
- **WHEN** static dependency checks run in CI
- **THEN** no import cycle or reverse dependency from global runtime package to MCP package is allowed

### Requirement: Runtime refactor SHALL provide migration compatibility guidance
The system MUST provide migration mapping for package moves, deprecation notes, and replacement API examples.

#### Scenario: User upgrades from previous structure
- **WHEN** user follows migration guide from old MCP 单体 runtime 包结构
- **THEN** user can locate equivalent function-scoped package APIs and complete migration without behavior ambiguity

### Requirement: Runtime diagnostics API SHALL be owned by global runtime layer
The system MUST place diagnostics API ownership in the global runtime layer, while MCP layer keeps only MCP-specific diagnostic field semantics.

#### Scenario: Multi-subsystem diagnostics access
- **WHEN** runner, tool, skill, MCP, or observability components request runtime diagnostics
- **THEN** they use a shared diagnostics API surface without importing MCP transport packages

### Requirement: Documentation SHALL include architecture and ownership reference
The repository MUST include a maintained architecture page with module responsibilities, ownership hints, and extension constraints.

#### Scenario: Contributor plans new runtime feature
- **WHEN** contributor proposes a new runtime capability
- **THEN** contributor can determine target module and boundary constraints from documentation without ad hoc clarification

