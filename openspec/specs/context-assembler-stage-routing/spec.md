# context-assembler-stage-routing Specification

## Purpose
TBD - created by archiving change implement-context-assembler-ca2-lazy-stage-routing-and-tail-recap. Update Purpose after archive.
## Requirements
### Requirement: Context assembler SHALL support CA2 two-stage assembly routing
Context assembler MUST execute Stage1 before Stage2. Stage2 invocation MUST be decided by configured CA2 routing mode and MUST remain deterministic and traceable.

When `routing_mode=rules`, Stage2 decision MUST follow existing rule-based routing conditions.
When `routing_mode=agentic`, Stage2 decision MUST follow agentic router callback output, subject to callback failure fallback policy.

#### Scenario: Rules mode skips Stage2 when routing threshold is not met
- **WHEN** `routing_mode=rules` and Stage1 output does not satisfy Stage2 trigger conditions
- **THEN** assembler skips Stage2 and records a normalized skip reason

#### Scenario: Rules mode triggers Stage2 when routing threshold is met
- **WHEN** `routing_mode=rules` and Stage1 output satisfies Stage2 trigger conditions
- **THEN** assembler invokes Stage2 provider and merges Stage2 output into assembled context

#### Scenario: Agentic mode triggers Stage2 from callback decision
- **WHEN** `routing_mode=agentic` and callback returns `run_stage2=true` with valid reason
- **THEN** assembler invokes Stage2 and records router decision metadata

#### Scenario: Agentic mode skips Stage2 from callback decision
- **WHEN** `routing_mode=agentic` and callback returns `run_stage2=false` with valid reason
- **THEN** assembler skips Stage2 and records router decision metadata

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
CA2 routing MUST provide a host callback extension for agentic decisioning. In `routing_mode=agentic`, assembler MUST call the registered callback with bounded timeout.

If callback is missing, times out, returns an error, or returns an invalid decision payload, assembler MUST fallback to `rules` routing under `best_effort` policy and MUST NOT terminate assemble flow solely due to agentic callback failure.

#### Scenario: Agentic callback is available and returns valid decision
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and registered callback returns valid decision
- **THEN** assembler applies callback decision and continues assemble flow

#### Scenario: Agentic callback is not registered
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and no callback is registered
- **THEN** assembler falls back to `rules` routing and records fallback reason

#### Scenario: Agentic callback times out
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and callback exceeds configured timeout
- **THEN** assembler falls back to `rules` routing, records timeout reason, and continues assemble flow

#### Scenario: Agentic callback returns error or invalid payload
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and callback returns error or invalid decision payload
- **THEN** assembler falls back to `rules` routing, records normalized router error, and continues assemble flow

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

### Requirement: CA2 Stage2 external retriever observability SHALL preserve existing stage policy behavior
CA2 Stage2 external retriever observability enhancements MUST NOT change existing stage policy semantics (`fail_fast` and `best_effort`).

Threshold-hit evaluation and provider trend aggregation MUST be observational only in this milestone.

#### Scenario: fail_fast policy with threshold hit
- **WHEN** Stage2 runs under `fail_fast` policy and threshold-hit signal is produced
- **THEN** Stage2 execution behavior remains governed by existing fail_fast semantics without additional automatic actions

#### Scenario: best_effort policy with threshold hit
- **WHEN** Stage2 runs under `best_effort` policy and threshold-hit signal is produced
- **THEN** Stage2 execution behavior remains governed by existing best_effort semantics without additional automatic actions

### Requirement: CA2 Stage2 error-layer trend semantics SHALL allow enum extension
CA2 Stage2 diagnostics trend aggregation MUST support baseline error layers (`transport`, `protocol`, `semantic`) and MUST allow forward-compatible enum extension.

#### Scenario: Baseline error layers are aggregated
- **WHEN** Stage2 retrieval failures occur across baseline layers
- **THEN** trend diagnostics aggregate and expose layer distribution without schema conflict

#### Scenario: Extended error layer value is emitted
- **WHEN** an implementation emits a new layer enum value in a backward-compatible extension
- **THEN** diagnostics trend aggregation accepts and preserves the value without failing parsing

