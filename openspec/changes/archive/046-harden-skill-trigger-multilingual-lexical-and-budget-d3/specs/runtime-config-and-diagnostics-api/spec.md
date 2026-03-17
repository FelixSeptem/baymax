## ADDED Requirements

### Requirement: Runtime SHALL expose skill lexical tokenizer mode and semantic candidate budget with deterministic precedence
Runtime configuration MUST expose multilingual lexical and semantic budget controls under skill trigger scoring configuration with precedence `env > file > default`.

At minimum, runtime MUST support:
- `skill.trigger_scoring.lexical.tokenizer_mode`
- `skill.trigger_scoring.max_semantic_candidates`

For this milestone, configuration MUST be managed through JSON/YAML path (with env overrides) and MUST NOT require additional CLI parameters.

#### Scenario: Environment overrides file for tokenizer and budget controls
- **WHEN** both YAML and environment variables define tokenizer mode and semantic candidate budget
- **THEN** effective runtime config resolves tokenizer and budget by `env > file > default`

#### Scenario: Startup uses default tokenizer and budget values
- **WHEN** runtime starts without explicit tokenizer mode or semantic budget configuration
- **THEN** effective config uses `tokenizer_mode=mixed_cjk_en` and `max_semantic_candidates=3`

### Requirement: Runtime SHALL fail fast on invalid skill lexical-budget configuration
Runtime startup and hot reload MUST validate skill lexical-budget controls before activation.

Validation MUST reject:
- unsupported `tokenizer_mode` values,
- non-positive `max_semantic_candidates`.

Invalid updates MUST NOT replace active configuration snapshot.

#### Scenario: Invalid tokenizer mode fails startup
- **WHEN** runtime configuration sets unsupported tokenizer mode
- **THEN** startup fails fast with validation error

#### Scenario: Invalid semantic budget fails hot reload and rolls back
- **WHEN** hot reload applies `max_semantic_candidates <= 0`
- **THEN** reload is rejected and runtime keeps previous valid configuration

### Requirement: Runtime diagnostics SHALL expose additive lexical-budget observability fields
Runtime diagnostics MUST include additive skill trigger fields:
- `tokenizer_mode`
- `candidate_pruned_count`

These fields MUST remain backward-compatible and MUST NOT alter existing skill lifecycle diagnostics semantics.

#### Scenario: Diagnostics include tokenizer mode and pruning count fields
- **WHEN** application queries skill diagnostics after compile evaluation
- **THEN** diagnostics payload includes `tokenizer_mode` and `candidate_pruned_count`

#### Scenario: Legacy consumers remain compatible with additive diagnostics fields
- **WHEN** existing diagnostics consumers read skill lifecycle records without parsing new fields
- **THEN** original lifecycle semantics remain unchanged

### Requirement: Run and Stream SHALL preserve lexical-budget diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent lexical-budget diagnostics fields.

#### Scenario: Equivalent lexical-budget diagnostics in Run and Stream
- **WHEN** equivalent requests execute with same tokenizer mode and semantic budget
- **THEN** diagnostics for `tokenizer_mode` and `candidate_pruned_count` are semantically equivalent across Run and Stream
