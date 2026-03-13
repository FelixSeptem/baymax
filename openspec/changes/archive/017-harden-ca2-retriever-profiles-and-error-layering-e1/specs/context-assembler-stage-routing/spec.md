## MODIFIED Requirements

### Requirement: Stage2 provider interface SHALL be extensible with file-first implementation
The context retrieval layer MUST expose a stable provider interface for Stage2. CA2 MUST keep local file provider as a supported path and MUST additionally support `http`, `rag`, `db`, and `elasticsearch` providers through a unified retriever SPI with normalized request/response/error semantics.

For non-file providers, Stage2 retrieval MUST support profile-based defaults with explicit override behavior. Runtime config MUST support at least `http_generic`, `ragflow_like`, `graphrag_like`, and `elasticsearch_like` profiles, and implementation MUST remain extensible for future profile additions without breaking existing configurations.

#### Scenario: File provider is selected
- **WHEN** runtime config selects file provider for Stage2
- **THEN** assembler loads retrieval payload from local file source through provider interface

#### Scenario: HTTP provider is selected
- **WHEN** runtime config selects http provider for Stage2
- **THEN** assembler calls configured HTTP retriever endpoint and maps request/response via configured JSON field mapping

#### Scenario: RAG/DB/Elasticsearch provider is selected
- **WHEN** runtime config selects rag, db, or elasticsearch provider for Stage2
- **THEN** assembler executes retrieval through the same SPI contract and returns normalized chunks or normalized provider error reason without partial state corruption

#### Scenario: Profile defaults are applied with explicit override
- **WHEN** runtime config selects a Stage2 external profile and also provides explicit mapping/auth/header fields
- **THEN** Stage2 retrieval uses profile defaults as baseline and applies explicit fields as final overrides

### Requirement: Stage2 retrieval SHALL preserve stage policy semantics
Stage2 retrieval failures MUST preserve existing CA2 stage policy behavior: `fail_fast` MUST terminate assemble flow immediately, and `best_effort` MUST continue with degraded status and recorded skip reason.

Stage2 retrieval failure classification MUST expose normalized error-layer semantics (`transport`, `protocol`, `semantic`) with stable reason code output, while preserving backward-compatible `stage2_reason` behavior.

#### Scenario: Stage2 retrieval fails in fail_fast mode
- **WHEN** Stage2 provider returns timeout/auth/mapping/transport error and stage policy is fail_fast
- **THEN** assemble flow terminates with error and commit diagnostics mark failed status

#### Scenario: Stage2 retrieval fails in best_effort mode
- **WHEN** Stage2 provider returns timeout/auth/mapping/transport error and stage policy is best_effort
- **THEN** assemble flow continues with degraded status and records normalized `stage2_reason`

#### Scenario: Stage2 retrieval emits layered reason in degraded path
- **WHEN** Stage2 retrieval fails in best_effort mode with a classified transport/protocol/semantic error
- **THEN** assembler records normalized reason layer and reason code without changing stage policy decision outcome
