## MODIFIED Requirements

### Requirement: Boundary governance outcomes SHALL be reflected in architecture docs
When module responsibility corrections are made, architecture and boundary documentation MUST be updated in the same change to preserve a single source of truth.

For R4 multi-agent scope, `docs/multi-agent-identifier-model.md` MUST be treated as the shared contract source for identifier, status mapping, and reason namespace conventions.

#### Scenario: Multi-agent contract is updated
- **WHEN** teams/workflow/a2a shared identifier or reason conventions change
- **THEN** architecture and shared contract docs are updated in the same change set

### Requirement: Boundary reviews SHALL validate dependency and semantic direction together
Boundary checks MUST include both import-direction validation and semantic responsibility validation for cross-module orchestration paths.

For R4 multi-agent domains, boundary governance MUST include a blocking shared-contract consistency gate. Changes touching Teams/Workflow/A2A specs or implementation MUST pass this gate before merge.

Minimum gate checks for this milestone:
- unified status mapping compliance (including `a2a.submitted -> pending`),
- reason code namespace compliance (`team.*|workflow.*|a2a.*`),
- canonical `peer_id` naming compliance.

#### Scenario: Multi-agent change violates reason namespace
- **WHEN** a change introduces reason code outside approved multi-agent namespaces
- **THEN** shared-contract gate fails and the change is blocked from merge

#### Scenario: Multi-agent change uses non-canonical A2A peer field
- **WHEN** a change emits remote peer field as non-`peer_id` key
- **THEN** shared-contract gate fails and the change is blocked from merge
