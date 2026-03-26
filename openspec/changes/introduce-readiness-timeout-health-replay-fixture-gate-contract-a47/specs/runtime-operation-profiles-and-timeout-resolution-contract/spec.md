## ADDED Requirements

### Requirement: Timeout-resolution semantics SHALL align with composite replay fixtures
Timeout-resolution semantics MUST remain alignable with A47 composite replay fixtures, including precedence source, parent-budget convergence, and reject classification behavior.

Composite fixture assertions MUST cover:
- resolved timeout source (`profile|domain|request`),
- parent clamp behavior,
- exhausted-budget reject behavior.

#### Scenario: Composite fixture validates parent-budget clamp
- **WHEN** child resolved timeout exceeds parent remaining budget in fixture case
- **THEN** replay assertion confirms deterministic clamp output and canonical convergence metadata

#### Scenario: Composite fixture validates exhausted-budget reject
- **WHEN** parent remaining budget is non-positive in fixture case
- **THEN** replay assertion confirms deterministic reject classification and no child execution path
