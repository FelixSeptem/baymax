# sandbox-execution-isolation Specification

## Purpose
TBD - created by archiving change introduce-sandbox-execution-isolation-contract-a51. Update Purpose after archive.
## Requirements
### Requirement: Runner SHALL support deterministic sandbox action resolution for tool execution
Runtime MUST resolve sandbox action per tool call using deterministic policy output:
- `host`
- `sandbox`
- `deny`

Action resolution MUST remain stable for equivalent input and effective configuration.

#### Scenario: Tool call resolves to host action
- **WHEN** sandbox policy resolves a tool call to `host`
- **THEN** runtime executes the call on host path and records sandbox decision as `host`

#### Scenario: Tool call resolves to sandbox action
- **WHEN** sandbox policy resolves a tool call to `sandbox`
- **THEN** runtime executes the call via sandbox executor path and records sandbox decision as `sandbox`

#### Scenario: Tool call resolves to deny action
- **WHEN** sandbox policy resolves a tool call to `deny`
- **THEN** runtime fail-fast denies the call and does not execute host or sandbox path

### Requirement: Runtime SHALL execute sandboxed calls through host-provided sandbox executor interface
Sandboxed execution MUST use host-provided executor interface and MUST normalize execution outcome fields (exit code, timeout, violation metadata) into tool result and diagnostics semantics.

#### Scenario: Sandboxed execution succeeds
- **WHEN** sandbox executor returns successful execution result
- **THEN** runtime maps output into normalized tool result without semantic drift from non-sandbox success path

#### Scenario: Sandboxed execution fails with timeout
- **WHEN** sandbox executor returns timeout-class failure
- **THEN** runtime emits deterministic timeout reason code and marks sandbox timeout observability fields

### Requirement: Sandbox execution SHALL use canonical ExecSpec and ExecResult contracts
Runtime MUST build sandbox execution requests and responses through canonical contracts so different sandbox backends remain semantically interchangeable.

Canonical ExecSpec MUST include at minimum:
- command + arguments
- sanitized environment
- workdir and mount policy
- network policy
- resource limits (cpu/memory/pid)
- session mode (`per_call|per_session`)
- launch and execution timeouts

Canonical ExecResult MUST include at minimum:
- exit code
- stdout/stderr payload
- timeout marker
- oom marker
- violation codes
- normalized resource usage summary

ExecResult canonical violation codes for this milestone MUST include:
- `sandbox.timeout`
- `sandbox.oom`
- `sandbox.network_violation`
- `sandbox.mount_violation`
- `sandbox.capability_mismatch`
- `sandbox.runtime_launch_failed`

ExecSpec normalization rules for this milestone MUST include:
- empty or inherited environment entries MUST be removed during sanitization
- mount entries MUST be normalized to deterministic order before execution
- unresolved relative workdir MUST be normalized to deterministic absolute workdir or rejected

#### Scenario: Equivalent tool call across two sandbox backends
- **WHEN** the same canonical ExecSpec is executed by two backend implementations that satisfy required capabilities
- **THEN** runtime receives semantically equivalent ExecResult classification and applies the same decision mapping

#### Scenario: Sandbox backend returns backend-specific error payload
- **WHEN** backend returns vendor-specific error fields
- **THEN** runtime normalizes output into canonical ExecResult fields without leaking backend-only semantics into contracts

#### Scenario: Sandbox execution returns unordered mount metadata
- **WHEN** backend receives mount inputs in non-deterministic order
- **THEN** runtime executes with canonical normalized mount ordering and preserves deterministic result classification

### Requirement: Sandbox executor capability negotiation SHALL gate enforce-mode execution
Runtime MUST evaluate sandbox executor capability probe before enforce-mode sandbox execution when required capabilities are configured.

If required capability is missing, runtime MUST deny execution deterministically under required/enforce policy.

#### Scenario: Required capability is missing
- **WHEN** effective policy requires `network_off` and executor capability probe reports unsupported
- **THEN** runtime denies execution with canonical capability-mismatch reason and does not execute host fallback by default

#### Scenario: Required capability set is satisfied
- **WHEN** executor capability probe satisfies configured required capability set
- **THEN** runtime allows sandbox execution path to proceed

