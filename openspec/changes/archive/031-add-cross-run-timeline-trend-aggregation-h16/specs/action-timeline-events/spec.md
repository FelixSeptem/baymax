## ADDED Requirements

### Requirement: Action timeline cross-run trend semantics SHALL preserve Run/Stream equivalence
Cross-run trend aggregation derived from Action Timeline events MUST preserve semantic equivalence between Run and Stream for equivalent workloads.

This requirement applies to `phase + status` aggregate distributions and latency summary semantics, including `latency_p95_ms`.

#### Scenario: Equivalent workload compared between Run and Stream
- **WHEN** equivalent requests are executed through Run and Stream within the same trend window
- **THEN** trend aggregates for `phase + status` and latency summaries are semantically equivalent

### Requirement: Action timeline trend output SHALL support phase and status dimensions simultaneously
Trend aggregation output derived from timeline events MUST support combined `phase + status` grouping rather than phase-only output.

#### Scenario: Consumer inspects failed tool-phase trends
- **WHEN** trend output is requested for a window containing mixed outcomes
- **THEN** consumer can distinguish `tool + failed` from other phase/status combinations

#### Scenario: Consumer inspects canceled hitl-phase trends
- **WHEN** trend output includes canceled clarification/gate paths
- **THEN** consumer can identify `hitl + canceled` as a distinct aggregate bucket
