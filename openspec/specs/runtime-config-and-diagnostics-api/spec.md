# runtime-config-and-diagnostics-api Specification

## Purpose
TBD - created by archiving change add-runtime-config-and-diagnostics-api-with-hot-reload. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`. This configuration capability MUST be owned by a global runtime module and be consumable by MCP, runner, local tool, skill loader, and observability components through stable interfaces.

For provider capability fallback, runtime configuration MUST additionally support validated fallback policy fields (including ordered provider candidates and discovery/cache controls) under the same precedence and validation pipeline.

Runtime configuration MUST additionally support context assembler CA1 baseline fields with deterministic precedence, including `context_assembler.enabled` (default true), file journal path, prefix version, guard fail-fast toggle, and storage backend selector (file active, db placeholder).

Runtime configuration MUST additionally support context assembler CA2 fields with deterministic precedence, including staged assembly enablement, stage routing mode, stage-level failure policy, stage timeouts, stage2 provider selector (file active; rag/db placeholders), routing threshold controls, and tail recap options.

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

