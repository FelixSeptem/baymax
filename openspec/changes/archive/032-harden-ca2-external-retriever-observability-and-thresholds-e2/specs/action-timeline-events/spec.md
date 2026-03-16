## ADDED Requirements

### Requirement: CA2 external observability threshold semantics SHALL remain equivalent between Run and Stream
For equivalent workloads, CA2 external retriever threshold-hit signals and provider trend diagnostics semantics MUST be equivalent between Run and Stream paths.

#### Scenario: Equivalent workload produces threshold hit in Run and Stream
- **WHEN** equivalent requests executed via Run and Stream both exceed the same CA2 external threshold
- **THEN** emitted threshold semantics and resulting diagnostics aggregates are semantically equivalent

#### Scenario: Equivalent workload remains below threshold in Run and Stream
- **WHEN** equivalent requests executed via Run and Stream stay under configured CA2 external thresholds
- **THEN** both paths expose semantically equivalent no-hit diagnostics outcomes
