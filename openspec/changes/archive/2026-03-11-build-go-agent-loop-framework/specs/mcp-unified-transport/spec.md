## ADDED Requirements

### Requirement: MCP runtime SHALL expose transport-agnostic interface
The system MUST provide a unified MCP client interface for `stdio` and `SSE/HTTP` transports with equivalent request/response semantics.

#### Scenario: Stdio and HTTP parity
- **WHEN** the same MCP tool is called via stdio and HTTP adapters
- **THEN** both adapters MUST return the same normalized tool result shape

### Requirement: MCP runtime SHALL enforce consistent timeout and retry policy
The system MUST apply the same timeout, retry, and cancellation semantics across both MCP transports.

#### Scenario: Timeout consistency
- **WHEN** a MCP call exceeds configured timeout on either transport
- **THEN** the runtime MUST classify timeout uniformly and emit the same error category

#### Scenario: Retry consistency
- **WHEN** transient MCP failure occurs on either transport
- **THEN** the runtime MUST retry according to shared retry policy and backoff configuration

### Requirement: HTTP transport SHALL support reconnect with continuity guarantees
The HTTP transport MUST support heartbeat monitoring and reconnect with bounded retries while preserving call identity for observability.

#### Scenario: Heartbeat timeout reconnect
- **WHEN** heartbeat is not observed within configured threshold
- **THEN** the adapter MUST reconnect and emit reconnection event before new call dispatch

#### Scenario: Call identity preservation
- **WHEN** reconnect occurs during active run
- **THEN** subsequent MCP events MUST preserve stable call identifiers for correlation

### Requirement: MCP events SHALL be normalized across transports
The runtime MUST emit normalized MCP lifecycle events regardless of underlying transport.

#### Scenario: Successful call event sequence
- **WHEN** a MCP tool call completes successfully
- **THEN** the runtime MUST emit `mcp.requested` then `mcp.completed` in order

#### Scenario: Failed call event sequence
- **WHEN** a MCP tool call fails terminally
- **THEN** the runtime MUST emit `mcp.requested` then `mcp.failed` with error classification
