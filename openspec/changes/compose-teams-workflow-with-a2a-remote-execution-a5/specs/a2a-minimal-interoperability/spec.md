## ADDED Requirements

### Requirement: A2A SHALL expose orchestration-consumable correlation contract
A2A execution used by orchestration modules MUST preserve and expose correlation metadata required by composed flows, including `workflow_id`, `team_id`, `step_id`, and `task_id` when provided.

#### Scenario: Orchestration passes cross-domain correlation metadata
- **WHEN** workflow or teams dispatches an A2A remote task with correlation metadata
- **THEN** A2A path preserves metadata for timeline and diagnostics mapping

### Requirement: A2A orchestration integration SHALL preserve normalized terminal semantics
When consumed by orchestration modules, A2A terminal outcomes MUST remain normalized and deterministic under retry, timeout, and cancellation paths.

#### Scenario: Remote call times out under orchestration path
- **WHEN** composed orchestration invokes A2A and remote call exceeds timeout budget
- **THEN** A2A returns normalized timeout-class outcome and deterministic error-layer mapping

### Requirement: A2A orchestration integration SHALL preserve MCP boundary separation
A2A orchestration integration MUST NOT redefine MCP tool-integration responsibilities and MUST keep peer collaboration semantics inside A2A domain.

#### Scenario: Composed path includes both remote collaboration and tool invocation
- **WHEN** one run includes A2A peer delegation and MCP tool calls
- **THEN** A2A handles peer lifecycle semantics while MCP handles tool semantics without namespace overlap