### Requirement: Sandbox launch and execution fallback SHALL follow deterministic policy
Runtime MUST support deterministic fallback behavior when sandbox launch/execution fails, controlled by effective fallback policy.

Supported fallback actions:
- `allow_and_record`
- `deny`

#### Scenario: Sandbox launch fails with allow-and-record fallback
- **WHEN** sandbox launch fails and fallback policy is `allow_and_record`
- **THEN** runtime executes host path and records sandbox fallback usage with canonical reason

#### Scenario: Sandbox launch fails with deny fallback
- **WHEN** sandbox launch fails and fallback policy is `deny`
- **THEN** runtime fail-fast denies execution and does not execute host path

### Requirement: MCP stdio command startup SHALL support sandboxed launcher path
Runtime MUST support sandbox-aware command startup for `mcp/stdio` transport when policy resolves command execution to sandbox path.

#### Scenario: MCP stdio command starts through sandbox launcher
- **WHEN** effective policy requires sandbox for mcp stdio command startup
- **THEN** runtime delegates command startup to sandbox executor/launcher path and preserves MCP transport contract semantics

#### Scenario: MCP stdio sandbox startup fails under deny fallback
- **WHEN** sandbox startup for mcp stdio command fails and fallback policy is `deny`
- **THEN** runtime returns deterministic fail-fast error and does not start host command

### Requirement: MCP stdio sandbox lifecycle SHALL preserve deterministic session semantics
Sandboxed MCP stdio startup MUST support deterministic session mode semantics:
- `per_call`
- `per_session`

Reconnect and close behavior MUST remain deterministic within the selected session mode.

#### Scenario: Per-session mode keeps stable sandboxed transport lifecycle
- **WHEN** mcp stdio sandbox session mode is `per_session`
- **THEN** runtime reuses the same sandboxed transport for calls in that session until explicit close or terminal failure

#### Scenario: Per-call mode isolates each invocation
- **WHEN** mcp stdio sandbox session mode is `per_call`
- **THEN** each invocation uses an isolated sandbox execution unit and teardown is performed per call

#### Scenario: Per-session sandbox transport crashes unexpectedly
- **WHEN** sandboxed mcp stdio transport crashes during `per_session` lifecycle
- **THEN** runtime performs deterministic reconnect or terminal-fail classification according to configured policy without ambiguity

#### Scenario: Cancel signal arrives during per-call sandbox invocation
- **WHEN** request context is canceled while per-call sandbox execution is running
- **THEN** runtime terminates sandbox invocation deterministically and returns canonical cancellation/timeout classification

#### Scenario: Repeated close on sandboxed mcp session
- **WHEN** close is invoked multiple times on the same sandboxed mcp session
- **THEN** close semantics remain idempotent and do not produce duplicate terminal side effects

### Requirement: Tool execution SHALL support sandbox adaptation bridge for in-process tools
Runtime MUST support a deterministic bridge for existing in-process tools that do not natively expose process execution plans.

For tools lacking sandbox execution adapter:
- `observe` mode MUST allow host execution and record adaptation-missing observability markers.
- `enforce` mode MUST deny execution unless explicit fallback policy permits host path.

#### Scenario: In-process tool without adapter under observe mode
- **WHEN** tool resolves to sandbox action but no sandbox adapter is available and mode is `observe`
- **THEN** runtime executes host path and records `sandbox_tool_not_adapted` marker

#### Scenario: In-process tool without adapter under enforce mode
- **WHEN** tool resolves to sandbox action but no sandbox adapter is available and mode is `enforce`
- **THEN** runtime denies execution with canonical adaptation-missing reason unless explicit fallback overrides

### Requirement: Run and Stream SHALL preserve sandbox semantic equivalence
For equivalent request and effective configuration, Run and Stream MUST produce semantically equivalent sandbox decisions, reason codes, and fallback outcomes.

#### Scenario: Equivalent sandbox deny in Run and Stream
- **WHEN** equivalent Run and Stream requests resolve to sandbox deny
- **THEN** both paths return semantically equivalent deny classification and no execution side effects

