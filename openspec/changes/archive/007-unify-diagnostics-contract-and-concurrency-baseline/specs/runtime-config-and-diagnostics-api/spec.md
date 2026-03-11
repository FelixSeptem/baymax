## MODIFIED Requirements

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support. Diagnostics returned by these APIs MUST follow single-writer and idempotent persistence semantics so repeated event submission does not alter logical aggregate counts.

#### Scenario: Consumer requests recent run diagnostics
- **WHEN** application calls diagnostics API for recent runs
- **THEN** runtime returns bounded summary records with normalized fields and without duplicated logical run entries caused by retries or replay

#### Scenario: Consumer requests effective configuration
- **WHEN** application calls API to fetch effective configuration
- **THEN** runtime returns a sanitized snapshot that masks secret-like fields

### Requirement: Runtime SHALL be concurrency-safe for config and diagnostics access
Configuration reads, diagnostics writes, diagnostics deduplication, and hot-reload swaps MUST be safe under concurrent goroutines.

#### Scenario: Concurrent reads during hot reload
- **WHEN** multiple goroutines read configuration while a reload is in progress
- **THEN** each read observes either old or new complete snapshot, never mixed fields

#### Scenario: Concurrent diagnostics recording and querying
- **WHEN** goroutines concurrently record call summaries and query diagnostics
- **THEN** runtime preserves data integrity, idempotent write behavior, and bounded-memory behavior

## ADDED Requirements

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