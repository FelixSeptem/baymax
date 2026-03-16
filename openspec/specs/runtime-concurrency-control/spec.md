# runtime-concurrency-control Specification

## Purpose
TBD - created by archiving change optimize-runtime-concurrency-and-async-io. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide unified concurrency control configuration
The runtime MUST expose a unified configuration object for concurrency behavior, including worker concurrency, queue capacity, backpressure mode, timeout budget, and cancellation policy.

#### Scenario: Runtime starts with default configuration
- **WHEN** no explicit concurrency configuration is provided
- **THEN** the runtime applies documented safe defaults for concurrency and queue handling

#### Scenario: Runtime starts with custom concurrency settings
- **WHEN** a user provides explicit concurrency and queue parameters
- **THEN** the runtime honors the provided parameters consistently across runner/tool/mcp execution paths

### Requirement: Runtime SHALL enforce explicit backpressure behavior under load
The runtime MUST enforce deterministic backpressure behavior when incoming work exceeds configured capacity.

#### Scenario: Queue capacity is exceeded
- **WHEN** incoming tool or async jobs exceed configured queue size
- **THEN** the runtime applies configured backpressure action (reject, block, or degrade) and emits a diagnosable event

### Requirement: Runtime SHALL propagate cancellation and converge goroutines
The runtime MUST propagate cancellation signals across spawned workers and converge active goroutines within bounded time.

#### Scenario: Parent context is canceled
- **WHEN** a run context is canceled during concurrent execution
- **THEN** spawned workers stop accepting new work and active work exits according to timeout policy

#### Scenario: Timeout triggers cancellation
- **WHEN** execution exceeds configured timeout budget
- **THEN** runtime emits timeout-classified failure and stops further fanout dispatch

### Requirement: Runtime SHALL support high fanout local tool execution
The tool execution path MUST support high fanout read-only tasks using goroutine concurrency without violating ordering guarantees for write operations.

#### Scenario: Mixed read and write tool calls
- **WHEN** a dispatch round contains both read-only and write-aware tools
- **THEN** read-only tools execute concurrently while write-aware tools preserve serialized execution guarantees

### Requirement: Runtime SHALL emit concurrency diagnostics for each run
The runtime MUST emit diagnostics including queue depth, queue wait time, fanout size, retry count, and cancellation reason.

#### Scenario: Concurrent run completes
- **WHEN** a run with concurrent work finishes
- **THEN** emitted events contain metrics required to diagnose queueing and fanout behavior

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

### Requirement: Runtime concurrency config SHALL accept drop_low_priority enum
Runtime MUST treat `drop_low_priority` as a valid enum value for concurrency backpressure configuration in addition to existing modes.

#### Scenario: Config validation accepts drop_low_priority
- **WHEN** configuration sets `concurrency.backpressure=drop_low_priority`
- **THEN** runtime validation passes for enum check

### Requirement: Default concurrency backpressure policy SHALL remain block after drop-low-priority scope expansion
Expanding `drop_low_priority` applicability MUST NOT change the default runtime backpressure behavior. When backpressure mode is not explicitly configured, the runtime SHALL continue using `block` semantics.

#### Scenario: Runtime starts without explicit backpressure config
- **WHEN** concurrency config omits backpressure mode
- **THEN** runtime behavior remains `block` and no low-priority dropping is applied

