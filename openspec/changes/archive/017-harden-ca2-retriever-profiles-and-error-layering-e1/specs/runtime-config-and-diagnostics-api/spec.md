## MODIFIED Requirements

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

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support. Diagnostics returned by these APIs MUST follow single-writer and idempotent persistence semantics so repeated event submission does not alter logical aggregate counts.

Diagnostics MUST include capability-preflight and provider-fallback summary fields for each affected model step, including requested capability set, candidate providers considered, selected provider, and fail-fast reason when chain is exhausted.

Diagnostics MUST additionally include context assembler CA1 baseline fields for each assemble cycle and related run summary context, including `prefix_hash`, `assemble_latency_ms`, `assemble_status`, and `guard_violation`.

Diagnostics MUST additionally include context assembler CA2 stage and recap fields, including normalized stage statuses, stage2 skip reason, stage latencies, and recap status.

Diagnostics and event payloads MUST additionally apply unified S1 redaction policy before persistence and emission.

Diagnostics for CA2 retrieval MUST additionally expose normalized Stage2 retrieval summary fields: `stage2_hit_count`, `stage2_source`, `stage2_reason`, `stage2_reason_code`, `stage2_error_layer`, and `stage2_profile`.

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

## ADDED Requirements

### Requirement: Runtime SHALL expose external retriever precheck API for library integrations
The runtime MUST provide a library-level precheck API for CA2 external retriever configuration. The API MUST return normalized findings that include severity (`warning` or `error`) and machine-readable reason codes.

`warning` findings MUST allow execution to continue, and `error` findings MUST require fail-fast behavior when used in startup or hot reload validation paths.

#### Scenario: Precheck returns warning findings only
- **WHEN** application runs precheck and receives only warning findings
- **THEN** runtime allows startup/hot reload to continue and exposes warnings for observability

#### Scenario: Precheck returns error finding
- **WHEN** application runs precheck and receives at least one error finding
- **THEN** runtime blocks startup/hot reload activation with explicit fail-fast validation error
