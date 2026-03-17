# context-assembler-memory-pressure-control Specification

## Purpose
TBD - created by archiving change implement-context-assembler-ca3-memory-pressure-and-recovery. Update Purpose after archive.
## Requirements
### Requirement: Context assembler SHALL enforce tiered memory pressure control
Context assembler MUST implement a tiered pressure strategy with deterministic transitions across at least five zones: safe, comfort, warning, danger, and emergency. The strategy MUST preserve a Goldilocks target band and support operator-tunable thresholds.

#### Scenario: Pressure remains in safe zone
- **WHEN** current context usage is below configured safe threshold
- **THEN** assembler continues normal loading without squash/prune/spill actions

#### Scenario: Pressure enters warning and danger zones
- **WHEN** context usage crosses warning or danger thresholds
- **THEN** assembler triggers configured batch squash/prune actions and records the triggered zone in diagnostics

#### Scenario: Pressure enters emergency zone
- **WHEN** context usage crosses emergency threshold
- **THEN** assembler enters protection mode, spills low-priority context to file storage, and rejects low-priority new loads by default

### Requirement: Pressure threshold evaluation SHALL support percentage and absolute token triggers
Pressure control MUST evaluate both percentage-based thresholds and absolute token limits. Crossing either configured condition MUST trigger the corresponding pressure zone.

#### Scenario: Absolute token threshold triggers before percentage threshold
- **WHEN** token usage exceeds configured absolute threshold while percentage remains below warning
- **THEN** assembler transitions to the zone implied by absolute threshold trigger

#### Scenario: Percentage threshold triggers before absolute threshold
- **WHEN** usage percentage exceeds configured threshold while absolute token usage is below its threshold
- **THEN** assembler transitions to the zone implied by percentage trigger

### Requirement: Squash and prune SHALL honor critical and immutable protection flags
Batch squash/prune logic MUST NOT compress or delete entries marked as `critical` or `immutable`.

#### Scenario: Critical entry present during prune
- **WHEN** prune execution evaluates candidate entries and encounters `critical=true`
- **THEN** that entry is excluded from deletion decisions

#### Scenario: Immutable entry present during squash
- **WHEN** squash execution evaluates candidate entries and encounters `immutable=true`
- **THEN** that entry is excluded from compression/rewrite operations

### Requirement: Spill and swap SHALL preserve provenance for recovery
Spill/swap operations MUST persist recoverable metadata including `origin_ref` so replay and recovery can reconstruct source lineage.

#### Scenario: Spilled content is later swapped back
- **WHEN** assembler loads previously spilled content back into active context
- **THEN** recovered entry includes its `origin_ref` chain for audit and replay

### Requirement: CA3 recovery SHALL guarantee single-process consistency for cancel/retry/replay
Within a single process, cancel/retry/replay flows MUST maintain consistent context state transitions without torn state.

#### Scenario: Cancel followed by retry
- **WHEN** a run is canceled during pressure handling and retried in the same process
- **THEN** assembler restores consistent state and does not duplicate spill/prune side effects

#### Scenario: Replay after spill
- **WHEN** replay is requested for a run that triggered spill
- **THEN** assembler reconstructs state from persisted spill metadata without missing protected entries

### Requirement: CA4 threshold precedence SHALL be explicit and testable
Memory pressure control MUST define and test precedence of global and stage-level thresholds, including conflict resolution for mixed threshold triggers.

#### Scenario: Global and stage thresholds both configured
- **WHEN** stage-level thresholds are present for active stage
- **THEN** active stage uses stage-level thresholds and does not mix global values for that stage

#### Scenario: Trigger conflict during pressure evaluation
- **WHEN** percent and absolute threshold evaluations produce different zones
- **THEN** higher-pressure zone is selected consistently and diagnostics include trigger source

### Requirement: CA4 counting fallback SHALL preserve pressure safety
If provider counting is unavailable, memory pressure control MUST still produce stable zone computation through local tokenizer and fallback estimator paths.

#### Scenario: Provider counting unavailable in sdk_preferred mode
- **WHEN** provider counting fails in pressure evaluation
- **THEN** fallback estimates are used and pressure safety actions continue to work

### Requirement: CA3 SHALL support pluggable compaction strategies with deterministic mode selection
Context Assembler CA3 MUST support pluggable compaction strategies through an internal SPI with at least `truncate` and `semantic` modes. Runtime selection MUST be deterministic from effective config and MUST default to `truncate`.

#### Scenario: Startup with default compaction mode
- **WHEN** runtime starts without explicit CA3 compaction mode override
- **THEN** CA3 uses `truncate` mode and preserves existing squash behavior compatibility

#### Scenario: Startup with semantic compaction mode
- **WHEN** runtime config sets CA3 compaction mode to `semantic`
- **THEN** CA3 executes semantic compaction strategy through configured client path

### Requirement: CA3 semantic compaction SHALL use existing LLM client path
When `semantic` mode is enabled, CA3 MUST perform semantic compaction by invoking the current LLM client path used by runtime model execution and MUST NOT require an additional standalone provider stack in this milestone.

