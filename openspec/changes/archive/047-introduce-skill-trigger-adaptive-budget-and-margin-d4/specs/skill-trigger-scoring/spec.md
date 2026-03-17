## MODIFIED Requirements

### Requirement: Skill trigger scoring SHALL enforce semantic candidate budget with top-k control
After semantic candidates are filtered by confidence threshold and deterministically sorted, loader MUST apply semantic candidate budget by configured budget mode.

Budget mode semantics:
- `fixed`: select top `max_semantic_candidates`.
- `adaptive`: select candidates deterministically within `[min_k, max_k]` using score-margin rule `min_score_margin`.

In adaptive mode, implementation MUST produce deterministic selection count for equivalent input/config/scorer outputs and MUST expose a normalized decision reason.

Candidates outside selected budget MUST be pruned from semantic selection. Explicitly referenced skills MUST remain selectable and MUST NOT be dropped by semantic budget capping.

#### Scenario: Default adaptive budget keeps minimum candidates when top score is clearly separated
- **WHEN** runtime uses default adaptive budget config and `top1-top2 >= min_score_margin`
- **THEN** loader selects `min_k` semantic candidates and prunes the remainder

#### Scenario: Adaptive budget expands candidates for close-score cluster
- **WHEN** runtime uses adaptive budget and top-ranked candidates have score margins below `min_score_margin`
- **THEN** loader expands selected semantic candidates up to `max_k` deterministically

#### Scenario: Fixed budget mode preserves top-k behavior
- **WHEN** runtime sets budget mode to `fixed` with `max_semantic_candidates=N`
- **THEN** loader selects exactly top `N` semantic candidates after sorting and threshold filtering

#### Scenario: Explicit skill selection bypasses semantic budget pruning
- **WHEN** user input explicitly references a skill and semantic candidate count exceeds selected budget
- **THEN** explicit skill selection remains effective while budget pruning applies only to semantic candidates

### Requirement: Skill trigger observability SHALL include tokenizer and pruning signals
Skill diagnostics/events MUST include additive lexical-budget fields for each compile evaluation path:
- `tokenizer_mode`
- `candidate_pruned_count`
- `budget_mode`
- `selected_semantic_count`
- `score_margin_top1_top2`
- `budget_decision_reason`

`candidate_pruned_count` MUST represent the number of semantic candidates removed by selected budget and MUST be `0` when no semantic candidate is pruned.

`score_margin_top1_top2` MUST be normalized when at least two semantic candidates exist.

#### Scenario: Adaptive minimum-selection path exposes decision fields
- **WHEN** adaptive budget keeps minimum candidates due to clear top margin
- **THEN** diagnostics include `budget_mode=adaptive`, selected count, top1-top2 margin, and normalized decision reason

#### Scenario: Fixed-mode path exposes deterministic budget fields
- **WHEN** fixed mode applies top-k selection
- **THEN** diagnostics include `budget_mode=fixed`, selected count, and pruning count consistent with fixed top-k result

### Requirement: Run and Stream SHALL preserve multilingual lexical-budget semantic equivalence
For equivalent input, equivalent effective configuration, and equivalent scorer behavior, Run and Stream MUST produce semantically equivalent skill selection outcomes with multilingual lexical tokenization and semantic budget control enabled.

#### Scenario: Equivalent adaptive-budget selection in Run and Stream
- **WHEN** equivalent Chinese or mixed Chinese-English requests execute under the same adaptive budget configuration
- **THEN** Run and Stream produce semantically equivalent selected skill set, order, and selected semantic candidate count

#### Scenario: Equivalent fixed-budget selection in Run and Stream
- **WHEN** equivalent requests execute under fixed budget mode with the same top-k setting
- **THEN** Run and Stream prune the same semantic candidates and expose semantically equivalent lexical-budget diagnostics
