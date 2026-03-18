## ADDED Requirements

### Requirement: Runtime config SHALL define composer consumption boundaries
Runtime config contract MUST define how composer consumes `teams.*`, `workflow.*`, `a2a.*`, `scheduler.*`, and `subagent.*` snapshots, including explicit next-attempt-only semantics for scheduler/subagent reload-sensitive fields.

#### Scenario: Host updates scheduler and subagent config via hot reload
- **WHEN** scheduler/subagent fields are reloaded during runtime
- **THEN** composer applies updated values on next-attempt boundaries and preserves deterministic behavior for in-flight attempts

### Requirement: Run diagnostics SHALL expose composer and scheduler-fallback additive markers
Run diagnostics MUST expose additive markers for composer-managed execution and scheduler backend fallback outcomes, while preserving backward compatibility with nullable defaults.

#### Scenario: Composer run triggers scheduler backend fallback
- **WHEN** a composer-managed run starts with scheduler fallback-to-memory
- **THEN** run summary includes additive fallback markers (`composer_managed`, `scheduler_backend_fallback`, `scheduler_backend_fallback_reason`) and legacy consumers can safely ignore absent fields without behavior change

### Requirement: New diagnostics fields SHALL remain in compatibility window
Any A8 additive diagnostics fields MUST follow the existing compatibility window contract (`additive + nullable + default`) and MUST NOT alter pre-A8 field semantics.

#### Scenario: Legacy consumer parses A8 run summary
- **WHEN** a legacy consumer reads run summaries produced after A8 rollout
- **THEN** existing fields retain previous meaning and newly added A8 fields are optional with nullable/default fallback behavior