#### Scenario: Semantic compaction executes on warning-or-higher pressure
- **WHEN** pressure zone enters `warning`, `danger`, or `emergency` and mode is `semantic`
- **THEN** CA3 invokes semantic compaction through the current LLM client integration and records execution outcome

#### Scenario: Semantic compaction respects configured timeout
- **WHEN** semantic compaction exceeds configured timeout
- **THEN** CA3 handles timeout using configured stage policy semantics without partial state corruption

### Requirement: CA3 compaction failure handling SHALL preserve fail-fast and best-effort semantics
CA3 MUST preserve existing failure policy semantics for compaction failures:
- under `best_effort`, semantic failure MUST fallback to `truncate`;
- under `fail_fast`, semantic failure MUST terminate assembly immediately.

#### Scenario: Semantic failure under best-effort
- **WHEN** semantic compaction returns an error and policy is `best_effort`
- **THEN** CA3 falls back to `truncate` and continues with diagnostics fallback marker

#### Scenario: Semantic failure under fail-fast
- **WHEN** semantic compaction returns an error and policy is `fail_fast`
- **THEN** CA3 returns failure and aborts assembly before model execution

### Requirement: CA3 prune SHALL retain minimum evidence set before deletion
CA3 prune logic MUST retain a minimum evidence set based on configured keyword markers and recent-window constraints so that key intent and decision traces are not dropped during danger/emergency mitigation.

#### Scenario: Evidence marker matches candidate message
- **WHEN** prune evaluates a message that matches evidence retention markers
- **THEN** message is excluded from prune candidate set

#### Scenario: Recent window protection applies during emergency
- **WHEN** emergency prune evaluates recent messages inside configured retention window
- **THEN** those recent messages are retained unless explicitly marked removable by policy

### Requirement: CA3 compaction semantics SHALL remain equivalent between Run and Stream
For equivalent input context and effective config, CA3 compaction mode selection, fallback behavior, and evidence-retention outcomes MUST be semantically equivalent between Run and Stream paths.

#### Scenario: Equivalent Run and Stream with semantic mode
- **WHEN** equivalent requests are executed via Run and Stream in semantic mode
- **THEN** compaction mode, fallback semantics, and evidence retention outcomes are equivalent

#### Scenario: Equivalent Run and Stream with truncate fallback
- **WHEN** semantic compaction fails in both paths under `best_effort`
- **THEN** both paths fallback to truncate with semantically equivalent diagnostics markers

### Requirement: CA3 semantic quality gate SHALL support optional embedding similarity component
CA3 semantic quality evaluation MUST support an optional cosine-based embedding similarity component in addition to existing rule-based scoring, and MUST preserve rule-only compatibility when embedding scorer is disabled.

When reranker is enabled, CA3 semantic quality evaluation MUST include a deterministic reranker stage for final gate decision while preserving compatibility of existing thresholds when reranker is disabled.

#### Scenario: Hybrid+reranker mode enabled
- **WHEN** CA3 semantic compaction runs with embedding scorer and reranker enabled
- **THEN** quality evaluation uses rule signal and cosine similarity signal, then applies reranker before final gate decision

#### Scenario: Default hybrid mode without reranker
- **WHEN** embedding scorer is enabled and reranker is disabled by config
- **THEN** CA3 uses base hybrid scoring path and existing threshold semantics

#### Scenario: Rule-only compatibility mode
- **WHEN** CA3 semantic compaction runs with embedding scorer disabled
- **THEN** quality evaluation behaves equivalently to existing rule-only scoring path

### Requirement: CA3 semantic compaction SHALL expose embedding fallback diagnostics
CA3 semantic compaction MUST emit explicit diagnostics for embedding scorer and reranker path selection, including fallback reasons when adapter execution is unavailable, reranker execution fails, or reranker is bypassed.

#### Scenario: Adapter unavailable fallback
- **WHEN** embedding scorer is enabled but configured adapter is unavailable
- **THEN** CA3 records fallback diagnostics and applies policy-driven fallback behavior

#### Scenario: Reranker timeout fallback
- **WHEN** reranker request times out under `best_effort`
- **THEN** CA3 records reranker timeout fallback reason and continues with pre-reranker quality path

#### Scenario: Multi-provider adapter and reranker selection
- **WHEN** runtime config selects OpenAI, Gemini, or Anthropic embedding provider with reranker enabled
- **THEN** CA3 executes selected provider path and preserves equivalent fallback semantics

### Requirement: CA3 semantic compaction SHALL keep deterministic fallback chain with reranker enabled
CA3 semantic compaction MUST preserve deterministic fallback chain under reranker-enabled quality path:
`hybrid+reranker` -> `hybrid only` -> `rule-only` according to policy and failure reason.

#### Scenario: Best-effort falls back one step
- **WHEN** reranker fails but embedding scorer succeeds and policy is `best_effort`
- **THEN** CA3 falls back to hybrid-only path and continues compaction

#### Scenario: Best-effort falls back to rule-only
- **WHEN** both embedding scorer and reranker paths fail under `best_effort`
- **THEN** CA3 falls back to rule-only path and continues compaction

#### Scenario: Fail-fast aborts without fallback
- **WHEN** reranker or embedding scorer path fails and policy is `fail_fast`
- **THEN** CA3 aborts assembly before model execution and does not enter fallback chain

