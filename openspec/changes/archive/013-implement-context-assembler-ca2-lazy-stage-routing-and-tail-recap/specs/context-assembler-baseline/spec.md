## MODIFIED Requirements

### Requirement: Runner SHALL execute context assembler at pre-model stage
The runner MUST invoke context assembler before each model invocation in both Run and Stream flows. This pre-model hook MUST produce deterministic context payloads without changing existing external run/stream semantic contracts. Starting in CA2, this hook MUST support two-stage assembly execution (Stage1 then optional Stage2) while preserving the same external event ordering and complete-only tool-call contract.

#### Scenario: Run path calls model after staged assembler
- **WHEN** runner executes a Run request with context assembler enabled and CA2 staging enabled
- **THEN** context assembler executes Stage1 and conditional Stage2 before model invocation, and the model receives staged assembled payload

#### Scenario: Stream path preserves semantic parity with staged assembler
- **WHEN** runner executes a Stream request with context assembler enabled and CA2 staging enabled
- **THEN** staged assembler executes before stream model invocation and streaming event semantics remain unchanged

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

## ADDED Requirements

### Requirement: Context assembler SHALL support configurable stage failure policy
Context assembler MUST allow stage failure policy to be configured for CA2, including fail-fast and best-effort behavior, and MUST apply the selected policy deterministically.

#### Scenario: Stage2 fail-fast policy
- **WHEN** Stage2 provider returns error and policy is fail-fast
- **THEN** assembler terminates current model step with classified context error

#### Scenario: Stage2 best-effort policy
- **WHEN** Stage2 provider returns error and policy is best-effort
- **THEN** assembler proceeds with Stage1 output, records degraded stage status, and keeps model step runnable
