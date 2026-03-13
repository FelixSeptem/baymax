# context-assembler-production-convergence Specification

## Purpose
TBD - created by archiving change implement-context-assembler-ca4-production-convergence. Update Purpose after archive.
## Requirements
### Requirement: CA4 SHALL standardize threshold strategy computation
Context Assembler CA4 MUST apply a deterministic threshold strategy that first resolves stage-specific overrides, then evaluates both percentage and absolute thresholds, and finally selects the higher pressure zone when triggers differ.

#### Scenario: Stage override exists and differs from global thresholds
- **WHEN** stage-specific thresholds are configured and valid
- **THEN** CA4 uses stage thresholds for that stage instead of global thresholds

#### Scenario: Percentage and absolute triggers disagree
- **WHEN** percentage trigger maps to a lower zone than absolute trigger
- **THEN** CA4 selects the higher zone and records trigger reason for diagnostics

### Requirement: CA4 SHALL keep token counting non-blocking with fixed fallback order
In `sdk_preferred` mode, token counting MUST follow fallback order: provider counter -> local tiktoken estimate -> lightweight estimate, and counting failure MUST NOT terminate run/stream execution.

#### Scenario: Provider counter fails
- **WHEN** provider `CountTokens` returns error or unsupported
- **THEN** CA4 falls back to local tiktoken estimate and continues execution

#### Scenario: Local tokenizer initialization fails
- **WHEN** local tiktoken estimate cannot initialize (e.g., no encoding resource)
- **THEN** CA4 falls back to lightweight estimate and continues execution

### Requirement: CA4 SHALL preserve Run and Stream semantic equivalence
CA4 pressure-zone decisions, fallback behavior, and diagnostics semantics MUST remain equivalent between `Run` and `Stream` paths for the same effective input context.

#### Scenario: Same context under Run and Stream
- **WHEN** identical context inputs are processed through Run and Stream
- **THEN** zone and trigger semantics are equivalent and diagnostics fields are comparable

### Requirement: CA4 SHALL define OpenAI counting semantics for threshold control
For OpenAI provider path, local tokenizer-based counts MUST be treated as threshold-control estimates and MUST NOT claim billing-level precision semantics.

#### Scenario: OpenAI path computes token count
- **WHEN** CA4 reports token-related diagnostics for OpenAI
- **THEN** documentation and diagnostics semantics identify the count as threshold-control estimate

