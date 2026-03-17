## MODIFIED Requirements

### Requirement: Runtime SHALL expose skill lexical tokenizer mode and semantic candidate budget with deterministic precedence
Runtime configuration MUST expose multilingual lexical and semantic budget controls under skill trigger scoring configuration with precedence `env > file > default`.

At minimum, runtime MUST support:
- `skill.trigger_scoring.lexical.tokenizer_mode`
- `skill.trigger_scoring.max_semantic_candidates`
- `skill.trigger_scoring.budget.mode` (`fixed|adaptive`)
- `skill.trigger_scoring.budget.adaptive.min_k`
- `skill.trigger_scoring.budget.adaptive.max_k`
- `skill.trigger_scoring.budget.adaptive.min_score_margin`

Default budget configuration for this milestone:
- `budget.mode=adaptive`
- `budget.adaptive.min_k=1`
- `budget.adaptive.max_k=5`
- `budget.adaptive.min_score_margin=0.08`

For this milestone, configuration MUST be managed through JSON/YAML path (with env overrides) and MUST NOT require additional CLI parameters.

#### Scenario: Environment overrides file for lexical-budget controls
- **WHEN** YAML and environment variables both define skill lexical-budget controls
- **THEN** effective runtime config resolves lexical-budget values by `env > file > default`

#### Scenario: Startup uses default adaptive budget values
- **WHEN** runtime starts without explicit skill lexical-budget configuration
- **THEN** effective config uses `tokenizer_mode=mixed_cjk_en` and default adaptive budget values

### Requirement: Runtime SHALL fail fast on invalid skill lexical-budget configuration
Runtime startup and hot reload MUST validate skill lexical-budget controls before activation.

Validation MUST reject:
- unsupported `tokenizer_mode` values,
- non-positive `max_semantic_candidates`,
- unsupported `budget.mode` values,
- non-positive `budget.adaptive.min_k`,
- `budget.adaptive.max_k < budget.adaptive.min_k`,
- `budget.adaptive.max_k > max_semantic_candidates`,
- `budget.adaptive.min_score_margin` outside `[0,1]`.

Invalid updates MUST NOT replace active configuration snapshot.

#### Scenario: Invalid adaptive margin fails startup
- **WHEN** runtime configuration sets adaptive `min_score_margin` outside `[0,1]`
- **THEN** startup fails fast with validation error

#### Scenario: Invalid adaptive range fails hot reload and rolls back
- **WHEN** hot reload applies adaptive budget with `max_k < min_k` or `max_k > max_semantic_candidates`
- **THEN** reload is rejected and runtime keeps previous valid configuration

### Requirement: Runtime diagnostics SHALL expose additive lexical-budget observability fields
Runtime diagnostics MUST include additive skill trigger fields:
- `tokenizer_mode`
- `candidate_pruned_count`
- `budget_mode`
- `selected_semantic_count`
- `score_margin_top1_top2`
- `budget_decision_reason`

These fields MUST remain backward-compatible and MUST NOT alter existing skill lifecycle diagnostics semantics.

#### Scenario: Diagnostics include adaptive budget decision fields
- **WHEN** application queries skill diagnostics after adaptive budget selection
- **THEN** diagnostics payload includes budget mode, selected semantic count, top1-top2 margin, and budget decision reason

#### Scenario: Legacy consumers remain compatible with additive diagnostics fields
- **WHEN** existing diagnostics consumers read skill lifecycle records without parsing new fields
- **THEN** original lifecycle semantics remain unchanged

### Requirement: Run and Stream SHALL preserve lexical-budget diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent lexical-budget diagnostics fields.

#### Scenario: Equivalent lexical-budget diagnostics in Run and Stream
- **WHEN** equivalent requests execute with the same tokenizer mode and budget controls
- **THEN** diagnostics for lexical-budget fields are semantically equivalent across Run and Stream
