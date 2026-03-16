## ADDED Requirements

### Requirement: Runtime SHALL expose CA3 compaction strategy config with deterministic precedence
Runtime configuration MUST expose CA3 compaction strategy fields with precedence `env > file > default`, including:
- `context_assembler.ca3.compaction.mode` (`truncate|semantic`)
- semantic compaction timeout controls
- evidence retention controls (keywords and recent-window constraints)

Invalid enum, timeout, or retention parameters MUST fail fast on startup and hot reload.

#### Scenario: Startup with default CA3 compaction config
- **WHEN** runtime starts without explicit CA3 compaction strategy overrides
- **THEN** effective config resolves to `truncate` mode and valid default retention controls

#### Scenario: Startup with semantic compaction overrides
- **WHEN** CA3 compaction strategy is set in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid CA3 compaction enum
- **WHEN** runtime config contains unsupported compaction mode
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime diagnostics SHALL expose minimal CA3 compaction observability fields
Run diagnostics MUST expose minimal CA3 compaction fields:
- `ca3_compaction_mode`
- `ca3_compaction_fallback`
- `ca3_compaction_retained_evidence_count`

These fields MUST be additive and backward-compatible for existing diagnostics consumers.

#### Scenario: Consumer inspects diagnostics with semantic compaction success
- **WHEN** a run completes with semantic compaction applied
- **THEN** diagnostics include mode=`semantic`, fallback=`false`, and non-negative retained evidence count

#### Scenario: Consumer inspects diagnostics with semantic fallback
- **WHEN** semantic compaction fails under best-effort and truncate fallback is used
- **THEN** diagnostics include mode=`semantic`, fallback=`true`, and non-negative retained evidence count

### Requirement: Runtime diagnostics contract SHALL preserve CA3 compaction semantics across Run and Stream
Diagnostics payload semantics for CA3 compaction MUST remain equivalent between Run and Stream for equivalent inputs, including mode selection, fallback marker, and retained evidence count.

#### Scenario: Equivalent Run and Stream semantic execution
- **WHEN** equivalent requests execute through Run and Stream with same CA3 config
- **THEN** emitted compaction diagnostics fields are semantically equivalent

#### Scenario: Equivalent Run and Stream fallback execution
- **WHEN** semantic compaction fails in both paths under best-effort
- **THEN** both paths emit semantically equivalent fallback diagnostics
