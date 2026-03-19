## ADDED Requirements

### Requirement: Teams runtime SHALL consume unified delegation primitive for remote workers
Teams orchestration MUST consume unified collaboration delegation primitive for remote-worker execution instead of module-local divergent delegation semantics.

#### Scenario: Team plan includes remote worker tasks
- **WHEN** teams engine dispatches remote worker tasks
- **THEN** remote dispatch path uses shared delegation primitive semantics and preserves normalized lifecycle outcomes

### Requirement: Teams runtime SHALL support handoff primitive across role transitions
Teams orchestration MUST support explicit handoff primitive semantics for role transition flows between leader/coordinator/worker.

#### Scenario: Coordinator hands off unresolved task to another worker
- **WHEN** team coordination requires handoff from current owner to another worker
- **THEN** runtime records deterministic task ownership transition and preserves terminal convergence semantics

### Requirement: Teams aggregation semantics SHALL align with collaboration primitive strategies
Teams aggregation behavior for remote/local mixed tasks MUST align with collab strategy semantics (`all_settled` and `first_success`) while preserving Run/Stream equivalence.

#### Scenario: Mixed local remote team run with first_success strategy
- **WHEN** teams engine runs mixed targets using `first_success` aggregation mode
- **THEN** aggregate team terminal classification remains semantically equivalent across Run and Stream
