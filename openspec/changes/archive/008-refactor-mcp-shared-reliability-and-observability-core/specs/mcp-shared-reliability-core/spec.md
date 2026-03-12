## ADDED Requirements

### Requirement: MCP shared core SHALL provide internal unified reliability execution
The MCP domain MUST provide an internal shared execution core for timeout wrapping, retry/backoff flow, fail-fast stop conditions, event emission templates, and diagnostics mapping. This core MUST be consumable by both `mcp/http` and `mcp/stdio` implementations.

#### Scenario: Retryable transient failure in HTTP transport
- **WHEN** HTTP MCP call fails with retryable transient error
- **THEN** call execution follows shared retry/backoff logic and emits normalized retry diagnostics via shared core

#### Scenario: Retryable transient failure in STDIO transport
- **WHEN** STDIO MCP call fails with retryable transient error
- **THEN** call execution follows the same shared retry/backoff logic and emits normalized retry diagnostics via shared core

### Requirement: MCP shared core SHALL remain internal-only
The shared reliability core MUST be placed under `mcp/internal/*` (or equivalent internal path) and MUST NOT be imported by non-MCP packages.

#### Scenario: Non-MCP package attempts to import shared core
- **WHEN** package outside `mcp/*` imports the internal shared core
- **THEN** build or static boundary checks reject the dependency

### Requirement: MCP shared core SHALL expose transport hook points
The shared core MUST allow transport-specific hooks for connection/reconnect strategy, queue/backpressure handling, and protocol-specific request execution while preserving shared semantic outcomes.

#### Scenario: Transport-specific reconnect is required
- **WHEN** call execution requires reconnect behavior under one transport
- **THEN** transport hook executes reconnect strategy while shared core preserves normalized event and diagnostics semantics

### Requirement: MCP refactor SHALL report duplicate-logic reduction ratio
The repository MUST produce a repeatable duplicate-logic comparison report for `mcp/http` and `mcp/stdio`, and MUST include relative percentage reduction against a documented baseline.

#### Scenario: Refactor validation is executed
- **WHEN** maintainers run the duplicate-logic report during change validation
- **THEN** output includes baseline value, current value, and relative reduction percentage for review