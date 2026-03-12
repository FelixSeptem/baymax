# context-assembler-baseline Specification

## Purpose
TBD - created by archiving change build-context-assembler-ca1-prefix-append-only-baseline. Update Purpose after archive.
## Requirements
### Requirement: Runner SHALL execute context assembler at pre-model stage
The runner MUST invoke context assembler before each model invocation in both Run and Stream flows. This pre-model hook MUST produce deterministic context payloads without changing existing external run/stream semantic contracts. Starting in CA2, this hook MUST support two-stage assembly execution (Stage1 then optional Stage2) while preserving the same external event ordering and complete-only tool-call contract.

#### Scenario: Run path calls model after staged assembler
- **WHEN** runner executes a Run request with context assembler enabled and CA2 staging enabled
- **THEN** context assembler executes Stage1 and conditional Stage2 before model invocation, and the model receives staged assembled payload

#### Scenario: Stream path preserves semantic parity with staged assembler
- **WHEN** runner executes a Stream request with context assembler enabled and CA2 staging enabled
- **THEN** staged assembler executes before stream model invocation and streaming event semantics remain unchanged

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
Assembler MUST emit baseline diagnostics fields `prefix_hash`, `assemble_latency_ms`, `assemble_status`, and `guard_violation` for each assemble cycle. Starting in CA2, assembler MUST additionally emit normalized stage and recap diagnostics fields to support routing and recap verification.

#### Scenario: Successful CA2 assemble cycle with Stage1-only path
- **WHEN** assembler completes CA2 cycle and Stage2 is skipped
- **THEN** diagnostics include baseline fields plus stage status and stage2 skip reason

#### Scenario: Successful CA2 assemble cycle with Stage2 path
- **WHEN** assembler completes CA2 cycle with Stage2 execution
- **THEN** diagnostics include baseline fields plus stage2 status and recap status

#### Scenario: Guard-failed assemble cycle
- **WHEN** assembler fails due to guard violation
- **THEN** diagnostics include failure status and guard violation summary

### Requirement: Context assembler SHALL support configurable stage failure policy
Context assembler MUST allow stage failure policy to be configured for CA2, including fail-fast and best-effort behavior, and MUST apply the selected policy deterministically.

#### Scenario: Stage2 fail-fast policy
- **WHEN** Stage2 provider returns error and policy is fail-fast
- **THEN** assembler terminates current model step with classified context error

#### Scenario: Stage2 best-effort policy
- **WHEN** Stage2 provider returns error and policy is best-effort
- **THEN** assembler proceeds with Stage1 output, records degraded stage status, and keeps model step runnable

