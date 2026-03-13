## ADDED Requirements

### Requirement: Boundary reviews SHALL verify context-model responsibility split
Boundary governance reviews MUST verify that `context/*` packages orchestrate policy only, while provider SDK protocol actions remain in `model/*` packages and are consumed via interfaces.

#### Scenario: Token counting path is reviewed
- **WHEN** reviewer inspects context assembly token-count flow
- **THEN** context layer invokes model-facing interfaces and does not import provider SDK packages directly

### Requirement: Boundary reviews SHALL validate dependency and semantic direction together
Boundary checks MUST include both import-direction validation and semantic responsibility validation for cross-module orchestration paths.

#### Scenario: Static dependency check passes but semantic ownership drifts
- **WHEN** review detects behavior implemented in the wrong module despite legal imports
- **THEN** the change is not accepted until ownership is moved back to the designated module

### Requirement: Boundary governance outcomes SHALL be reflected in architecture docs
When module responsibility corrections are made, architecture and boundary documentation MUST be updated in the same change to preserve a single source of truth.

#### Scenario: Runtime boundary fix is merged
- **WHEN** a change modifies cross-module responsibilities
- **THEN** `docs/runtime-module-boundaries.md` and related docs are updated before completion
