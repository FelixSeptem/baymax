## ADDED Requirements

### Requirement: ReAct JIT Context Organization SHALL Preserve Run-Stream Semantic Equivalence
For equivalent input, configuration, and dependency state, Run and Stream MUST preserve semantically equivalent JIT context outcomes, including:
- reference discovery/resolution counts,
- edit-gate allow/deny decision class,
- swap-back relevance decision class,
- lifecycle tier transition class,
- recap source classification.

#### Scenario: Equivalent requests resolve references consistently
- **WHEN** equivalent Run and Stream requests execute reference-first assembly in the same logical step
- **THEN** both paths MUST expose semantically equivalent reference discover/resolve outcome classes

#### Scenario: Equivalent requests evaluate edit gate consistently
- **WHEN** equivalent Run and Stream requests evaluate `clear_at_least` thresholds
- **THEN** both paths MUST expose semantically equivalent edit-gate decision class and final context semantics

### Requirement: JIT Context Organization MUST NOT Introduce Parallel ReAct Semantics
JIT context organization MUST be implemented inside canonical ReAct loop boundaries and MUST NOT introduce:
- parallel loop taxonomy,
- alternate termination channels,
- or divergent decision-explainability fields.

#### Scenario: JIT context features enabled during ReAct execution
- **WHEN** JIT context organization features are enabled
- **THEN** runtime MUST continue using canonical A56 termination taxonomy and A58 decision-trace semantics

#### Scenario: Context organization failure maps to existing runtime families
- **WHEN** JIT context organization operation fails
- **THEN** runtime MUST classify failure through existing loop-compatible families without creating parallel taxonomy
