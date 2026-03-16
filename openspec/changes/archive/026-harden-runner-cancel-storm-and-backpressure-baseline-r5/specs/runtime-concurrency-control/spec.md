## ADDED Requirements

### Requirement: Runtime SHALL default backpressure mode to block
The runtime MUST treat `block` as the default backpressure mode for runner concurrency control when no explicit override is provided.

#### Scenario: Startup with no backpressure override
- **WHEN** runtime starts without explicit backpressure mode configuration
- **THEN** effective backpressure mode is `block`

#### Scenario: Startup with explicit backpressure override
- **WHEN** runtime starts with a valid configured backpressure mode
- **THEN** runtime applies the configured mode and does not silently fall back to a different mode

### Requirement: Runtime SHALL converge under cancellation storm across tool, mcp, and skill paths
The runtime MUST propagate parent-context cancellation to all in-flight worker branches and MUST prevent new dispatch after cancellation for tool, mcp, and skill execution paths.

#### Scenario: Parent context canceled during tool fanout
- **WHEN** parent run context is canceled while multiple tool calls are in flight
- **THEN** runner stops accepting new tool dispatch and in-flight branches converge according to timeout policy

#### Scenario: Parent context canceled during mcp activity
- **WHEN** parent run context is canceled while MCP call dispatch or retry loop is active
- **THEN** runtime stops further MCP dispatch/retry fanout and converges active goroutines within bounded time

#### Scenario: Parent context canceled during skill lifecycle
- **WHEN** parent run context is canceled while skill discovery/trigger/compile pipeline is active
- **THEN** runtime terminates pending skill work and emits terminal cancellation semantics

### Requirement: Run and Stream SHALL preserve cancellation and backpressure semantic equivalence
For equivalent execution inputs and cancellation timing classes, Run and Stream paths MUST expose semantically equivalent cancellation outcome and backpressure behavior.

#### Scenario: Equivalent cancellation in Run and Stream
- **WHEN** Run and Stream execute equivalent requests and receive cancellation in the same execution phase
- **THEN** both paths emit semantically equivalent terminal classification and stop further dispatch

#### Scenario: Equivalent backpressure in Run and Stream
- **WHEN** Run and Stream execute equivalent high-fanout requests that hit concurrency limits
- **THEN** both paths apply the same configured backpressure policy and preserve consistent fail-fast semantics
