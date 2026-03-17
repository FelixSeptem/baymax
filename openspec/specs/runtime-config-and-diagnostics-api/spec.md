# runtime-config-and-diagnostics-api Specification

## Purpose
TBD - created by archiving change add-runtime-config-and-diagnostics-api-with-hot-reload. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`. This configuration capability MUST be owned by a global runtime module and be consumable by MCP, runner, local tool, skill loader, and observability components through stable interfaces.

For provider capability fallback, runtime configuration MUST additionally support validated fallback policy fields (including ordered provider candidates and discovery/cache controls) under the same precedence and validation pipeline.

Runtime configuration MUST additionally support context assembler CA1 baseline fields with deterministic precedence, including `context_assembler.enabled` (default true), file journal path, prefix version, guard fail-fast toggle, and storage backend selector (file active, db placeholder).

Runtime configuration MUST additionally support context assembler CA2 fields with deterministic precedence, including staged assembly enablement, stage routing mode, stage-level failure policy, stage timeouts, and Stage2 provider selector (`file|http|rag|db|elasticsearch`) with provider-level endpoint/auth/JSON-mapping controls.

For CA2 external retriever, runtime configuration MUST support `external.profile` defaults (`http_generic`, `ragflow_like`, `graphrag_like`, `elasticsearch_like`) and MUST apply explicit config overrides on top of selected profile defaults.

Runtime configuration MUST additionally support security S1 baseline fields with deterministic precedence, including security scan mode (`strict|warn`), scan tool toggles, redaction enablement, and extensible sensitive-key keyword lists.

#### Scenario: Startup with file and environment overrides
- **WHEN** runtime starts with a YAML config file and overlapping environment variables
- **THEN** effective configuration uses environment values first, then file values, then defaults for unset keys

#### Scenario: Startup without config file
- **WHEN** runtime starts without a config file
- **THEN** runtime uses default values and applicable environment overrides

#### Scenario: Startup with fallback policy overrides
- **WHEN** runtime starts with fallback policy defined in both YAML and environment variables
- **THEN** effective fallback policy resolves using `env > file > default` and is available to model-step provider selection

#### Scenario: Startup with context assembler defaults
- **WHEN** runtime starts without explicit context assembler overrides
- **THEN** context assembler baseline config resolves with default enabled state and valid file-backed journal settings

#### Scenario: Startup with CA2 stage policy overrides
- **WHEN** runtime starts with CA2 stage policy fields defined in both YAML and environment variables
- **THEN** effective CA2 stage policy resolves using `env > file > default` and is available to assembler routing and stage execution

#### Scenario: Startup with security baseline overrides
- **WHEN** runtime starts with security scan and redaction fields defined in both YAML and environment variables
- **THEN** effective security baseline configuration resolves using `env > file > default` and is available to quality-gate and runtime redaction flow

#### Scenario: Startup with external retriever profile overrides
- **WHEN** runtime starts with CA2 external profile selected and explicit mapping/auth/header overrides configured
- **THEN** effective Stage2 external config resolves profile defaults first and then applies explicit override values

### Requirement: Runtime SHALL validate configuration and fail fast on invalid startup input
The runtime MUST validate required fields, numeric ranges, and enum values before activation; invalid startup configuration MUST return an error and abort initialization.

#### Scenario: Invalid enum value at startup
- **WHEN** configuration provides an unsupported enum value
- **THEN** runtime returns validation error and does not start

#### Scenario: Invalid numeric range at startup
- **WHEN** configuration contains out-of-range numeric values
- **THEN** runtime returns validation error and does not start

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support. Diagnostics returned by these APIs MUST follow single-writer and idempotent persistence semantics so repeated event submission does not alter logical aggregate counts.

Diagnostics MUST include capability-preflight and provider-fallback summary fields for each affected model step, including requested capability set, candidate providers considered, selected provider, and fail-fast reason when chain is exhausted.

Diagnostics MUST additionally include context assembler CA1 baseline fields for each assemble cycle and related run summary context, including `prefix_hash`, `assemble_latency_ms`, `assemble_status`, and `guard_violation`.

Diagnostics MUST additionally include context assembler CA2 stage and recap fields, including normalized stage statuses, stage2 skip reason, stage latencies, and recap status.

Diagnostics and event payloads MUST additionally apply unified S1 redaction policy before persistence and emission.

Diagnostics for CA2 retrieval MUST additionally expose normalized Stage2 retrieval summary fields: `stage2_hit_count`, `stage2_source`, `stage2_reason`, `stage2_reason_code`, `stage2_error_layer`, and `stage2_profile`.

Diagnostics for Action Timeline H1.5 MUST additionally expose run-level phase aggregates with minimum fields per phase: `count_total`, `failed_total`, `canceled_total`, `skipped_total`, `latency_ms`, and `latency_p95_ms`.

#### Scenario: Consumer requests recent run diagnostics
- **WHEN** application calls diagnostics API for recent runs
- **THEN** runtime returns bounded summary records with normalized fields and without duplicated logical run entries caused by retries or replay

#### Scenario: Consumer requests effective configuration
- **WHEN** application calls API to fetch effective configuration
- **THEN** runtime returns a sanitized snapshot that masks secret-like fields

#### Scenario: Consumer inspects fallback diagnostics
- **WHEN** application queries diagnostics for a run that triggered provider fallback
- **THEN** runtime returns normalized fallback summary fields sufficient to reconstruct capability decision path

#### Scenario: Consumer inspects context assembler diagnostics
- **WHEN** application queries diagnostics for runs with context assembler enabled
- **THEN** runtime returns assembler baseline fields that allow verification of prefix consistency and guard outcomes

#### Scenario: Consumer inspects CA2 stage diagnostics
- **WHEN** application queries diagnostics for runs with CA2 staged assembly enabled
- **THEN** runtime returns normalized stage and recap fields sufficient to reconstruct Stage1/Stage2 routing outcome

#### Scenario: Consumer inspects redaction behavior
- **WHEN** application queries diagnostics containing sensitive-key fields
- **THEN** returned payload contains masked values according to active redaction policy

#### Scenario: Consumer inspects Stage2 retrieval summary
- **WHEN** application queries diagnostics for runs that executed Stage2 retrieval
- **THEN** runtime returns normalized `stage2_hit_count`, `stage2_source`, `stage2_reason`, `stage2_reason_code`, `stage2_error_layer`, and `stage2_profile` fields

#### Scenario: Consumer inspects action timeline phase aggregates
- **WHEN** application queries diagnostics for runs with action timeline events
- **THEN** runtime returns phase-level aggregate metrics including counts and `latency_p95_ms`

### Requirement: Runtime SHALL support hot reload with atomic swap and rollback safety
The runtime MUST watch config file changes, rebuild and validate a new snapshot, and atomically replace active configuration only on successful validation.

#### Scenario: Valid configuration update arrives
- **WHEN** watched YAML file changes to a valid configuration
- **THEN** runtime atomically switches to the new snapshot without exposing partial state

#### Scenario: Invalid configuration update arrives
- **WHEN** watched YAML file changes to an invalid configuration
- **THEN** runtime rejects the update, keeps current active snapshot unchanged, and emits reload error diagnostics

### Requirement: Runtime SHALL be concurrency-safe for config and diagnostics access
Configuration reads, diagnostics writes, diagnostics deduplication, and hot-reload swaps MUST be safe under concurrent goroutines.

#### Scenario: Concurrent reads during hot reload
- **WHEN** multiple goroutines read configuration while a reload is in progress
- **THEN** each read observes either old or new complete snapshot, never mixed fields

#### Scenario: Concurrent diagnostics recording and querying
- **WHEN** goroutines concurrently record call summaries and query diagnostics
- **THEN** runtime preserves data integrity, idempotent write behavior, and bounded-memory behavior

### Requirement: Runtime config API migration SHALL preserve behavior compatibility
When runtime config implementation paths are refactored, the system MUST preserve existing precedence, validation, and hot-reload semantics for callers migrating from previous package locations.

#### Scenario: Caller migrates from legacy package path
- **WHEN** caller switches from legacy runtime config package path to the new global runtime package path
- **THEN** resolved effective config and diagnostics behavior remain semantically equivalent under the same inputs

### Requirement: Runtime diagnostics API migration SHALL preserve behavior compatibility
When diagnostics API implementation paths are refactored, the system MUST preserve existing normalized fields, bounded history semantics, and sanitized config output behavior for callers migrating from previous package locations.

#### Scenario: Caller migrates diagnostics API package path
- **WHEN** caller switches diagnostics API imports from legacy package path to new global runtime package path
- **THEN** recent runs/calls/reloads outputs remain semantically equivalent for the same recorded inputs

### Requirement: Runtime diagnostics API SHALL cover skill lifecycle semantics
The runtime diagnostics API MUST support normalized skill lifecycle diagnostics, including discovery, trigger matching, compile outcomes, and failure classification, while preserving shared correlation fields.

#### Scenario: Skill loader emits diagnostics
- **WHEN** skill discovery or compile pipeline runs
- **THEN** diagnostics API returns normalized skill records with shared correlation fields and skill-specific payload metadata

### Requirement: Runtime documentation SHALL publish config field index and migration mapping
The repository MUST publish a configuration field index and package migration mapping, including old-to-new API references and deprecation notes.

#### Scenario: Maintainer updates runtime config docs
- **WHEN** config fields or package paths change
- **THEN** docs include synchronized schema reference and migration table for affected APIs

### Requirement: Runtime diagnostics contract SHALL define normalized status and error semantics
The runtime MUST define shared diagnostics status enums and error classification semantics for run and skill records, while allowing domain-specific extension fields.

#### Scenario: Run and skill producers emit diagnostics
- **WHEN** runner and skill loader publish diagnostics records
- **THEN** persisted diagnostics use shared normalized status and error fields with consistent meanings

### Requirement: Runtime diagnostics contract SHALL be protected by contract tests
The repository MUST include contract tests that validate schema and semantic consistency across success, failure, warning, and retry/replay paths for run and skill diagnostics.

#### Scenario: Contract test suite is executed
- **WHEN** diagnostics contract tests run in CI or local validation
- **THEN** inconsistent field mapping or semantic mismatch fails the test suite

### Requirement: Runtime SHALL validate provider fallback policy and discovery controls
The runtime MUST validate fallback policy configuration at startup and hot reload, including non-empty candidate constraints (when fallback is enabled), enum/range validation for discovery controls, and deterministic ordering guarantees.

#### Scenario: Invalid fallback policy at startup
- **WHEN** fallback configuration includes invalid provider identifiers or malformed ordering
- **THEN** runtime initialization fails fast with validation error

#### Scenario: Invalid fallback policy during hot reload
- **WHEN** watched configuration updates fallback policy to an invalid value
- **THEN** runtime rejects the update and preserves previous active snapshot

### Requirement: Runtime SHALL enforce CA1 storage backend behavior for context assembler
Runtime MUST treat file backend as active default for context assembler CA1 and MUST reject db backend activation with explicit unsupported error until later milestones enable it.

#### Scenario: DB backend requested in CA1
- **WHEN** runtime configuration sets context assembler storage backend to db during CA1
- **THEN** initialization fails fast with backend-not-ready error and no partial activation occurs

### Requirement: Runtime SHALL validate CA2 stage provider and routing mode constraints
The runtime MUST validate CA2 stage provider and routing mode enums at startup and hot reload. Unsupported modes or provider selections in current milestone MUST fail fast with explicit classification.

#### Scenario: Unsupported CA2 routing mode
- **WHEN** runtime configuration sets unknown CA2 routing mode
- **THEN** initialization fails fast with validation error

#### Scenario: CA2 rag/db provider requested before implementation
- **WHEN** runtime configuration selects Stage2 provider as rag or db in CA2 milestone
- **THEN** runtime returns explicit provider-not-ready classification and does not partially activate staged assembly

### Requirement: Runtime SHALL validate security baseline scan and redaction config at startup and hot reload
The runtime MUST validate security scan mode enums, redaction strategy configuration, and keyword list constraints during startup and hot reload; invalid values MUST fail fast.

#### Scenario: Invalid scan mode
- **WHEN** runtime configuration sets unsupported scan mode
- **THEN** initialization fails fast with validation error

#### Scenario: Invalid redaction keyword config
- **WHEN** runtime configuration sets malformed redaction keyword list
- **THEN** initialization fails fast with validation error and no partial activation

### Requirement: Runtime SHALL validate external retriever config and fail fast on invalid mapping
Runtime MUST validate Stage2 external retriever configuration (provider enum, endpoint/auth fields, profile values, JSON mapping schema) at startup and hot reload; invalid values MUST fail fast with explicit validation errors.

Runtime MUST treat warning-class findings as non-blocking and MUST treat error-class findings as blocking for startup/hot reload activation.

#### Scenario: Invalid Stage2 provider enum
- **WHEN** runtime configuration sets unsupported Stage2 provider value
- **THEN** initialization fails fast with validation error

#### Scenario: Invalid HTTP mapping configuration
- **WHEN** runtime configuration defines malformed request/response JSON mapping
- **THEN** initialization fails fast with mapping validation error and no partial activation

#### Scenario: Missing required endpoint for external provider
- **WHEN** runtime configuration enables http/rag/db/elasticsearch provider without required endpoint fields
- **THEN** initialization fails fast with validation error

#### Scenario: Invalid external profile value
- **WHEN** runtime configuration sets unsupported external profile value
- **THEN** initialization fails fast with validation error and no partial activation

### Requirement: Runtime SHALL expose external retriever precheck API for library integrations
The runtime MUST provide a library-level precheck API for CA2 external retriever configuration. The API MUST return normalized findings that include severity (`warning` or `error`) and machine-readable reason codes.

`warning` findings MUST allow execution to continue, and `error` findings MUST require fail-fast behavior when used in startup or hot reload validation paths.

#### Scenario: Precheck returns warning findings only
- **WHEN** application runs precheck and receives only warning findings
- **THEN** runtime allows startup/hot reload to continue and exposes warnings for observability

#### Scenario: Precheck returns error finding
- **WHEN** application runs precheck and receives at least one error finding
- **THEN** runtime blocks startup/hot reload activation with explicit fail-fast validation error

### Requirement: Runtime SHALL enable action timeline emission by default
Runtime event emission MUST enable Action Timeline output by default without requiring additional runtime configuration toggles.

#### Scenario: Runtime starts with default configuration
- **WHEN** application starts runtime without timeline-specific overrides
- **THEN** timeline events are emitted and consumable by library integrations

### Requirement: Runtime diagnostics contract SHALL defer timeline aggregation fields in H1 with explicit TODO traceability
H1 MUST NOT introduce new timeline aggregation fields into persisted diagnostics run records. The repository documentation MUST record an explicit TODO for follow-up change(s) that converge timeline observability aggregation.

This constraint applies only to H1 scope. Starting from H1.5, timeline aggregate diagnostics fields are allowed and expected under backward-compatible field extension rules.

#### Scenario: Consumer queries diagnostics during H1
- **WHEN** application queries diagnostics APIs after timeline event rollout
- **THEN** existing diagnostics field schema remains stable without new timeline aggregate fields

#### Scenario: Maintainer reviews runtime docs after H1 rollout
- **WHEN** maintainer checks README and runtime diagnostics documentation
- **THEN** documentation contains explicit TODO notes for future timeline aggregation convergence

#### Scenario: Consumer queries diagnostics during H1.5+
- **WHEN** application queries diagnostics APIs after H1.5 observability convergence
- **THEN** run diagnostics include timeline aggregate fields while preserving backward compatibility for existing consumers

### Requirement: Runtime SHALL expose CA3 pressure-control configuration with deterministic precedence
Runtime configuration MUST support CA3 pressure-control fields with deterministic precedence `env > file > default`, including tier thresholds, absolute token limits, emergency protection behavior, and spill/swap file backend parameters.

#### Scenario: Startup with CA3 threshold overrides
- **WHEN** YAML and environment variables both define CA3 pressure thresholds
- **THEN** effective CA3 configuration resolves with `env > file > default` precedence

#### Scenario: Invalid CA3 threshold configuration
- **WHEN** CA3 thresholds are malformed, overlapping, or out of range
- **THEN** runtime fails fast during startup or hot reload and retains previous valid snapshot

### Requirement: Runtime diagnostics SHALL include CA3 pressure and recovery aggregates
Run diagnostics MUST include CA3 observability fields at minimum for zone residency duration, trigger counts, compression ratio, spill count, and swap-back count.

#### Scenario: Consumer inspects run diagnostics after CA3 pressure event
- **WHEN** a run triggers CA3 pressure controls
- **THEN** diagnostics contain CA3 aggregate fields sufficient to identify zone transitions and mitigation actions

#### Scenario: Consumer inspects run diagnostics after replay with recovery
- **WHEN** replay executes for a run that previously triggered spill/swap
- **THEN** diagnostics include recovery-related counters and preserve consistent aggregate semantics

### Requirement: Diagnostics SHALL expose CA4 token-counting semantics clearly
Runtime diagnostics documentation and fields MUST clarify that CA4 token counts are used for threshold strategy control, with explicit fallback semantics and non-blocking behavior.

#### Scenario: Token counting falls back during execution
- **WHEN** provider or local tokenizer counting fails and fallback is used
- **THEN** diagnostics semantics remain consistent and execution continues without run termination caused solely by counting failure

### Requirement: Configuration docs SHALL define CA4 threshold resolution order
Runtime configuration documentation MUST describe the exact resolution order among global thresholds, stage overrides, and mixed trigger selection.

#### Scenario: Operator reads CA4 config guide
- **WHEN** operator configures global and stage thresholds
- **THEN** operator can determine effective thresholds and conflict resolution deterministically from docs

### Requirement: Runtime config SHALL define Action Gate defaults and policy fields
Runtime configuration MUST support Action Gate policy fields with deterministic precedence `env > file > default`. Default policy MUST be `require_confirm`. Runtime MUST provide timeout configuration for confirmation resolution, with timeout outcome interpreted as deny.

Runtime configuration MUST additionally support parameter-rule fields for Action Gate, including rule identifiers, condition trees (`and`/`or`), operators, optional per-rule action override, and evaluation priority semantics.

#### Scenario: Startup with no Action Gate override
- **WHEN** runtime starts without Action Gate config overrides
- **THEN** effective Action Gate policy is `require_confirm` and timeout-deny behavior is enabled

#### Scenario: Startup with Action Gate overrides
- **WHEN** Action Gate fields are provided in both YAML and environment variables
- **THEN** effective Action Gate settings resolve by `env > file > default`

#### Scenario: Startup with invalid parameter-rule config
- **WHEN** Action Gate parameter-rule config contains malformed condition tree or unsupported operator
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

### Requirement: Runtime diagnostics SHALL expose minimal Action Gate counters
Run diagnostics MUST expose minimal Action Gate counters including `gate_checks`, `gate_denied_count`, and `gate_timeout_count`.

Run diagnostics MUST additionally expose minimal parameter-rule counters/metadata fields including `gate_rule_hit_count` and `gate_rule_last_id`.

#### Scenario: Consumer inspects run diagnostics with gated actions
- **WHEN** a run performs Action Gate checks for one or more tool actions
- **THEN** diagnostics include non-negative values for `gate_checks`, `gate_denied_count`, and `gate_timeout_count`

#### Scenario: Consumer inspects run diagnostics with parameter-rule hit
- **WHEN** a run triggers at least one parameter-level rule match
- **THEN** diagnostics include non-negative `gate_rule_hit_count` and a stable `gate_rule_last_id` value

#### Scenario: Consumer inspects run diagnostics without gate activity
- **WHEN** a run does not trigger any Action Gate check
- **THEN** diagnostics expose zero-value counters without breaking existing diagnostics schema compatibility

### Requirement: Runtime config SHALL define H3 clarification timeout policy
Runtime configuration MUST support H3 clarification fields with deterministic precedence `env > file > default`, including `enabled`, clarification timeout, and timeout policy. Default timeout policy MUST be `cancel_by_user`.

#### Scenario: Startup with default clarification config
- **WHEN** runtime starts without clarification overrides
- **THEN** clarification HITL is enabled with configured default timeout and `cancel_by_user` timeout policy

#### Scenario: Startup with clarification overrides
- **WHEN** clarification fields are configured in YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

### Requirement: Runtime diagnostics SHALL expose minimal H3 clarification counters
Run diagnostics MUST expose minimal clarification counters including `await_count`, `resume_count`, and `cancel_by_user_count`.

#### Scenario: Consumer inspects run diagnostics with clarification flow
- **WHEN** a run triggers clarification wait and resume/cancel lifecycle
- **THEN** diagnostics include non-negative values for `await_count`, `resume_count`, and `cancel_by_user_count`

#### Scenario: Consumer inspects run diagnostics without clarification flow
- **WHEN** a run never triggers clarification
- **THEN** diagnostics expose zero-value clarification counters without breaking schema compatibility

### Requirement: Runtime config SHALL expose cancel-storm and backpressure baseline controls
Runtime configuration MUST expose concurrency baseline controls for cancellation storm and backpressure behavior with deterministic precedence `env > file > default`. Default backpressure mode MUST be `block`.

This requirement MUST NOT introduce a new public API surface; configuration behavior MUST remain library-first through existing runtime config manager entry points.

#### Scenario: Startup with default cancel-storm controls
- **WHEN** runtime starts without explicit cancel-storm/backpressure overrides
- **THEN** effective configuration uses documented defaults including backpressure mode `block`

#### Scenario: Startup with environment overrides for concurrency controls
- **WHEN** YAML and environment variables both define cancel-storm/backpressure fields
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Startup with invalid backpressure mode
- **WHEN** configuration provides unsupported backpressure mode value
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

### Requirement: Runtime diagnostics SHALL expose minimal cancel-storm and backpressure counters
Run diagnostics MUST expose the following minimum fields for concurrency-control observability: `cancel_propagated_count`, `backpressure_drop_count`, and `inflight_peak`.

`cancel_propagated_count` MUST be non-negative and count successful cancellation propagation actions.
`backpressure_drop_count` MUST be non-negative and MUST remain zero when active policy is `block`.
`inflight_peak` MUST be non-negative and represent the run-scoped peak in-flight work count.

#### Scenario: Consumer inspects diagnostics after canceled high-fanout run
- **WHEN** a run is canceled during high-fanout execution
- **THEN** diagnostics expose non-negative `cancel_propagated_count` and `inflight_peak`

#### Scenario: Consumer inspects diagnostics under default block policy
- **WHEN** active backpressure policy is `block`
- **THEN** diagnostics expose `backpressure_drop_count` as zero without breaking schema compatibility

#### Scenario: Consumer inspects diagnostics with no concurrency pressure
- **WHEN** run completes without cancellation and without backpressure pressure-hit
- **THEN** diagnostics still expose zero-valued baseline fields in a stable schema

### Requirement: Runtime performance baseline SHALL include p95 latency and goroutine peak gates
Quality validation for runtime concurrency baseline MUST include contract-level verification and benchmark/pressure checks for `p95 latency` and `goroutine peak`.

#### Scenario: Quality gate checks concurrency baseline
- **WHEN** maintainers run baseline quality checks for this capability
- **THEN** reported outputs include both `p95 latency` and `goroutine peak` signals for regression judgment

### Requirement: Runtime SHALL expose skill trigger scoring policy in YAML and environment overrides
Runtime configuration MUST expose skill trigger scoring policy fields through YAML and environment variables, and MUST resolve effective values with precedence `env > file > default`.

The policy MUST include at least:
- scorer strategy selector (default lexical weighted-keyword)
- confidence threshold
- tie-break mode (default `highest-priority`)
- low-confidence suppression toggle (default enabled)
- keyword/weight mapping inputs needed by default scorer

#### Scenario: Startup with file and env scoring overrides
- **WHEN** runtime starts with skill trigger scoring fields set in YAML and overlapping environment values
- **THEN** effective scoring policy follows `env > file > default`

#### Scenario: Startup with default scoring policy
- **WHEN** runtime starts without explicit skill trigger scoring overrides
- **THEN** effective policy uses lexical weighted-keyword scorer, tie-break `highest-priority`, and suppression enabled

### Requirement: Runtime SHALL fail fast on invalid skill trigger scoring configuration
Runtime MUST validate skill trigger scoring configuration during startup and hot reload; invalid enum values, out-of-range thresholds, or malformed weight entries MUST fail fast and block activation.

#### Scenario: Invalid tie-break mode
- **WHEN** configuration sets unsupported tie-break mode
- **THEN** runtime returns validation error and does not activate the snapshot

#### Scenario: Invalid confidence threshold range
- **WHEN** configuration sets confidence threshold outside supported range
- **THEN** runtime returns validation error and does not activate the snapshot

#### Scenario: Malformed keyword weight mapping
- **WHEN** configuration contains malformed or duplicate-conflicting keyword weights
- **THEN** runtime returns validation error and does not activate the snapshot

### Requirement: Runtime SHALL expose drop_low_priority policy rules and drop-set controls in config
Runtime configuration MUST expose policy fields for `drop_low_priority` behavior, including rule-based low-priority matching and configurable droppable-priority set controls.

#### Scenario: Startup with drop policy rules from file and env
- **WHEN** drop policy fields are set in YAML and overridden by environment variables
- **THEN** effective configuration resolves with precedence `env > file > default`

#### Scenario: Invalid drop policy config
- **WHEN** droppable-priority set or rule enum contains unsupported value
- **THEN** runtime fails fast and rejects startup/hot-reload activation

### Requirement: Runtime diagnostics SHALL expose drop_low_priority outcome semantics
Runtime diagnostics MUST expose backpressure drop outcomes with semantically consistent counters and reason mapping aligned to timeline events.

#### Scenario: Consumer inspects drop outcomes
- **WHEN** a run triggers low-priority drops under queue pressure
- **THEN** diagnostics include non-zero drop counters and timeline correlation with `backpressure.drop_low_priority`

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

### Requirement: Runtime SHALL expose cross-run timeline trend configuration with deterministic precedence
Runtime configuration MUST expose cross-run Action Timeline trend aggregation controls with precedence `env > file > default`.

The minimum configuration set MUST include:
- enable switch (default enabled),
- `last_n_runs` window size (default `100`),
- `time_window` duration (default `15m`).

Invalid values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default trend configuration
- **WHEN** runtime starts without trend-specific overrides
- **THEN** cross-run trend aggregation is enabled with `last_n_runs=100` and `time_window=15m`

#### Scenario: Startup with file and environment trend overrides
- **WHEN** trend fields are configured in both YAML and environment variables
- **THEN** effective trend settings resolve with `env > file > default`

#### Scenario: Invalid trend window configuration
- **WHEN** `last_n_runs` is non-positive or `time_window` is invalid
- **THEN** runtime rejects startup or hot reload snapshot with fail-fast validation error

### Requirement: Runtime diagnostics SHALL expose cross-run timeline trend aggregates
Runtime diagnostics API MUST expose cross-run timeline trend aggregates using both `last_n_runs` and `time_window` selection modes.

Trend aggregates MUST support `phase + status` dimensions and MUST include at least:
- `count_total`
- `failed_total`
- `canceled_total`
- `skipped_total`
- `latency_avg_ms`
- `latency_p95_ms`
- `window_start`
- `window_end`

The capability MUST be additive and MUST NOT break existing run-level diagnostics consumers.

#### Scenario: Consumer queries trends with last_n_runs mode
- **WHEN** application queries trend diagnostics using `last_n_runs`
- **THEN** runtime returns bounded `phase + status` aggregates over the most recent N runs with required metric fields

#### Scenario: Consumer queries trends with time_window mode
- **WHEN** application queries trend diagnostics using `time_window`
- **THEN** runtime returns bounded `phase + status` aggregates over runs inside the time window with required metric fields

#### Scenario: Consumer reads existing run-level diagnostics only
- **WHEN** existing integrations continue reading legacy run-level fields
- **THEN** diagnostics remain backward-compatible without requiring consumer changes

#### Scenario: Consumer queries trends for empty window
- **WHEN** selected window has no eligible run samples
- **THEN** runtime returns an empty aggregate set and does not fabricate metrics

### Requirement: Runtime SHALL expose CA2 external retriever observability config with deterministic precedence
Runtime configuration MUST expose CA2 external retriever observability settings with precedence `env > file > default`.

The minimum set MUST include:
- observability window duration (default `15m`),
- static threshold `p95_latency_ms`,
- static threshold `error_rate`,
- static threshold `hit_rate`.

Invalid window or threshold values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default CA2 external observability config
- **WHEN** runtime starts without explicit CA2 external observability overrides
- **THEN** effective config uses default window `15m` with valid threshold defaults

#### Scenario: Startup with env and file overrides
- **WHEN** observability window and thresholds are set in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid threshold or window config
- **WHEN** runtime receives out-of-range threshold values or non-positive window duration
- **THEN** startup or hot reload is rejected with fail-fast validation error

### Requirement: Runtime diagnostics SHALL expose provider-scoped CA2 external trend aggregates
Runtime diagnostics API MUST expose CA2 external retriever trend aggregates grouped by provider and window.

The minimum output fields MUST include:
- `provider`
- `window_start`
- `window_end`
- `p95_latency_ms`
- `error_rate`
- `hit_rate`

Trend outputs MUST be additive and MUST NOT break existing diagnostics consumers.

#### Scenario: Consumer queries CA2 external trends in default window
- **WHEN** application queries CA2 external trend diagnostics without explicit window override
- **THEN** runtime returns provider-scoped aggregates for default window `15m` with required fields

#### Scenario: Consumer queries CA2 external trends with custom window
- **WHEN** application queries CA2 external trend diagnostics with explicit window parameter
- **THEN** runtime returns provider-scoped aggregates for the requested window

#### Scenario: Consumer reads existing diagnostics only
- **WHEN** existing integration reads only legacy run-level fields
- **THEN** diagnostics remain backward-compatible without requiring consumer changes

### Requirement: Runtime diagnostics SHALL emit threshold-hit signals without automatic strategy actions
Runtime MUST evaluate CA2 external trend aggregates against static thresholds and emit normalized threshold-hit signals for observability and operator workflows.

Threshold-hit signals MUST NOT trigger automatic provider switching, routing changes, or policy mutation in this milestone.

#### Scenario: p95 latency threshold is exceeded
- **WHEN** provider trend `p95_latency_ms` exceeds configured threshold
- **THEN** diagnostics include threshold-hit signal for `p95_latency_ms` and runtime behavior remains unchanged

#### Scenario: error-rate threshold is exceeded
- **WHEN** provider trend `error_rate` exceeds configured threshold
- **THEN** diagnostics include threshold-hit signal for `error_rate` and runtime behavior remains unchanged

#### Scenario: hit-rate threshold is under target
- **WHEN** provider trend `hit_rate` is below configured threshold
- **THEN** diagnostics include threshold-hit signal for `hit_rate` and runtime behavior remains unchanged

### Requirement: Runtime SHALL expose CA3 compaction strategy config with deterministic precedence
Runtime configuration MUST expose CA3 compaction strategy fields with precedence `env > file > default`, including:
- `context_assembler.ca3.compaction.mode` (`truncate|semantic`)
- semantic compaction timeout controls
- evidence retention controls (keywords and recent-window constraints)

Invalid enum, timeout, or retention parameters MUST fail fast on startup and hot reload.

#### Scenario: Startup with default CA3 compaction config
- **WHEN** runtime starts without explicit CA3 compaction strategy overrides
- **THEN** effective config resolves to `truncate` mode and valid default retention controls

#### Scenario: Startup with semantic compaction overrides
- **WHEN** CA3 compaction strategy is set in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid CA3 compaction enum
- **WHEN** runtime config contains unsupported compaction mode
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime diagnostics SHALL expose minimal CA3 compaction observability fields
Run diagnostics MUST expose minimal CA3 compaction fields:
- `ca3_compaction_mode`
- `ca3_compaction_fallback`
- `ca3_compaction_retained_evidence_count`

These fields MUST be additive and backward-compatible for existing diagnostics consumers.

#### Scenario: Consumer inspects diagnostics with semantic compaction success
- **WHEN** a run completes with semantic compaction applied
- **THEN** diagnostics include mode=`semantic`, fallback=`false`, and non-negative retained evidence count

#### Scenario: Consumer inspects diagnostics with semantic fallback
- **WHEN** semantic compaction fails under best-effort and truncate fallback is used
- **THEN** diagnostics include mode=`semantic`, fallback=`true`, and non-negative retained evidence count

### Requirement: Runtime diagnostics contract SHALL preserve CA3 compaction semantics across Run and Stream
Diagnostics payload semantics for CA3 compaction MUST remain equivalent between Run and Stream for equivalent inputs, including mode selection, fallback marker, and retained evidence count.

#### Scenario: Equivalent Run and Stream semantic execution
- **WHEN** equivalent requests execute through Run and Stream with same CA3 config
- **THEN** emitted compaction diagnostics fields are semantically equivalent

#### Scenario: Equivalent Run and Stream fallback execution
- **WHEN** semantic compaction fails in both paths under best-effort
- **THEN** both paths emit semantically equivalent fallback diagnostics

### Requirement: Runtime config SHALL expose CA3 embedding scorer controls
Runtime config MUST expose CA3 embedding scorer controls including enablement flag, provider/model selector (OpenAI/Gemini/Anthropic), optional independent embedding credentials, timeout, cosine metric selector, and hybrid score weight fields with fail-fast validation.

#### Scenario: Startup with valid embedding scorer config
- **WHEN** runtime starts with valid CA3 embedding scorer configuration
- **THEN** effective config includes embedding scorer controls and CA3 can evaluate hybrid scoring path

#### Scenario: Hot reload with invalid embedding scorer config
- **WHEN** runtime receives invalid CA3 embedding scorer configuration update
- **THEN** runtime rejects update and preserves previous valid config snapshot

#### Scenario: Default embedding scorer config
- **WHEN** runtime loads default CA3 embedding scorer settings
- **THEN** effective defaults use cosine metric, `rule_weight=0.7`, `embedding_weight=0.3`, and shared quality threshold strategy

#### Scenario: Independent embedding credentials configured
- **WHEN** runtime config includes provider-specific embedding credentials
- **THEN** effective config uses independent embedding credentials for CA3 embedding calls

### Requirement: Diagnostics API SHALL include CA3 embedding scoring fields
Runtime diagnostics MUST include additive CA3 embedding scoring fields for adapter status, similarity contribution, and fallback reasons without breaking existing field semantics.

#### Scenario: Embedding scoring success
- **WHEN** CA3 completes embedding scoring successfully
- **THEN** diagnostics include embedding contribution fields and adapter status markers

#### Scenario: Embedding scoring fallback
- **WHEN** CA3 falls back from embedding scoring to rule-only path
- **THEN** diagnostics include explicit embedding fallback reason and fallback mode markers

#### Scenario: Provider path observability
- **WHEN** CA3 embedding scorer executes
- **THEN** diagnostics include which provider adapter path was selected for the scoring attempt

### Requirement: Runtime config SHALL expose CA3 reranker controls and threshold profile settings
Runtime config MUST expose CA3 reranker controls with deterministic precedence `env > file > default`, including:
- reranker enablement,
- reranker timeout and bounded retry policy,
- provider/model threshold profile map.

Invalid reranker or threshold profile configuration MUST fail fast at startup and hot reload.

#### Scenario: Startup with valid reranker config
- **WHEN** runtime starts with valid CA3 reranker controls and threshold profiles
- **THEN** effective config includes reranker settings and deterministic threshold precedence behavior

#### Scenario: Hot reload with invalid reranker profile
- **WHEN** config update includes malformed threshold profile or invalid timeout
- **THEN** runtime rejects update and preserves previous valid snapshot

#### Scenario: Reranker enabled without provider/model profile
- **WHEN** reranker is enabled and selected provider/model has no configured threshold profile
- **THEN** runtime fails fast with missing-profile validation error

### Requirement: Runtime diagnostics SHALL expose provider/model-scoped CA3 reranker quality fields
Runtime diagnostics MUST expose additive CA3 reranker fields sufficient for tuning and incident triage, including:
- reranker enabled/used marker,
- provider/model identity,
- threshold source,
- threshold-hit status,
- reranker fallback reason.

These fields MUST NOT break existing diagnostics consumers.

#### Scenario: Reranker path succeeds
- **WHEN** CA3 reranker executes successfully
- **THEN** diagnostics include reranker usage marker, provider/model identity, and threshold source

#### Scenario: Reranker path falls back
- **WHEN** reranker is bypassed or fails under `best_effort`
- **THEN** diagnostics include fallback reason and effective decision path marker

#### Scenario: Existing consumer reads legacy fields only
- **WHEN** diagnostics consumer does not parse new reranker fields
- **THEN** existing diagnostics semantics remain backward-compatible

### Requirement: Runtime SHALL expose threshold tuning toolkit integration contract
Runtime-adjacent tooling contract MUST define stable input/output schema for CA3 threshold tuning toolkit, including corpus metadata fields and recommendation artifact schema versioning.

#### Scenario: Toolkit runs with supported schema version
- **WHEN** tuning toolkit receives input matching supported schema version
- **THEN** toolkit produces recommendation artifacts with declared output schema version

#### Scenario: Toolkit receives unsupported schema version
- **WHEN** tuning toolkit input schema version is unsupported
- **THEN** toolkit fails fast with explicit schema-version error and no partial output

#### Scenario: Toolkit minimal output mode
- **WHEN** tuning toolkit run succeeds in configured minimal mode
- **THEN** output contract requires markdown artifact and does not require JSON artifact

#### Scenario: Corpus readiness guidance reported
- **WHEN** tuning toolkit evaluates a corpus for selected provider+model segment
- **THEN** output includes corpus readiness and confidence guidance fields without enforcing fixed hard-gate constants

### Requirement: Runtime SHALL expose reranker extension registration contract
Runtime MUST expose a stable extension registration contract for provider-specific reranker implementations.

The contract MUST preserve existing fail-fast and best-effort policy semantics regardless of built-in or custom implementation path.

#### Scenario: Valid custom reranker registration
- **WHEN** application registers a valid provider-specific reranker implementation
- **THEN** runtime accepts registration and executes custom implementation for matching provider/model

#### Scenario: Invalid custom reranker registration
- **WHEN** application registers incompatible reranker implementation
- **THEN** runtime rejects registration with explicit validation error and preserves built-in path

### Requirement: Runtime config SHALL expose CA3 threshold governance rollout controls
Runtime config MUST expose CA3 threshold governance controls with deterministic precedence `env > file > default`, including governance mode (`enforce|dry_run`), profile version identifier, and provider:model-scoped rollout match settings.

#### Scenario: Startup with valid CA3 governance config
- **WHEN** runtime starts with valid CA3 governance mode and provider:model rollout settings
- **THEN** effective config includes resolved governance fields and CA3 can evaluate rollout matching deterministically

#### Scenario: Invalid CA3 governance mode value
- **WHEN** runtime loads CA3 governance config with unsupported mode value
- **THEN** startup or hot reload fails fast with a validation error

### Requirement: Runtime diagnostics SHALL expose additive CA3 threshold governance fields
Runtime diagnostics MUST expose additive CA3 threshold governance observability fields sufficient for rollout triage, including profile version, rollout-match hit, threshold-source, threshold-hit, and fallback reason, without changing existing field semantics.

#### Scenario: Governance-enabled CA3 enforcement run
- **WHEN** CA3 executes with governance mode `enforce` and rollout match hits selected provider:model
- **THEN** diagnostics include additive governance fields for profile version, rollout hit, and threshold evaluation outcome

#### Scenario: Governance fallback path in best-effort mode
- **WHEN** governance evaluation fails under `best_effort`
- **THEN** diagnostics include governance fallback reason while preserving existing reranker/compaction fields

### Requirement: Runtime SHALL expose CA2 external capability-hint and template-pack config with deterministic precedence
Runtime configuration MUST expose CA2 external retriever capability-hint and template-pack fields with precedence `env > file > default`.

The minimum template-pack profile set for this milestone MUST include:
- `graphrag_like`
- `ragflow_like`
- `elasticsearch_like`

Runtime MUST support deterministic resolution semantics aligned with Stage2 execution:
- profile defaults are resolved first,
- explicit mapping/auth/header fields override profile defaults,
- explicit-only mode is valid when no template profile is selected.

Invalid template-pack profile values or malformed hint structure MUST fail fast during startup and hot reload.

#### Scenario: Startup with template-pack profile override
- **WHEN** CA2 template-pack profile is configured in both YAML and environment variables
- **THEN** effective profile resolves by `env > file > default` and participates in deterministic template resolution

#### Scenario: Startup with explicit-only external mapping
- **WHEN** runtime starts with no template-pack profile and valid explicit external mapping fields
- **THEN** runtime accepts configuration and Stage2 can execute explicit-only mapping path

#### Scenario: Invalid template-pack profile value
- **WHEN** runtime receives unsupported template-pack profile value
- **THEN** startup or hot reload is rejected with fail-fast validation error

#### Scenario: Malformed capability-hint config
- **WHEN** runtime receives malformed capability-hint structure
- **THEN** startup or hot reload is rejected with fail-fast validation error

### Requirement: Runtime diagnostics SHALL expose additive CA2 hint and template resolution fields
Runtime diagnostics MUST expose additive CA2 Stage2 fields for hint/template observability without breaking existing consumer semantics.

The minimum additive fields MUST include:
- `stage2_template_profile`
- `stage2_template_resolution_source`
- `stage2_hint_applied`
- `stage2_hint_mismatch_reason`

Hint mismatch and template anomalies MUST remain observational only and MUST NOT imply automatic strategy actions.

#### Scenario: Consumer inspects successful hint and template resolution
- **WHEN** Stage2 executes with valid template profile and applied capability hints
- **THEN** diagnostics include resolved template profile, resolution source, and hint-applied marker

#### Scenario: Consumer inspects hint mismatch
- **WHEN** Stage2 executes with unsupported or invalid capability hints
- **THEN** diagnostics include normalized `stage2_hint_mismatch_reason` while preserving existing stage-policy outcomes

#### Scenario: Existing diagnostics consumer reads legacy fields only
- **WHEN** consumer parses only pre-existing CA2 diagnostic fields
- **THEN** diagnostics remain backward-compatible and existing field semantics are unchanged

### Requirement: Runtime SHALL preserve CA2 layered error semantics while extending hint and template fields
Runtime diagnostics and event mappings for CA2 Stage2 MUST preserve baseline layered error semantics (`transport`, `protocol`, `semantic`) and MAY include forward-compatible enum extension values.

Hint/template-related diagnostics MUST be additive extensions and MUST NOT redefine baseline error-layer meanings.

#### Scenario: Baseline error layer with template profile
- **WHEN** Stage2 retrieval fails with baseline layered error under active template profile
- **THEN** diagnostics preserve baseline error-layer semantics and include additive template context fields

#### Scenario: Extended error layer value with hint mismatch
- **WHEN** implementation emits an extended layer enum value while Stage2 also records hint mismatch
- **THEN** diagnostics preserve extended value and additive mismatch fields without schema conflict

### Requirement: Repository SHALL protect CA2 hint-template contract with contract tests and benchmark baseline
The repository MUST include contract tests that validate CA2 hint/template configuration, deterministic resolution behavior, observational-only mismatch semantics, and Run/Stream semantic equivalence.

The repository MUST additionally include benchmark coverage for hint/template resolution overhead and maintain compatibility with existing CA2 trend benchmark baselines.

#### Scenario: Contract test suite for hint and template semantics is executed
- **WHEN** CI or local validation runs CA2 hint/template contract tests
- **THEN** semantic mismatches in precedence, mismatch policy, or Run/Stream equivalence fail the suite

#### Scenario: Benchmark suite for hint and template resolution is executed
- **WHEN** CI or local performance validation runs CA2 benchmarks
- **THEN** benchmark outputs include hint/template resolution baseline and remain comparable with existing CA2 trend baselines

### Requirement: Runtime config SHALL expose S2 tool-security governance controls with deterministic precedence
Runtime configuration MUST expose S2 tool-security governance fields with precedence `env > file > default`, including:
- governance mode (default `enforce`),
- `namespace+tool` permission policy entries,
- process-scoped rate-limit policy entries,
- deny behavior controls for permission and rate-limit violations.

Invalid policy keys, malformed `namespace+tool` selectors, or unsupported mode values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default S2 governance config
- **WHEN** runtime starts without explicit S2 governance overrides
- **THEN** effective config resolves governance mode as `enforce` with valid default deny semantics

#### Scenario: Startup with env and file governance overrides
- **WHEN** S2 governance fields are defined in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid namespace+tool policy key
- **WHEN** runtime receives malformed `namespace+tool` selector in permission or rate-limit policy
- **THEN** startup or hot reload is rejected with fail-fast validation error

### Requirement: Runtime config SHALL expose model I/O security filtering controls with deterministic precedence
Runtime configuration MUST expose model input/output security filtering controls with precedence `env > file > default`, including filter enablement, stage-specific execution controls, and extension registration settings.

Invalid stage configuration or malformed filter policy values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default model I/O filtering config
- **WHEN** runtime starts without explicit I/O filtering overrides
- **THEN** effective config resolves to valid default input/output filtering controls

#### Scenario: Startup with I/O filtering overrides
- **WHEN** input/output filtering settings are defined in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid I/O filter stage configuration
- **WHEN** runtime receives unsupported stage option or malformed filter policy value
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime SHALL apply S2 security config hot reload with atomic switch and rollback safety
Security governance and I/O filtering config updates MUST be validated and atomically activated on success; invalid updates MUST be rejected and previous snapshot MUST remain active.

#### Scenario: Valid S2 security config update arrives
- **WHEN** watched config file changes to valid S2 governance or I/O filtering settings
- **THEN** runtime atomically switches to new snapshot and subsequent requests observe updated security policy immediately

#### Scenario: Invalid S2 security config update arrives
- **WHEN** watched config file changes to invalid S2 governance or I/O filtering settings
- **THEN** runtime rejects the update, preserves previous active snapshot, and emits reload error diagnostics

### Requirement: Runtime diagnostics SHALL expose additive S2 security governance and I/O filtering fields
Runtime diagnostics MUST expose additive fields for S2 security decisions, including at minimum:
- policy kind (`permission|rate_limit|io_filter`),
- selector context (`namespace+tool` when applicable),
- filter stage (`input|output` when applicable),
- deny/match outcome,
- normalized reason code.

These fields MUST be backward-compatible and MUST NOT change existing diagnostics field semantics.

#### Scenario: Permission denial diagnostics
- **WHEN** runtime denies tool execution by permission policy
- **THEN** diagnostics include additive permission-deny fields with selector and reason code

#### Scenario: Rate-limit denial diagnostics
- **WHEN** runtime denies tool execution by process-scoped rate limit
- **THEN** diagnostics include additive rate-limit fields with selector, window context, and reason code

#### Scenario: I/O filter diagnostics
- **WHEN** runtime evaluates input or output filtering and records match or deny outcomes
- **THEN** diagnostics include additive filter stage/result fields with normalized reason code

### Requirement: Runtime config SHALL expose S3 security-event and alert callback controls with deterministic precedence
Runtime configuration MUST expose S3 security-event controls with precedence `env > file > default`, including event enablement, deny-alert trigger policy, severity mapping controls, and callback registration constraints.

Invalid S3 event config values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default S3 event config
- **WHEN** runtime starts without explicit S3 security-event overrides
- **THEN** effective config resolves valid defaults and deny-alert policy

#### Scenario: Invalid S3 event config update arrives
- **WHEN** watched config changes to malformed S3 event settings
- **THEN** runtime rejects update and preserves previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive S3 security-event fields
Runtime diagnostics MUST expose additive S3 event fields at minimum:
- `policy_kind`,
- `namespace_tool`,
- `filter_stage`,
- `decision`,
- `reason_code`,
- `severity`,
- alert-dispatch status marker.

These fields MUST remain backward-compatible with existing consumers.

#### Scenario: Consumer inspects deny alert diagnostics
- **WHEN** runtime dispatches a deny alert callback
- **THEN** diagnostics include S3 taxonomy fields and alert-dispatch status

#### Scenario: Consumer inspects callback failure diagnostics
- **WHEN** callback dispatch fails
- **THEN** diagnostics include failure marker and normalized failure reason without changing deny decision outcome

### Requirement: Run and Stream SHALL preserve S3 diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent S3 diagnostics payload fields.

#### Scenario: Equivalent S3 diagnostics in Run and Stream
- **WHEN** equivalent deny decisions occur in Run and Stream
- **THEN** diagnostics include equivalent S3 taxonomy and alert-dispatch semantics

### Requirement: Runtime config SHALL expose S4 delivery controls with deterministic precedence and fail-fast validation
Runtime configuration MUST expose S4 delivery controls under `security.security_event.delivery` with precedence `env > file > default`.
At minimum, configuration MUST include delivery mode, queue bounds/overflow policy, timeout, retry settings, and circuit breaker controls.
Invalid delivery enum or malformed numeric threshold values MUST fail fast during startup and hot reload.

#### Scenario: Startup resolves S4 delivery defaults
- **WHEN** runtime starts without explicit delivery overrides
- **THEN** effective config resolves valid defaults including `mode=async`, bounded queue, retry budget, and circuit breaker baseline

#### Scenario: Invalid S4 delivery hot-reload update is rejected
- **WHEN** runtime receives malformed delivery config during hot reload
- **THEN** runtime rejects update, records reload failure diagnostics, and keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive S4 delivery observability fields
Runtime diagnostics MUST expose additive delivery fields for security alerts, including at minimum delivery mode, retry count, queue-drop marker/count, circuit state, and delivery failure reason.
These fields MUST remain backward-compatible with existing diagnostics consumers.

#### Scenario: Consumer inspects retry and circuit diagnostics
- **WHEN** deny alert delivery experiences retries or circuit transitions
- **THEN** run diagnostics include normalized retry and circuit state markers

#### Scenario: Consumer inspects queue overflow diagnostics
- **WHEN** deny alerts exceed bounded queue capacity under async mode
- **THEN** diagnostics include queue overflow/drop markers with configured overflow-policy semantics

### Requirement: Run and Stream SHALL preserve S4 diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent S4 delivery diagnostics fields.

#### Scenario: Equivalent S4 diagnostics in Run and Stream
- **WHEN** equivalent deny alerts are produced in Run and Stream
- **THEN** delivery-mode, retry, queue-drop, and circuit-state diagnostics are semantically equivalent

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

### Requirement: Runtime config SHALL expose skill embedding trigger scoring controls with deterministic precedence
Runtime configuration MUST expose embedding-enhanced skill trigger scoring controls under `skill.trigger_scoring.embedding.*` with precedence `env > file > default`.

At minimum, runtime MUST support:
- embedding scorer enablement and strategy activation controls,
- embedding timeout control,
- similarity metric selector for this milestone,
- lexical/embedding linear fusion weights.

For this milestone, configuration is managed through runtime JSON/YAML path only and MUST NOT require additional CLI parameters.

#### Scenario: Startup with skill embedding scoring overrides
- **WHEN** runtime starts with skill embedding scoring controls defined in both YAML and environment variables
- **THEN** effective skill embedding controls resolve by `env > file > default`

#### Scenario: Invalid skill embedding timeout
- **WHEN** runtime configuration sets non-positive skill embedding timeout
- **THEN** startup or hot reload fails fast with validation error

#### Scenario: Invalid skill embedding fusion weights
- **WHEN** runtime configuration sets invalid lexical/embedding fusion weights
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime diagnostics SHALL expose additive skill trigger scoring observability fields
Runtime diagnostics MUST expose additive skill-trigger observability fields sufficient for lexical-plus-embedding triage, including at minimum:
- active trigger scoring strategy,
- final trigger score,
- embedding score contribution (when available),
- embedding fallback reason (when fallback occurs).

These fields MUST be backward-compatible and MUST NOT redefine existing skill lifecycle diagnostics semantics.

#### Scenario: Consumer inspects successful lexical-plus-embedding trigger
- **WHEN** application queries skill diagnostics for runs using embedding-enhanced trigger scoring
- **THEN** diagnostics include strategy, final score, and embedding score contribution fields

#### Scenario: Consumer inspects embedding fallback
- **WHEN** application queries skill diagnostics for runs where embedding path falls back to lexical
- **THEN** diagnostics include normalized fallback reason while preserving existing lifecycle fields

### Requirement: Run and Stream SHALL preserve skill trigger scoring diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent skill trigger scoring diagnostics fields.

#### Scenario: Equivalent skill trigger diagnostics in Run and Stream
- **WHEN** equivalent requests execute under the same skill trigger scoring configuration and scorer behavior
- **THEN** diagnostics fields for strategy, final score, and fallback class are semantically equivalent across Run and Stream

### Requirement: Runtime SHALL expose skill lexical tokenizer mode and semantic candidate budget with deterministic precedence
Runtime configuration MUST expose multilingual lexical and semantic budget controls under skill trigger scoring configuration with precedence `env > file > default`.

At minimum, runtime MUST support:
- `skill.trigger_scoring.lexical.tokenizer_mode`
- `skill.trigger_scoring.max_semantic_candidates`

For this milestone, configuration MUST be managed through JSON/YAML path (with env overrides) and MUST NOT require additional CLI parameters.

#### Scenario: Environment overrides file for tokenizer and budget controls
- **WHEN** both YAML and environment variables define tokenizer mode and semantic candidate budget
- **THEN** effective runtime config resolves tokenizer and budget by `env > file > default`

#### Scenario: Startup uses default tokenizer and budget values
- **WHEN** runtime starts without explicit tokenizer mode or semantic budget configuration
- **THEN** effective config uses `tokenizer_mode=mixed_cjk_en` and `max_semantic_candidates=3`

### Requirement: Runtime SHALL fail fast on invalid skill lexical-budget configuration
Runtime startup and hot reload MUST validate skill lexical-budget controls before activation.

Validation MUST reject:
- unsupported `tokenizer_mode` values,
- non-positive `max_semantic_candidates`.

Invalid updates MUST NOT replace active configuration snapshot.

#### Scenario: Invalid tokenizer mode fails startup
- **WHEN** runtime configuration sets unsupported tokenizer mode
- **THEN** startup fails fast with validation error

#### Scenario: Invalid semantic budget fails hot reload and rolls back
- **WHEN** hot reload applies `max_semantic_candidates <= 0`
- **THEN** reload is rejected and runtime keeps previous valid configuration

### Requirement: Runtime diagnostics SHALL expose additive lexical-budget observability fields
Runtime diagnostics MUST include additive skill trigger fields:
- `tokenizer_mode`
- `candidate_pruned_count`

These fields MUST remain backward-compatible and MUST NOT alter existing skill lifecycle diagnostics semantics.

#### Scenario: Diagnostics include tokenizer mode and pruning count fields
- **WHEN** application queries skill diagnostics after compile evaluation
- **THEN** diagnostics payload includes `tokenizer_mode` and `candidate_pruned_count`

#### Scenario: Legacy consumers remain compatible with additive diagnostics fields
- **WHEN** existing diagnostics consumers read skill lifecycle records without parsing new fields
- **THEN** original lifecycle semantics remain unchanged

### Requirement: Run and Stream SHALL preserve lexical-budget diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent lexical-budget diagnostics fields.

#### Scenario: Equivalent lexical-budget diagnostics in Run and Stream
- **WHEN** equivalent requests execute with same tokenizer mode and semantic budget
- **THEN** diagnostics for `tokenizer_mode` and `candidate_pruned_count` are semantically equivalent across Run and Stream