### Requirement: CA2 Stage2 retriever SPI SHALL support capability-hint extension without provider coupling in assembler flow
CA2 Stage2 retriever SPI MUST support optional capability-hint extension fields that can be consumed by provider adapters.

Assembler routing and stage orchestration MUST remain provider-agnostic and MUST NOT introduce provider-specific branch logic in the main CA2 flow for this milestone.

#### Scenario: Capability hints are provided and consumed by adapter
- **WHEN** runtime config enables capability hints and Stage2 invokes an adapter that supports relevant hints
- **THEN** assembler forwards hints through SPI extension fields and Stage2 execution completes without changing main routing semantics

#### Scenario: Capability hints are absent
- **WHEN** Stage2 request does not include capability hints
- **THEN** Stage2 execution follows existing SPI baseline behavior with no additional routing side effects

### Requirement: CA2 Stage2 template-pack resolution SHALL be deterministic and support explicit-only mode
CA2 Stage2 external retrieval MUST support a standardized template-pack profile set for this milestone:
- `graphrag_like`
- `ragflow_like`
- `elasticsearch_like`

Template resolution MUST apply `profile defaults -> explicit overrides` precedence, and MUST allow explicit mapping fields to run independently when no profile is selected.

#### Scenario: Profile defaults are resolved and explicit fields override
- **WHEN** Stage2 external config selects `ragflow_like` and also provides explicit mapping/auth/header fields
- **THEN** Stage2 resolves `ragflow_like` defaults first and applies explicit fields as final values

#### Scenario: Explicit-only mapping is selected
- **WHEN** Stage2 external config omits template-pack profile and provides explicit mapping/auth/header fields
- **THEN** Stage2 executes retrieval using explicit mapping only without requiring template defaults

### Requirement: CA2 Stage2 capability-hint mismatch SHALL remain observational only
When capability hints are unsupported, invalid, or mismatched for the selected provider path, Stage2 MUST emit normalized mismatch diagnostics and MUST NOT trigger automatic provider switching, route mutation, or stage-policy changes.

#### Scenario: Adapter does not support provided hint
- **WHEN** Stage2 receives a capability hint that selected adapter does not support
- **THEN** Stage2 records normalized hint-mismatch diagnostics and continues according to existing stage policy semantics

#### Scenario: Hint payload is malformed but stage policy is best_effort
- **WHEN** Stage2 receives malformed capability-hint payload and stage policy is `best_effort`
- **THEN** Stage2 records mismatch reason and continues with degraded-but-compatible behavior under existing best_effort rules

### Requirement: CA2 Stage2 hint and template semantics SHALL remain equivalent between Run and Stream
For equivalent inputs and configuration, Run and Stream MUST produce semantically equivalent outcomes for template resolution, hint application/mismatch classification, and Stage2 result classification, while allowing implementation-level event ordering differences.

#### Scenario: Equivalent profile and hint path in Run and Stream
- **WHEN** Run and Stream execute equivalent Stage2 requests with the same template-pack profile and hint set
- **THEN** both paths expose semantically equivalent resolved-profile and hint-outcome diagnostics

#### Scenario: Equivalent hint mismatch path in Run and Stream
- **WHEN** Run and Stream execute equivalent Stage2 requests that produce the same hint mismatch condition
- **THEN** both paths expose semantically equivalent mismatch reason and Stage2 classification outcomes

### Requirement: CA2 routing decisions SHALL remain semantically equivalent between Run and Stream
For equivalent inputs and effective configuration, Run and Stream MUST produce semantically equivalent CA2 routing outcomes in both `rules` and `agentic` modes, allowing implementation-level event ordering differences.

#### Scenario: Equivalent callback-driven decision in Run and Stream
- **WHEN** equivalent requests execute in `routing_mode=agentic` with the same callback behavior
- **THEN** Run and Stream expose semantically equivalent Stage2 invoke/skip outcomes and router reason classes

#### Scenario: Equivalent callback failure fallback in Run and Stream
- **WHEN** equivalent requests execute in `routing_mode=agentic` and callback path fails with the same failure class
- **THEN** Run and Stream both fallback to `rules` and expose semantically equivalent fallback reason classes

