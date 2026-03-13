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

