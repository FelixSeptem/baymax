## ADDED Requirements

### Requirement: Runtime Config SHALL Expose A69 Context Compression Governance Fields
Runtime configuration MUST expose A69 governance fields for semantic compaction quality, rule-based eligibility, swap-back ranking, and cold-store lifecycle controls with precedence `env > file > default`.

At minimum, A69 config coverage MUST include:
- semantic compaction quality thresholds and fallback policy controls,
- rule-based eligibility controls for tool-result history compaction,
- swap-back ranking strategy and candidate window controls,
- cold-store retention/quota/cleanup/compact policy controls.

#### Scenario: Environment overrides file for A69 fields
- **WHEN** the same A69 governance field is set in env and file
- **THEN** runtime resolves effective value using `env > file > default`

#### Scenario: Invalid A69 governance config fails fast
- **WHEN** startup or hot reload includes malformed A69 governance values
- **THEN** runtime rejects update fail-fast and atomically rolls back to previous valid snapshot

### Requirement: Runtime Diagnostics SHALL Expose A69 Additive Governance Fields
Run diagnostics MUST expose additive A69 governance fields for compaction outcome class, tier transition reason, swap-back ranking metadata, cold-store governance actions, and recovery consistency markers.

A69 diagnostics fields MUST remain backward-compatible under `additive + nullable + default` rules.

#### Scenario: A69 diagnostics are present after compression governance path
- **WHEN** a run triggers A69 semantic/rule-based compression and lifecycle governance
- **THEN** diagnostics include A69 additive fields without breaking existing parsers

#### Scenario: A69 fields absent in legacy runs
- **WHEN** a run executes without A69 paths activated
- **THEN** diagnostics remain valid with nullable/default semantics and no schema breakage

### Requirement: Runtime Diagnostics SHALL Preserve Run Stream Equivalence for A69 Fields
For equivalent inputs and effective config, A69 diagnostics semantics MUST remain equivalent between Run and Stream paths.

#### Scenario: Equivalent Run and Stream produce matching A69 governance diagnostics
- **WHEN** equivalent requests trigger A69 governance behaviors in Run and Stream
- **THEN** A69 diagnostics semantics remain equivalent, including compaction outcome and recovery markers