#### Scenario: Equivalent sandbox fallback in Run and Stream
- **WHEN** equivalent Run and Stream requests hit sandbox launch failure with same fallback policy
- **THEN** both paths produce semantically equivalent fallback decision and diagnostics semantics

### Requirement: Sandbox action resolution SHALL remain deterministic across ReAct iterations
For equivalent selector, effective sandbox policy, and runtime snapshot, sandbox action resolution (`host|sandbox|deny`) MUST remain deterministic across multiple ReAct loop iterations.

#### Scenario: Equivalent tool selector appears in multiple ReAct iterations
- **WHEN** the same tool selector is invoked in successive ReAct iterations with unchanged effective config
- **THEN** runtime resolves semantically equivalent sandbox action on each iteration

#### Scenario: Equivalent selector appears in Run and Stream ReAct loops
- **WHEN** equivalent Run and Stream ReAct loops invoke the same tool selector under unchanged config
- **THEN** both paths resolve semantically equivalent sandbox action classification

### Requirement: Sandbox fallback behavior in ReAct loops SHALL preserve canonical taxonomy
When sandbox execution fails during ReAct loop, runtime MUST apply configured fallback policy deterministically and MUST preserve canonical fallback reason taxonomy per iteration.

#### Scenario: ReAct iteration hits sandbox launch failure with allow-and-record fallback
- **WHEN** sandbox launch fails during a ReAct iteration and fallback policy is `allow_and_record`
- **THEN** runtime executes host fallback and records canonical fallback reason for that iteration

#### Scenario: ReAct iteration hits sandbox launch failure with deny fallback
- **WHEN** sandbox launch fails during a ReAct iteration and fallback policy is `deny`
- **THEN** runtime fail-fast denies the tool call and maps loop termination to canonical sandbox failure classification

### Requirement: Sandbox capability mismatch in ReAct loop SHALL terminate deterministically
If sandbox required capability negotiation fails for a ReAct tool call in enforce mode, runtime MUST terminate deterministically with canonical capability-mismatch semantics and MUST NOT silently downgrade execution path.

#### Scenario: ReAct tool call requires unsupported sandbox capability
- **WHEN** sandbox capability probe reports missing required capability for dispatched ReAct tool call
- **THEN** runtime terminates tool step with canonical `sandbox.capability_mismatch` classification

#### Scenario: Equivalent capability mismatch under Run and Stream
- **WHEN** equivalent Run and Stream ReAct loops encounter same capability mismatch
- **THEN** both paths produce semantically equivalent terminal classification and fallback usage semantics

### Requirement: Sandbox execution SHALL apply canonical egress policy resolution per tool selector
For each sandbox-governed tool selector, runtime MUST resolve egress action deterministically using effective egress policy and allowlist rules.

#### Scenario: Equivalent selector with unchanged policy
- **WHEN** the same selector executes multiple times under unchanged egress configuration
- **THEN** runtime resolves semantically equivalent egress action and reason code each time

#### Scenario: Selector-specific override exists
- **WHEN** per-selector egress override is configured
- **THEN** runtime uses selector override instead of global default action

### Requirement: Egress deny path SHALL preserve sandbox fail-fast semantics
When egress action resolves to deny in enforce mode, runtime MUST block the outbound action before execution side effects and emit canonical violation classification.

#### Scenario: Outbound request denied in enforce mode
- **WHEN** sandbox egress policy resolves outbound target to deny
- **THEN** runtime blocks request and returns canonical `sandbox.egress_deny` classification

#### Scenario: Equivalent deny under Run and Stream
- **WHEN** equivalent Run and Stream requests hit identical egress deny condition
- **THEN** both paths produce semantically equivalent deny classification

### Requirement: Egress allow-and-record SHALL keep deterministic observability semantics
When policy action is `allow_and_record`, runtime MUST allow outbound action and persist deterministic observability markers for policy source and decision.

#### Scenario: Allow-and-record action executes outbound request
- **WHEN** egress policy resolves to `allow_and_record`
- **THEN** runtime executes request and records canonical decision metadata

#### Scenario: Equivalent allow-and-record events are replayed
- **WHEN** duplicate equivalent egress decision events are ingested
- **THEN** logical egress aggregate counters remain replay-idempotent

