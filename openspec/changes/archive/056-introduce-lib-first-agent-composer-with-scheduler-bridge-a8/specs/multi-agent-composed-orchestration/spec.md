## ADDED Requirements

### Requirement: Composed orchestration SHALL expose a composer-first integration path
Composed orchestration MUST define a first-class composer integration contract so workflow, teams, and A2A cooperation can be consumed through a single runtime entrypoint instead of host-side manual composition.

#### Scenario: Workflow and Teams run through composer entrypoint
- **WHEN** host invokes composed orchestration through the composer package
- **THEN** workflow and teams orchestration semantics remain available without requiring custom host glue code

### Requirement: Composed orchestration SHALL preserve existing reason namespace contract
Composer-managed composed flows MUST continue using existing namespaced timeline reasons (`team.*`, `workflow.*`, `a2a.*`, `scheduler.*`, `subagent.*`) and MUST NOT introduce non-namespaced reasons in multi-agent paths.

#### Scenario: Composer emits timeline events in composed path
- **WHEN** composed orchestration emits timeline events under composer management
- **THEN** each multi-agent reason remains in the existing canonical namespace set and remains correlation-compatible with shared contract checks
