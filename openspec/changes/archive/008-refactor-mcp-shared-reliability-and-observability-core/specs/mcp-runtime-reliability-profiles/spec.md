## MODIFIED Requirements

### Requirement: MCP transports SHALL share common retry and reconnect semantics
`mcp/http` and `mcp/stdio` MUST execute retry, backoff, reconnect, timeout handling, and fail-fast stop conditions through a shared internal core, so semantic outcomes remain aligned across transports.

#### Scenario: Retryable transient failure occurs
- **WHEN** MCP call fails with retryable transient error
- **THEN** both transports follow the same shared retry policy and emit consistent retry diagnostics

#### Scenario: Non-retryable failure occurs
- **WHEN** MCP call fails with non-retryable condition
- **THEN** both transports stop retry immediately and return aligned error classification

### Requirement: MCP runtime SHALL expose normalized diagnostic summary
The runtime MUST expose a diagnostic summary for recent MCP calls, including latency, retry count, reconnect count, error class, and active profile, through library API endpoints that are consistent with runtime diagnostics APIs. Summary records MUST be produced via shared internal mapping to avoid transport-specific drift.

#### Scenario: Operator requests recent summary
- **WHEN** diagnostic summary for latest N MCP calls is requested through library API
- **THEN** output contains normalized fields that can be compared across `http` and `stdio`

## ADDED Requirements

### Requirement: MCP runtime refactor SHALL satisfy duplicate-logic reduction threshold
The MCP reliability refactor MUST document baseline duplicate-logic metrics for `mcp/http` and `mcp/stdio`, and MUST achieve an agreed relative reduction threshold during acceptance.

#### Scenario: Duplicate-logic threshold check
- **WHEN** refactor acceptance checks run
- **THEN** reported duplicate-logic reduction percentage meets or exceeds the documented threshold