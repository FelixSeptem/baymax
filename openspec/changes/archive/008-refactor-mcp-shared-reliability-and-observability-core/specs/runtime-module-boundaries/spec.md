## MODIFIED Requirements

### Requirement: Runtime modules SHALL enforce one-way dependency direction
Global runtime packages MUST NOT depend on MCP transport packages, and MCP packages MUST consume runtime configuration and diagnostics APIs via stable interfaces. MCP shared reliability internals MUST be restricted to `mcp/internal/*` and MUST NOT be imported by non-MCP packages.

#### Scenario: Build-time dependency check
- **WHEN** static dependency checks run in CI
- **THEN** no import cycle or reverse dependency from global runtime package to MCP package is allowed, and no non-MCP package imports `mcp/internal/*`

### Requirement: Documentation SHALL include architecture and ownership reference
The repository MUST include a maintained architecture page with module responsibilities, ownership hints, extension constraints, and internal/shared versus transport-specific MCP layering guidance.

#### Scenario: Contributor plans new runtime feature
- **WHEN** contributor proposes a new runtime capability
- **THEN** contributor can determine target module and boundary constraints from documentation without ad hoc clarification

## ADDED Requirements

### Requirement: MCP architecture docs SHALL describe shared-core layering
Documentation MUST explicitly describe MCP shared-core and transport-specific layering, including examples of which logic belongs in shared internal modules versus `mcp/http` or `mcp/stdio` packages.

#### Scenario: Contributor modifies MCP retry behavior
- **WHEN** contributor changes retry behavior for MCP clients
- **THEN** docs indicate that semantic retry logic belongs to shared internal MCP core and transport-specific code only provides protocol hooks