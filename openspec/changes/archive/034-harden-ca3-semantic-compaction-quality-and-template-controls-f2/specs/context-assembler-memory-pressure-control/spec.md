## MODIFIED Requirements

### Requirement: CA3 semantic compaction SHALL enforce deterministic quality gate before acceptance
When `context_assembler.ca3.compaction.mode=semantic`, CA3 MUST evaluate semantic output with deterministic rule-based scoring (`coverage`, `compression`, `validity`) and compare the result with configured threshold before accepting rewritten content.

#### Scenario: Quality score meets threshold
- **WHEN** semantic compaction returns output and computed score is greater than or equal to threshold
- **THEN** CA3 accepts semantic result and records quality score/reason diagnostics

#### Scenario: Quality score is below threshold under best-effort
- **WHEN** score is below threshold and stage policy is `best_effort`
- **THEN** CA3 falls back to `truncate` and records fallback reason `quality_below_threshold`

#### Scenario: Quality score is below threshold under fail-fast
- **WHEN** score is below threshold and stage policy is `fail_fast`
- **THEN** CA3 aborts assembly without partially applying semantic output

### Requirement: CA3 semantic prompt generation SHALL be controlled by validated template contract
CA3 semantic prompt generation MUST be driven by runtime template config with placeholder whitelist constraints. Invalid template configuration MUST fail fast at startup/hot reload.

#### Scenario: Prompt uses only allowed placeholders
- **WHEN** semantic template prompt contains balanced placeholders and all placeholders are in whitelist
- **THEN** runtime accepts configuration and CA3 renders prompt deterministically

#### Scenario: Prompt contains invalid placeholder
- **WHEN** semantic template prompt contains unsupported or unbalanced placeholders
- **THEN** runtime rejects configuration during startup/hot reload with validation error

### Requirement: CA3 semantic compaction SHALL preserve embedding hook placeholder semantics
CA3 MUST preserve embedding scorer hook semantics as a placeholder interface in this milestone. If hook is enabled but no adapter is bound, CA3 MUST continue deterministic rule-only quality scoring and surface explicit reason marker.

#### Scenario: Embedding hook enabled without bound adapter
- **WHEN** `embedding.enabled=true` and no runtime embedding adapter is available
- **THEN** CA3 computes quality via rule-only scoring path and diagnostics include hook-not-bound reason marker

#### Scenario: Embedding hook disabled
- **WHEN** `embedding.enabled=false`
- **THEN** CA3 behaves identically to rule-only semantic quality gate with no adapter dependency
