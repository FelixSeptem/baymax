## ADDED Requirements

### Requirement: Runtime SHALL validate sandbox configuration with deterministic precedence and fail-fast safety
Runtime configuration MUST support sandbox governance fields with deterministic precedence `env > file > default`.

Sandbox configuration MUST be validated at startup and hot reload. Invalid sandbox config updates MUST fail fast and MUST NOT partially activate.

Sandbox config surface MUST include:
- policy fields (`enabled/mode/default_action/by_tool/fallback_action/fallback_action_by_tool`)
- executor fields (`backend/session_mode/required_capabilities`)
- profile fields (`profiles.<name>.network/filesystem/mounts/resource_limits/timeouts`)

#### Scenario: Startup with invalid sandbox enum value
- **WHEN** configuration sets unsupported sandbox mode, action, or fallback enum
- **THEN** runtime initialization fails fast with validation error

#### Scenario: Hot reload with invalid sandbox selector
- **WHEN** hot reload applies malformed sandbox `namespace+tool` selector
- **THEN** runtime rejects update and keeps previous active snapshot unchanged

#### Scenario: Startup with unsupported backend identifier
- **WHEN** config sets sandbox backend outside supported backend identifiers
- **THEN** runtime initialization fails fast with backend-invalid error

#### Scenario: Startup with unsatisfied required capability declaration
- **WHEN** required capability declaration contains unknown capability token
- **THEN** runtime initialization fails fast and does not start

### Requirement: Sandbox backend and capability identifiers SHALL use canonical stable enums
Sandbox backend identifiers and capability tokens MUST use canonical stable enums to keep cross-platform behavior deterministic.

Canonical backend identifiers for this milestone:
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job`

Canonical capability tokens for this milestone:
- `network_off`
- `network_egress_allowlist`
- `readonly_root`
- `mount_rw_allowlist`
- `cpu_limit`
- `memory_limit`
- `pid_limit`
- `session_per_call`
- `session_per_session`
- `stdout_stderr_capture`
- `oom_signal`

#### Scenario: Config references non-canonical backend identifier
- **WHEN** sandbox backend is set to identifier outside canonical backend enum
- **THEN** runtime fails fast with deterministic backend-invalid classification

#### Scenario: Config references non-canonical capability token
- **WHEN** required capability list includes token outside canonical capability enum
- **THEN** runtime fails fast with deterministic capability-token-invalid classification

### Requirement: Sandbox execution schema SHALL define field-level units and bounds
Sandbox configuration and execution schema MUST define deterministic units, defaults, and bounds for field-level interoperability.

At minimum:
- `resource_limits.cpu_milli` uses milli-CPU unit and MUST be `> 0`
- `resource_limits.memory_bytes` uses bytes and MUST be `> 0`
- `resource_limits.pid_limit` MUST be `> 0`
- `timeouts.launch_timeout` MUST be `> 0`
- `timeouts.exec_timeout` MUST be `> 0`
- `session_mode` MUST be one of `per_call|per_session`

#### Scenario: Config uses invalid resource limit bounds
- **WHEN** sandbox config sets cpu/memory/pid limits to non-positive values
- **THEN** runtime fails fast with deterministic resource-limit-invalid classification

#### Scenario: Config uses invalid timeout bounds
- **WHEN** sandbox config sets launch or execution timeout to non-positive value
- **THEN** runtime fails fast with deterministic timeout-invalid classification

### Requirement: Sandbox fallback defaults SHALL be deny-first for high-risk selectors
Sandbox fallback policy MUST be deny-first by default for high-risk tool selectors.

High-risk selector baseline for this milestone MUST include:
- `local+shell`
- `local+process_exec`
- `local+fs_write`
- `mcp+stdio_command`

`allow_and_record` host fallback for high-risk selectors MUST require explicit per-selector override in config.

#### Scenario: High-risk selector uses implicit fallback policy
- **WHEN** high-risk selector has no explicit fallback override
- **THEN** effective fallback action resolves to `deny`

#### Scenario: High-risk selector uses explicit allow override
- **WHEN** config explicitly sets high-risk selector fallback override to `allow_and_record`
- **THEN** runtime allows host fallback and records override source in diagnostics

### Requirement: Runtime diagnostics SHALL expose sandbox additive fields with compatibility guarantees
Runtime diagnostics MUST expose sandbox additive fields while preserving compatibility contract `additive + nullable + default`.

At minimum, run diagnostics MUST support:
- `sandbox_mode`
- `sandbox_backend`
- `sandbox_profile`
- `sandbox_session_mode`
- `sandbox_required_capabilities`
- `sandbox_decision`
- `sandbox_reason_code`
- `sandbox_fallback_used`
- `sandbox_fallback_reason`
- `sandbox_timeout_total`
- `sandbox_launch_failed_total`
- `sandbox_capability_mismatch_total`
- `sandbox_queue_wait_ms_p95`
- `sandbox_exec_latency_ms_p95`
- `sandbox_exit_code_last`
- `sandbox_oom_total`
- `sandbox_resource_cpu_ms_total`
- `sandbox_resource_memory_peak_bytes_p95`

#### Scenario: Consumer queries run diagnostics with sandbox activity
- **WHEN** a run executes one or more sandbox-governed tool calls
- **THEN** diagnostics return sandbox additive fields with deterministic semantics

#### Scenario: Consumer queries run diagnostics without sandbox activity
- **WHEN** a run executes without sandbox-governed paths
- **THEN** diagnostics preserve schema compatibility with nullable/default sandbox fields

### Requirement: Runtime SHALL keep sandbox diagnostics within cardinality and performance budgets
Sandbox additive fields MUST respect existing diagnostics cardinality governance and diagnostics-query performance baseline constraints.

#### Scenario: Sandbox-enriched diagnostics exceed cardinality budget
- **WHEN** sandbox metadata exceeds configured cardinality limits
- **THEN** runtime applies configured cardinality overflow policy without breaking contract compatibility

#### Scenario: Sandbox-enriched QueryRuns regression exceeds threshold
- **WHEN** diagnostics query benchmark runs with sandbox-enriched dataset and exceeds configured degradation thresholds
- **THEN** performance gate fails and blocks merge
