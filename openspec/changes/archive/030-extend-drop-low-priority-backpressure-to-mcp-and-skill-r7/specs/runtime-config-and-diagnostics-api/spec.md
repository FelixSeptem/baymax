## ADDED Requirements

### Requirement: Diagnostics SHALL expose drop-low-priority counts by dispatch phase
Runtime diagnostics MUST expose low-priority drop counts with source buckets for `local`, `mcp`, and `skill`, while preserving existing aggregate drop count semantics for compatibility.

#### Scenario: Mixed drops across multiple dispatch paths
- **WHEN** low-priority drops occur in local, mcp, and skill within recent runs
- **THEN** diagnostics include per-phase bucket counts and an aggregate count consistent with bucket totals

#### Scenario: Existing diagnostics consumer reads aggregate only
- **WHEN** a consumer reads only existing aggregate drop count fields
- **THEN** diagnostics remain backward-compatible and do not require consumer changes

### Requirement: Drop-low-priority configuration semantics SHALL remain unified across dispatch paths
The runtime configuration contract for drop-low-priority MUST use one shared rule model across local, mcp, and skill paths.

#### Scenario: Rule is configured by tool and keyword
- **WHEN** `priority_by_tool` and `priority_by_keyword` are configured
- **THEN** the same rules are applied regardless of whether call is local, mcp, or skill
