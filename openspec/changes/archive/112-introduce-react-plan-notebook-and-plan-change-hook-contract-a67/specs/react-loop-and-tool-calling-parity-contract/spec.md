## ADDED Requirements

### Requirement: ReAct Plan Notebook Run-Stream Semantic Equivalence
For equivalent input, configuration, and dependency state, Run and Stream SHALL preserve semantically equivalent plan notebook lifecycle outcomes, including action sequence class, version progression, and terminal plan status.

#### Scenario: Equivalent requests perform plan revision
- **WHEN** equivalent Run and Stream flows trigger a plan revision in the same logical step
- **THEN** both paths MUST expose semantically equivalent `plan_version` progression and `revise` action classification

#### Scenario: Equivalent requests complete with same plan terminal
- **WHEN** equivalent Run and Stream flows complete the same plan branch
- **THEN** both paths MUST expose semantically equivalent plan terminal status and completion action semantics

### Requirement: Plan Change MUST NOT Introduce Parallel ReAct Loop Semantics
Plan notebook and plan-change hook execution MUST be implemented within canonical ReAct loop boundaries and MUST NOT introduce parallel loop taxonomy or separate termination channels.

#### Scenario: Plan notebook enabled with ReAct loop
- **WHEN** notebook and plan-change hooks are enabled
- **THEN** runtime MUST continue using canonical A56 ReAct termination taxonomy for final classification

#### Scenario: Plan change hook failure classification
- **WHEN** plan-change hook returns failure
- **THEN** runtime MUST map outcome to existing loop-compatible classification without introducing alternate loop family
