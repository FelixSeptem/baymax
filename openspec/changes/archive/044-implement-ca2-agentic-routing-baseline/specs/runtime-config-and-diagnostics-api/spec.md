## ADDED Requirements

### Requirement: Runtime config SHALL expose CA2 agentic routing controls with deterministic precedence
Runtime configuration MUST expose CA2 agentic routing controls under `context_assembler.ca2.agentic.*` with precedence `env > file > default`.

At minimum, runtime MUST support:
- callback decision timeout,
- callback failure policy.

For this milestone, callback failure policy MUST support `best_effort_rules`, meaning callback failures fallback to rule-based routing and do not terminate assemble flow.

Invalid timeout or unsupported failure policy values MUST fail fast during startup and hot reload.

#### Scenario: Startup with CA2 agentic routing overrides
- **WHEN** runtime starts with CA2 agentic controls defined in both YAML and environment variables
- **THEN** effective CA2 agentic controls resolve by `env > file > default`

#### Scenario: Invalid CA2 agentic timeout
- **WHEN** runtime configuration sets non-positive callback decision timeout
- **THEN** startup or hot reload fails fast with validation error

#### Scenario: Invalid CA2 agentic failure policy
- **WHEN** runtime configuration sets unsupported callback failure policy
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime diagnostics SHALL expose additive CA2 agentic routing fields
Runtime diagnostics MUST expose additive CA2 routing observability fields sufficient to triage agentic decision and fallback behavior, including:
- `stage2_router_mode`,
- `stage2_router_decision`,
- `stage2_router_reason`,
- `stage2_router_latency_ms`,
- `stage2_router_error`.

These fields MUST be backward-compatible and MUST NOT redefine existing CA2 Stage2 retrieval field semantics.

#### Scenario: Consumer inspects successful agentic routing decision
- **WHEN** application queries diagnostics for runs using `routing_mode=agentic` with successful callback decision
- **THEN** diagnostics include normalized router mode, decision, reason, and decision latency fields

#### Scenario: Consumer inspects callback failure fallback
- **WHEN** application queries diagnostics for runs using `routing_mode=agentic` and callback fails
- **THEN** diagnostics include normalized router error and fallback reason while preserving existing stage-policy behavior

### Requirement: Run and Stream SHALL preserve CA2 agentic routing diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent CA2 agentic routing diagnostics fields.

#### Scenario: Equivalent CA2 agentic routing diagnostics in Run and Stream
- **WHEN** equivalent requests execute under the same CA2 agentic routing configuration
- **THEN** diagnostics fields `stage2_router_mode|stage2_router_decision|stage2_router_reason|stage2_router_latency_ms|stage2_router_error` are semantically equivalent across Run and Stream
