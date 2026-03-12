# context-assembler-baseline Specification

## Purpose
TBD - created by archiving change build-context-assembler-ca1-prefix-append-only-baseline. Update Purpose after archive.
## Requirements
### Requirement: Runner SHALL execute context assembler at pre-model stage
The runner MUST invoke context assembler before each model invocation in both Run and Stream flows. This pre-model hook MUST produce deterministic context payloads without changing existing external run/stream semantic contracts.

#### Scenario: Run path calls model after assembler
- **WHEN** runner executes a Run request with context assembler enabled
- **THEN** context assembler executes before model invocation and the model receives assembled context payload

#### Scenario: Stream path preserves semantic parity
- **WHEN** runner executes a Stream request with context assembler enabled
- **THEN** context assembler executes before stream model invocation and streaming event semantics remain unchanged

### Requirement: Context assembler SHALL enforce immutable prefix consistency
Context assembler MUST build a stable prefix from configured immutable blocks and MUST compute a `prefix_hash` for each assemble cycle. If the prefix changes under the same prefix version/session, guardrails MUST fail fast.

#### Scenario: Prefix is unchanged within same session version
- **WHEN** two consecutive assemble cycles run with identical immutable inputs and same prefix version
- **THEN** computed `prefix_hash` remains identical

#### Scenario: Prefix drift is detected
- **WHEN** assembler detects prefix bytes changed unexpectedly under same prefix version/session
- **THEN** assembler fails fast and blocks model invocation

### Requirement: Context journal SHALL be append-only and file-backed in CA1
Context assembler MUST persist journal entries as append-only JSONL records in local filesystem storage. CA1 MUST NOT support in-place mutation or mid-log reordering.

#### Scenario: Assembler writes intent and commit events
- **WHEN** assembler performs one successful context build
- **THEN** journal appends ordered intent/commit records to file without modifying prior entries

#### Scenario: Concurrent writes are attempted
- **WHEN** multiple goroutines attempt journal writes for same session
- **THEN** persisted log order remains append-only and data integrity is preserved

### Requirement: Guardrails SHALL be independent from LLM and fail fast by default
Assembler guardrails MUST run before model invocation and MUST rely on deterministic rules (hash/schema/sanitization) rather than model output. Default behavior MUST terminate current step on hard guard violations.

#### Scenario: Guard detects schema violation
- **WHEN** assembled payload violates required schema constraints
- **THEN** assembler aborts model invocation with fail-fast error

#### Scenario: Sensitive field is detected
- **WHEN** sensitive key patterns appear in assembled context
- **THEN** assembler applies sanitization policy before model invocation and records guard result

### Requirement: Context storage SHALL expose DB extension interface without DB implementation in CA1
The context storage layer MUST define a backend interface that allows file and db backends, while CA1 MUST implement file backend only and provide db as non-active placeholder.

#### Scenario: Runtime uses default storage backend
- **WHEN** context assembler starts with default config
- **THEN** file backend is selected and db backend remains unimplemented placeholder

#### Scenario: DB backend is configured in CA1
- **WHEN** runtime config requests db backend in CA1
- **THEN** runtime returns explicit unsupported/backend-not-ready error

### Requirement: Assembler SHALL expose minimal diagnostics baseline
Assembler MUST emit baseline diagnostics fields `prefix_hash`, `assemble_latency_ms`, `assemble_status`, and `guard_violation` for each assemble cycle.

#### Scenario: Successful assemble cycle
- **WHEN** assembler completes successfully
- **THEN** diagnostics include prefix hash, latency, and success status

#### Scenario: Guard-failed assemble cycle
- **WHEN** assembler fails due to guard violation
- **THEN** diagnostics include failure status and guard violation summary

