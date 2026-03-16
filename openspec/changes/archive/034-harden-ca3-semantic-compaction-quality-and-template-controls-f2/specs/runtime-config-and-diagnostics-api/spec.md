## MODIFIED Requirements

### Requirement: Runtime SHALL expose CA3 semantic quality and template controls with fail-fast validation
Runtime configuration MUST expose CA3 semantic quality and template controls with precedence `env > file > default`, including:
- `context_assembler.ca3.compaction.quality.threshold`
- `context_assembler.ca3.compaction.quality.weights.coverage`
- `context_assembler.ca3.compaction.quality.weights.compression`
- `context_assembler.ca3.compaction.quality.weights.validity`
- `context_assembler.ca3.compaction.semantic_template.prompt`
- `context_assembler.ca3.compaction.semantic_template.allowed_placeholders`

Invalid threshold/weights/template values MUST fail fast on startup and hot reload.

#### Scenario: Startup with default quality/template config
- **WHEN** runtime starts without explicit overrides
- **THEN** effective config resolves to valid defaults and semantic compaction remains deterministic

#### Scenario: Invalid quality threshold or weights
- **WHEN** threshold is outside `[0,1]` or quality weights are invalid
- **THEN** runtime rejects startup/hot reload snapshot with validation error

#### Scenario: Invalid semantic template placeholder config
- **WHEN** template prompt is empty, has unbalanced placeholders, or uses non-whitelisted placeholders
- **THEN** runtime rejects startup/hot reload snapshot with validation error

### Requirement: Runtime SHALL expose CA3 embedding hook placeholder config without adapter binding requirement in this milestone
Runtime configuration MUST expose CA3 embedding hook placeholder fields:
- `context_assembler.ca3.compaction.embedding.enabled`
- `context_assembler.ca3.compaction.embedding.selector`

This milestone MUST NOT require provider adapter binding for startup success when hook remains placeholder-only.

#### Scenario: Embedding hook disabled
- **WHEN** embedding hook is not enabled
- **THEN** runtime uses rule-only quality scoring path without embedding adapter dependency

#### Scenario: Embedding hook enabled with selector configured
- **WHEN** hook is enabled and selector is configured
- **THEN** runtime accepts configuration and CA3 remains deterministic even if adapter is not bound in this milestone

### Requirement: Runtime diagnostics SHALL expose CA3 compaction quality and fallback reason fields
Run diagnostics MUST expose additive CA3 compaction fields:
- `ca3_compaction_fallback_reason`
- `ca3_compaction_quality_score`
- `ca3_compaction_quality_reason`

These fields MUST be backward-compatible and semantically equivalent between Run and Stream for equivalent inputs/config.

#### Scenario: Consumer inspects successful semantic quality gate
- **WHEN** semantic output passes quality gate
- **THEN** diagnostics include non-empty quality score/reason and empty or absent fallback reason

#### Scenario: Consumer inspects quality-gate fallback
- **WHEN** semantic output fails quality gate under `best_effort`
- **THEN** diagnostics include fallback reason `quality_below_threshold` with quality score/reason

#### Scenario: Consumer compares Run and Stream diagnostics
- **WHEN** equivalent requests run through Run and Stream with same CA3 config
- **THEN** quality score/reason and fallback reason semantics are equivalent across both paths
