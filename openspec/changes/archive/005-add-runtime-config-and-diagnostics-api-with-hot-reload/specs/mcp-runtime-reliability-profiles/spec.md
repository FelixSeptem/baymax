## MODIFIED Requirements

### Requirement: MCP runtime SHALL provide named reliability profiles
The system MUST provide named MCP reliability profiles with documented defaults, including at least `dev`, `default`, `high-throughput`, and `high-reliability`. Profile parameters MUST be loadable from runtime configuration and resolved with precedence `env > file > default`; explicit per-call overrides MAY still override resolved profile fields.

#### Scenario: Profile is selected without overrides
- **WHEN** runtime starts with a named MCP profile
- **THEN** MCP call timeout, retry, backoff, reconnect, queue, and heartbeat settings use resolved profile values from the effective configuration

#### Scenario: Profile is selected with explicit override
- **WHEN** runtime starts with profile and explicit parameter overrides
- **THEN** resolved profile values apply first and explicit values override selected fields

### Requirement: MCP runtime SHALL expose normalized diagnostic summary
The runtime MUST expose a diagnostic summary for recent MCP calls, including latency, retry count, reconnect count, error class, and active profile, through library API endpoints that are consistent with runtime diagnostics APIs.

#### Scenario: Operator requests recent summary
- **WHEN** diagnostic summary for latest N MCP calls is requested through library API
- **THEN** output contains normalized fields that can be compared across `http` and `stdio`

## ADDED Requirements

### Requirement: MCP runtime SHALL fail fast on invalid reliability profile configuration
MCP runtime initialization MUST terminate with an error when reliability profile configuration is invalid; runtime MUST NOT continue with partially valid profile state.

#### Scenario: Startup profile config is invalid
- **WHEN** runtime loads reliability profile configuration containing invalid required fields or values
- **THEN** startup returns an error and no MCP transport is activated

#### Scenario: Hot reload profile config is invalid
- **WHEN** watched configuration updates profile values to an invalid state
- **THEN** runtime rejects that update, keeps the previously active profile configuration, and emits diagnostics for reload failure
