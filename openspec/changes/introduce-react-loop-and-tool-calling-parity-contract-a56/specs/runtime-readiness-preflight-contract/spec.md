## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate ReAct prerequisite availability
Runtime readiness preflight MUST evaluate ReAct prerequisite dependencies when `runtime.react.enabled=true`.

Canonical finding codes for this milestone MUST include:
- `react.loop_disabled`
- `react.stream_dispatch_unavailable`
- `react.provider_tool_calling_unsupported`
- `react.tool_registry_unavailable`
- `react.sandbox_dependency_unavailable`

#### Scenario: ReAct enabled but provider lacks tool-calling capability
- **WHEN** preflight evaluates effective config with ReAct enabled and active provider cannot satisfy tool-calling requirements
- **THEN** readiness returns canonical finding `react.provider_tool_calling_unsupported`

#### Scenario: ReAct enabled but Stream dispatch dependency is unavailable
- **WHEN** preflight evaluates managed Stream path and stream dispatch prerequisite is not available
- **THEN** readiness returns canonical finding `react.stream_dispatch_unavailable`

### Requirement: ReAct readiness findings SHALL preserve strict and non-strict mapping semantics
Readiness preflight MUST apply existing strict/non-strict mapping for ReAct findings:
- non-strict mode MAY classify recoverable ReAct findings as `degraded`,
- strict mode MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict mode with recoverable tool-registry finding
- **WHEN** preflight detects recoverable `react.tool_registry_unavailable` and `runtime.readiness.strict=false`
- **THEN** readiness status is `degraded` with canonical ReAct finding

#### Scenario: Strict mode with equivalent ReAct finding
- **WHEN** preflight evaluates equivalent finding with `runtime.readiness.strict=true`
- **THEN** readiness status escalates to `blocked`

### Requirement: ReAct readiness output SHALL remain deterministic for equivalent snapshots
For equivalent runtime snapshot and dependency state, readiness preflight MUST return semantically equivalent ReAct status and finding codes.

#### Scenario: Repeated preflight with unchanged ReAct dependencies
- **WHEN** host calls readiness preflight repeatedly without config or dependency changes
- **THEN** ReAct-related status and finding-code semantics remain equivalent

#### Scenario: Regression emits non-canonical ReAct finding code
- **WHEN** implementation outputs ReAct finding code outside canonical taxonomy
- **THEN** contract validation fails and blocks merge
