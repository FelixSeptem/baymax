## MODIFIED Requirements

### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`. This configuration capability MUST be owned by a global runtime module and be consumable by MCP, runner, local tool, skill loader, and observability components through stable interfaces.

For provider capability fallback, runtime configuration MUST additionally support validated fallback policy fields (including ordered provider candidates and discovery/cache controls) under the same precedence and validation pipeline.

Runtime configuration MUST additionally support context assembler CA1 baseline fields with deterministic precedence, including `context_assembler.enabled` (default true), file journal path, prefix version, guard fail-fast toggle, and storage backend selector (file active, db placeholder).

Runtime configuration MUST additionally support context assembler CA2 fields with deterministic precedence, including staged assembly enablement, stage routing mode, stage-level failure policy, stage timeouts, stage2 provider selector (file active; rag/db placeholders), routing threshold controls, and tail recap options.

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

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support. Diagnostics returned by these APIs MUST follow single-writer and idempotent persistence semantics so repeated event submission does not alter logical aggregate counts.

Diagnostics MUST include capability-preflight and provider-fallback summary fields for each affected model step, including requested capability set, candidate providers considered, selected provider, and fail-fast reason when chain is exhausted.

Diagnostics MUST additionally include context assembler CA1 baseline fields for each assemble cycle and related run summary context, including `prefix_hash`, `assemble_latency_ms`, `assemble_status`, and `guard_violation`.

Diagnostics MUST additionally include context assembler CA2 stage and recap fields, including normalized stage statuses, stage2 skip reason, stage latencies, and recap status.

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

## ADDED Requirements

### Requirement: Runtime SHALL validate CA2 stage provider and routing mode constraints
The runtime MUST validate CA2 stage provider and routing mode enums at startup and hot reload. Unsupported modes or provider selections in current milestone MUST fail fast with explicit classification.

#### Scenario: Unsupported CA2 routing mode
- **WHEN** runtime configuration sets unknown CA2 routing mode
- **THEN** initialization fails fast with validation error

#### Scenario: CA2 rag/db provider requested before implementation
- **WHEN** runtime configuration selects Stage2 provider as rag or db in CA2 milestone
- **THEN** runtime returns explicit provider-not-ready classification and does not partially activate staged assembly
