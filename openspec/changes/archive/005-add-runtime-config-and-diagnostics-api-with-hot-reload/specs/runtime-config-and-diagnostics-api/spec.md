## ADDED Requirements

### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`.

#### Scenario: Startup with file and environment overrides
- **WHEN** runtime starts with a YAML config file and overlapping environment variables
- **THEN** effective configuration uses environment values first, then file values, then defaults for unset keys

#### Scenario: Startup without config file
- **WHEN** runtime starts without a config file
- **THEN** runtime uses default values and applicable environment overrides

### Requirement: Runtime SHALL validate configuration and fail fast on invalid startup input
The runtime MUST validate required fields, numeric ranges, and enum values before activation; invalid startup configuration MUST return an error and abort initialization.

#### Scenario: Invalid enum value at startup
- **WHEN** configuration provides an unsupported enum value
- **THEN** runtime returns validation error and does not start

#### Scenario: Invalid numeric range at startup
- **WHEN** configuration contains out-of-range numeric values
- **THEN** runtime returns validation error and does not start

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support.

#### Scenario: Consumer requests recent run diagnostics
- **WHEN** application calls diagnostics API for recent runs
- **THEN** runtime returns bounded summary records with normalized fields

#### Scenario: Consumer requests effective configuration
- **WHEN** application calls API to fetch effective configuration
- **THEN** runtime returns a sanitized snapshot that masks secret-like fields

### Requirement: Runtime SHALL support hot reload with atomic swap and rollback safety
The runtime MUST watch config file changes, rebuild and validate a new snapshot, and atomically replace active configuration only on successful validation.

#### Scenario: Valid configuration update arrives
- **WHEN** watched YAML file changes to a valid configuration
- **THEN** runtime atomically switches to the new snapshot without exposing partial state

#### Scenario: Invalid configuration update arrives
- **WHEN** watched YAML file changes to an invalid configuration
- **THEN** runtime rejects the update, keeps current active snapshot unchanged, and emits reload error diagnostics

### Requirement: Runtime SHALL be concurrency-safe for config and diagnostics access
Configuration reads, diagnostics writes, and hot-reload swaps MUST be safe under concurrent goroutines.

#### Scenario: Concurrent reads during hot reload
- **WHEN** multiple goroutines read configuration while a reload is in progress
- **THEN** each read observes either old or new complete snapshot, never mixed fields

#### Scenario: Concurrent diagnostics recording and querying
- **WHEN** goroutines concurrently record call summaries and query diagnostics
- **THEN** runtime preserves data integrity and bounded-memory behavior
