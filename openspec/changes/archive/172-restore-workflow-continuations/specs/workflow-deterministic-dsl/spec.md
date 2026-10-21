## ADDED Requirements

### Requirement: Workflow pause reasons SHALL have source-owned safe points

For a workflow form whose manifest declares support, the runtime SHALL preserve `input_required`, `approval_required`, and `budget_exhausted` at source-owned graph safe points. Input pause SHALL retain pending input and graph dependency state; approval pause SHALL occur before the pending effect dispatch; budget pause SHALL retain consumed resources and the original deadline. A deny or approval timeout SHALL produce source terminal evidence without dispatching the pending effect.

#### Scenario: Approval pause prevents effect dispatch
- **WHEN** a declared workflow approval boundary is reached for a pending effect
- **THEN** the workflow persists the pending action before dispatch
- **AND** denial or timeout records the source terminal outcome without executing that effect

### Requirement: Restored workflow execution SHALL preserve deterministic graph semantics

After a valid restore, the runtime SHALL schedule only eligible nonterminal steps using the original normalized graph and deterministic ordering rules. Completed step outcomes, retry/timeout facts, dependency edges, and effect references SHALL remain part of the same graph continuation. Restore SHALL NOT repeat a completed step merely to reconstruct state.

#### Scenario: Expanded graph continues after partial completion
- **WHEN** an expanded workflow graph restores after some aliases have terminal step outcomes
- **THEN** only remaining eligible expanded steps are scheduled
- **AND** the aggregate workflow outcome remains deterministic

### Requirement: Workflow Run and Stream restore SHALL remain equivalent

Equivalent Run and Stream workflow checkpoints and resume requests SHALL expose semantically equivalent source identity, pause boundary, step transitions, retry/timeout precedence, terminal classification, and duplicate projection, apart from transport representation.

#### Scenario: Run and Stream restore the same pause
- **WHEN** equivalent workflow Run and Stream executions restore the same declared pause reason after source reopen
- **THEN** their graph facts and terminal outcomes are semantically equivalent
