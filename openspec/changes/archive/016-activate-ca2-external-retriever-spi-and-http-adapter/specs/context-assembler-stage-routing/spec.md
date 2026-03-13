## MODIFIED Requirements

### Requirement: Stage2 provider interface SHALL be extensible with file-first implementation
The context retrieval layer MUST expose a stable provider interface for Stage2. CA2 MUST keep local file provider as a supported path and MUST additionally support `http`, `rag`, `db`, and `elasticsearch` providers through a unified retriever SPI with normalized request/response/error semantics.

#### Scenario: File provider is selected
- **WHEN** runtime config selects file provider for Stage2
- **THEN** assembler loads retrieval payload from local file source through provider interface

#### Scenario: HTTP provider is selected
- **WHEN** runtime config selects http provider for Stage2
- **THEN** assembler calls configured HTTP retriever endpoint and maps request/response via configured JSON field mapping

#### Scenario: RAG/DB/Elasticsearch provider is selected
- **WHEN** runtime config selects rag, db, or elasticsearch provider for Stage2
- **THEN** assembler executes retrieval through the same SPI contract and returns normalized chunks or normalized provider error reason without partial state corruption

## ADDED Requirements

### Requirement: Stage2 retrieval SHALL preserve stage policy semantics
Stage2 retrieval failures MUST preserve existing CA2 stage policy behavior: `fail_fast` MUST terminate assemble flow immediately, and `best_effort` MUST continue with degraded status and recorded skip reason.

#### Scenario: Stage2 retrieval fails in fail_fast mode
- **WHEN** Stage2 provider returns timeout/auth/mapping/transport error and stage policy is fail_fast
- **THEN** assemble flow terminates with error and commit diagnostics mark failed status

#### Scenario: Stage2 retrieval fails in best_effort mode
- **WHEN** Stage2 provider returns timeout/auth/mapping/transport error and stage policy is best_effort
- **THEN** assemble flow continues with degraded status and records normalized `stage2_reason`
