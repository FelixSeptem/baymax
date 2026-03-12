## MODIFIED Requirements

### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`. This configuration capability MUST be owned by a global runtime module and be consumable by MCP, runner, local tool, skill loader, and observability components through stable interfaces.

For provider capability fallback, runtime configuration MUST additionally support validated fallback policy fields (including ordered provider candidates and discovery/cache controls) under the same precedence and validation pipeline.

#### Scenario: Startup with file and environment overrides
- **WHEN** runtime starts with a YAML config file and overlapping environment variables
- **THEN** effective configuration uses environment values first, then file values, then defaults for unset keys

#### Scenario: Startup without config file
- **WHEN** runtime starts without a config file
- **THEN** runtime uses default values and applicable environment overrides

#### Scenario: Startup with fallback policy overrides
- **WHEN** runtime starts with fallback policy defined in both YAML and environment variables
- **THEN** effective fallback policy resolves using `env > file > default` and is available to model-step provider selection

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support. Diagnostics returned by these APIs MUST follow single-writer and idempotent persistence semantics so repeated event submission does not alter logical aggregate counts.

Diagnostics MUST include capability-preflight and provider-fallback summary fields for each affected model step, including requested capability set, candidate providers considered, selected provider, and fail-fast reason when chain is exhausted.

#### Scenario: Consumer requests recent run diagnostics
- **WHEN** application calls diagnostics API for recent runs
- **THEN** runtime returns bounded summary records with normalized fields and without duplicated logical run entries caused by retries or replay

#### Scenario: Consumer requests effective configuration
- **WHEN** application calls API to fetch effective configuration
- **THEN** runtime returns a sanitized snapshot that masks secret-like fields

#### Scenario: Consumer inspects fallback diagnostics
- **WHEN** application queries diagnostics for a run that triggered provider fallback
- **THEN** runtime returns normalized fallback summary fields sufficient to reconstruct capability decision path

## ADDED Requirements

### Requirement: Runtime SHALL validate provider fallback policy and discovery controls
The runtime MUST validate fallback policy configuration at startup and hot reload, including non-empty candidate constraints (when fallback is enabled), enum/range validation for discovery controls, and deterministic ordering guarantees.

#### Scenario: Invalid fallback policy at startup
- **WHEN** fallback configuration includes invalid provider identifiers or malformed ordering
- **THEN** runtime initialization fails fast with validation error

#### Scenario: Invalid fallback policy during hot reload
- **WHEN** watched configuration updates fallback policy to an invalid value
- **THEN** runtime rejects the update and preserves previous active snapshot
