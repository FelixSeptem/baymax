## ADDED Requirements

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
