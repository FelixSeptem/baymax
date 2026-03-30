## ADDED Requirements

### Requirement: CA2 Stage2 memory access SHALL route through unified memory facade
Context assembler CA2 Stage2 MUST access memory via unified memory facade rather than provider-specific branches in assembler flow.

Facade resolution MUST be driven by effective memory mode and profile, while assembler routing logic remains provider-agnostic.

#### Scenario: Stage2 runs with builtin filesystem mode
- **WHEN** effective memory mode is `builtin_filesystem`
- **THEN** CA2 Stage2 reads and writes memory through builtin engine facade without provider-specific assembler branching

#### Scenario: Stage2 runs with external SPI mode
- **WHEN** effective memory mode is `external_spi` with canonical profile
- **THEN** CA2 Stage2 executes memory calls through SPI facade and receives normalized results

### Requirement: CA2 stage policy SHALL preserve existing semantics with memory fallback integration
CA2 Stage2 memory integration MUST preserve existing stage policy semantics:
- `fail_fast` remains fail-fast,
- `best_effort` remains continue-with-observable-degradation.

Memory fallback handling MUST execute within these existing policy boundaries and MUST NOT silently mutate stage policy semantics.

#### Scenario: Best effort path with external memory failure and degrade fallback
- **WHEN** Stage2 runs with `best_effort`, external SPI fails, and fallback policy allows degradation
- **THEN** assembler continues flow with deterministic degraded status and canonical fallback reason

#### Scenario: Fail fast path with blocking memory failure
- **WHEN** Stage2 runs with `fail_fast` and memory path returns blocking failure
- **THEN** assembler terminates assemble flow with canonical memory reason and no partial state commit

### Requirement: CA2 routing outcomes SHALL remain equivalent between Run and Stream after memory facade integration
For equivalent requests and effective configuration, Run and Stream CA2 routing outcomes MUST remain semantically equivalent after introducing memory facade and mode switching.

#### Scenario: Equivalent CA2 route decision under same memory mode
- **WHEN** equivalent requests run through Run and Stream with same routing mode and memory mode
- **THEN** both paths expose semantically equivalent Stage2 invoke or skip outcomes and memory classification

#### Scenario: Equivalent fallback branch under same failure class
- **WHEN** equivalent Run and Stream requests encounter same memory failure class and fallback policy
- **THEN** both paths expose semantically equivalent fallback-used classification without route-semantic drift
