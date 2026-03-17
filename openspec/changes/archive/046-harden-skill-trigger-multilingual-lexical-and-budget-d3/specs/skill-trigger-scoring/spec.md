## ADDED Requirements

### Requirement: Skill loader SHALL support deterministic mixed CJK-English lexical tokenization
Skill lexical trigger scoring MUST support `mixed_cjk_en` tokenization so that Chinese and mixed Chinese-English inputs can participate in lexical matching without changing external loader APIs.

`mixed_cjk_en` mode MUST remain deterministic for the same input and MUST preserve existing lexical weighted-keyword scoring pipeline semantics (weighting, thresholding, tie-break order).

#### Scenario: Chinese lexical trigger is recognized under default tokenizer mode
- **WHEN** runtime uses default skill trigger scoring configuration and user input is Chinese
- **THEN** loader produces non-empty lexical tokens and evaluates candidate skills through lexical weighted-keyword scoring

#### Scenario: Mixed Chinese-English lexical trigger is recognized
- **WHEN** user input contains both Chinese and English trigger phrases
- **THEN** loader evaluates mixed-language tokens in one lexical scoring pass and keeps deterministic candidate ordering

### Requirement: Skill trigger scoring SHALL enforce semantic candidate budget with top-k control
After semantic candidates are filtered by confidence threshold and deterministically sorted, loader MUST apply `max_semantic_candidates` as top-k cap.

Candidates outside top-k MUST be pruned from semantic selection. Explicitly referenced skills MUST remain selectable and MUST NOT be dropped by semantic budget capping.

#### Scenario: Default top-k budget prunes low-ranked semantic candidates
- **WHEN** semantic candidates exceed default `max_semantic_candidates=3`
- **THEN** loader keeps top 3 semantic candidates and prunes the remaining candidates before compile assembly

#### Scenario: Explicit skill selection bypasses semantic budget pruning
- **WHEN** user input explicitly references a skill and semantic candidate count exceeds top-k
- **THEN** explicit skill selection remains effective while semantic pruning applies only to semantic candidates

### Requirement: Skill trigger observability SHALL include tokenizer and pruning signals
Skill diagnostics/events MUST include additive fields `tokenizer_mode` and `candidate_pruned_count` for each compile evaluation path.

`candidate_pruned_count` MUST represent the number of semantic candidates removed by top-k budget and MUST be `0` when no semantic candidate is pruned.

#### Scenario: No pruning path exposes zero pruned count
- **WHEN** semantic candidate count is less than or equal to configured top-k budget
- **THEN** diagnostics expose `candidate_pruned_count=0` with active `tokenizer_mode`

#### Scenario: Pruning path exposes non-zero pruned count
- **WHEN** semantic candidate count exceeds configured top-k budget
- **THEN** diagnostics expose active `tokenizer_mode` and `candidate_pruned_count` greater than `0`

### Requirement: Run and Stream SHALL preserve multilingual lexical-budget semantic equivalence
For equivalent input, equivalent effective configuration, and equivalent scorer behavior, Run and Stream MUST produce semantically equivalent skill selection outcomes with multilingual lexical tokenization and semantic budget pruning enabled.

#### Scenario: Equivalent multilingual lexical selection in Run and Stream
- **WHEN** equivalent Chinese or mixed Chinese-English requests execute under the same tokenizer mode and threshold settings
- **THEN** Run and Stream produce semantically equivalent selected skill set and deterministic order

#### Scenario: Equivalent top-k pruning in Run and Stream
- **WHEN** equivalent requests produce semantic candidates exceeding `max_semantic_candidates`
- **THEN** Run and Stream prune the same semantic candidates and expose semantically equivalent pruning diagnostics
