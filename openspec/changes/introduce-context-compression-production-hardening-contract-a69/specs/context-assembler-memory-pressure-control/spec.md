## ADDED Requirements

### Requirement: A69 SHALL Preserve A67-CTX Semantic Invariants While Hardening Production Behavior
A69 changes to context compression MUST preserve A67-CTX semantic invariants (`reference-first`, `isolate handoff`, `edit gate`, `swap-back`, `lifecycle tiering`) and MUST NOT introduce a parallel semantic model.

#### Scenario: Compression hardening does not redefine context semantics
- **WHEN** A69 production-hardening logic is enabled
- **THEN** A67-CTX contract outputs remain semantically equivalent for equivalent inputs

### Requirement: A69 SHALL Harden Semantic Compaction Quality and Fallback Determinism
Context semantic compaction MUST expose deterministic quality gating and fallback behavior with explicit outcome classes (`applied`, `degraded`, `fallback`, `failed`).

#### Scenario: Semantic compaction falls back under best-effort
- **WHEN** semantic compaction quality or execution fails under `best_effort`
- **THEN** assembler falls back deterministically to allowed rule-based path and emits stable fallback classification

#### Scenario: Semantic compaction fails under fail-fast
- **WHEN** semantic compaction fails under `fail_fast`
- **THEN** assembly aborts before model execution without partial-state corruption

### Requirement: A69 SHALL Define Rule-Based Compression Eligibility for Tool Result History
Rule-based compression eligibility MUST explicitly govern tool result history, including oldest tool-call result items, and MUST preserve minimum evidence required for replay and decision-trace reconstruction.

#### Scenario: Oldest tool result is eligible for compaction
- **WHEN** oldest tool result satisfies configured eligibility and evidence-retention constraints
- **THEN** assembler MAY compact or remove it deterministically with provenance markers preserved

#### Scenario: Tool result is protected from compaction
- **WHEN** tool result is marked critical/immutable or required by evidence-retention policy
- **THEN** assembler MUST retain the entry and exclude it from compaction candidates

### Requirement: A69 SHALL Govern Lifecycle Tiering and Swap-Back Retrieval Deterministically
Context lifecycle transitions across `hot|warm|cold|pruned` and swap-back retrieval MUST be deterministic and auditable. Swap-back ranking MUST prioritize relevance before recency with deterministic tie-breaks.

#### Scenario: Swap-back ranking uses relevance then recency
- **WHEN** multiple cold candidates are eligible for swap-back
- **THEN** assembler selects candidates by canonical relevance-first ranking and deterministic recency tie-break

#### Scenario: Tier transition conflict resolves deterministically
- **WHEN** multiple lifecycle transition conditions are simultaneously satisfied
- **THEN** assembler applies canonical transition precedence and records transition reason

### Requirement: A69 SHALL Enforce File Cold-Store Lifecycle Governance
File cold-store backend MUST enforce retention/quota/cleanup/compact governance to prevent unbounded growth and keep recoverability stable.

#### Scenario: Cold-store exceeds configured quota
- **WHEN** cold-store size exceeds configured quota threshold
- **THEN** cleanup/compaction policy executes deterministically and records governance actions

#### Scenario: Cold-store file is partially written or corrupted
- **WHEN** assembler detects malformed cold-store segment during load or cleanup
- **THEN** recovery behavior follows deterministic fail-safe policy without introducing second source-of-truth state

### Requirement: A69 Recovery SHALL Be Idempotent Across Crash Restart Replay
Crash/restart/replay flows MUST preserve idempotent spill/swap-back semantics and MUST avoid duplicate restores, duplicate counters, or torn state transitions.

#### Scenario: Restart resumes after spill and swap-back
- **WHEN** runtime restarts after spill and before full swap-back completion
- **THEN** resumed execution converges to deterministic state without duplicate swap-back side effects

#### Scenario: Replay validates equivalent recovery behavior
- **WHEN** equivalent fixture input is replayed after recovery-related events
- **THEN** replay output remains semantically equivalent to canonical recovery expectation
