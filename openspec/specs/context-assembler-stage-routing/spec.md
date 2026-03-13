# context-assembler-stage-routing Specification

## Purpose
TBD - created by archiving change implement-context-assembler-ca2-lazy-stage-routing-and-tail-recap. Update Purpose after archive.
## Requirements
### Requirement: Context assembler SHALL support CA2 two-stage assembly routing
Context assembler MUST execute Stage1 before Stage2. Stage2 MUST be invoked only when routing rules determine Stage1 output is insufficient. Routing decisions MUST be deterministic and traceable.

#### Scenario: Stage1 satisfies request and Stage2 is skipped
- **WHEN** Stage1 output satisfies configured routing conditions
- **THEN** assembler skips Stage2 and records a normalized skip reason

#### Scenario: Stage1 is insufficient and Stage2 is triggered
- **WHEN** routing rules detect required context gaps after Stage1
- **THEN** assembler invokes Stage2 provider and merges Stage2 output into assembled context

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

### Requirement: Tail recap SHALL append minimal stable fields
Assembler MUST append a tail recap block at the end of assembled context with stable field order and minimum schema: `status`, `decisions`, `todo`, `risks`.

#### Scenario: Tail recap is enabled
- **WHEN** CA2 tail recap is enabled
- **THEN** assembled context contains recap block at the tail with all minimum fields

#### Scenario: Tail recap content exceeds configured limits
- **WHEN** recap payload violates configured size limit
- **THEN** assembler applies deterministic truncation/sanitization and records recap status

### Requirement: Routing engine SHALL provide agentic extension hook placeholder
CA2 routing MUST provide a documented extension hook for future agentic decision providers, while current production decision path remains rule-based.

#### Scenario: Default routing mode
- **WHEN** runtime runs CA2 without extension provider
- **THEN** assembler uses deterministic rule-based routing only

#### Scenario: Agentic hook is configured in CA2 baseline
- **WHEN** runtime config references agentic decision mode in CA2
- **THEN** runtime returns explicit not-ready/TODO classification until agentic provider milestone is implemented

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

### Requirement: CA3 pressure decisions SHALL remain semantically equivalent between Run and Stream
For equivalent inputs and configuration, Run and Stream paths MUST produce semantically equivalent CA3 pressure-zone decisions, allowing implementation-level event order differences.

#### Scenario: Equivalent pressure path in Run and Stream
- **WHEN** Run and Stream process equivalent requests under identical CA3 pressure config
- **THEN** both paths report equivalent pressure-zone outcomes in diagnostics

#### Scenario: Equivalent emergency downgrade in Run and Stream
- **WHEN** Run and Stream both enter emergency pressure zone
- **THEN** both paths apply equivalent low-priority rejection semantics and record equivalent downgrade reason classes

