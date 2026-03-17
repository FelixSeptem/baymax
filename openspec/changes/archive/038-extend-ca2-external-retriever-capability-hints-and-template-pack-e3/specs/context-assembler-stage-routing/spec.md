## ADDED Requirements

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
