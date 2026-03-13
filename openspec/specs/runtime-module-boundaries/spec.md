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
Global runtime packages MUST NOT depend on MCP transport packages, and MCP packages MUST consume runtime configuration and diagnostics APIs via stable interfaces. MCP shared reliability internals MUST be restricted to `mcp/internal/*` and MUST NOT be imported by non-MCP packages.

#### Scenario: Build-time dependency check
- **WHEN** static dependency checks run in CI
- **THEN** no import cycle or reverse dependency from global runtime package to MCP package is allowed, and no non-MCP package imports `mcp/internal/*`

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
The repository MUST include a maintained architecture page with module responsibilities, ownership hints, extension constraints, and internal/shared versus transport-specific MCP layering guidance.

#### Scenario: Contributor plans new runtime feature
- **WHEN** contributor proposes a new runtime capability
- **THEN** contributor can determine target module and boundary constraints from documentation without ad hoc clarification

### Requirement: MCP architecture docs SHALL describe shared-core layering
Documentation MUST explicitly describe MCP shared-core and transport-specific layering, including examples of which logic belongs in shared internal modules versus `mcp/http` or `mcp/stdio` packages.

#### Scenario: Contributor modifies MCP retry behavior
- **WHEN** contributor changes retry behavior for MCP clients
- **THEN** docs indicate that semantic retry logic belongs to shared internal MCP core and transport-specific code only provides protocol hooks

### Requirement: Boundary reviews SHALL verify context-model responsibility split
Boundary governance reviews MUST verify that `context/*` packages orchestrate policy only, while provider SDK protocol actions remain in `model/*` packages and are consumed via interfaces.

#### Scenario: Token counting path is reviewed
- **WHEN** reviewer inspects context assembly token-count flow
- **THEN** context layer invokes model-facing interfaces and does not import provider SDK packages directly

### Requirement: Boundary reviews SHALL validate dependency and semantic direction together
Boundary checks MUST include both import-direction validation and semantic responsibility validation for cross-module orchestration paths.

#### Scenario: Static dependency check passes but semantic ownership drifts
- **WHEN** review detects behavior implemented in the wrong module despite legal imports
- **THEN** the change is not accepted until ownership is moved back to the designated module

### Requirement: Boundary governance outcomes SHALL be reflected in architecture docs
When module responsibility corrections are made, architecture and boundary documentation MUST be updated in the same change to preserve a single source of truth.

#### Scenario: Runtime boundary fix is merged
- **WHEN** a change modifies cross-module responsibilities
- **THEN** `docs/runtime-module-boundaries.md` and related docs are updated before completion

