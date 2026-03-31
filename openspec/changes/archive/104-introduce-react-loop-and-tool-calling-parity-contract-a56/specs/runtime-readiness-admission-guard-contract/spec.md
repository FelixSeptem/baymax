## ADDED Requirements

### Requirement: Admission guard SHALL enforce ReAct readiness findings before managed execution
Runtime readiness-admission guard MUST evaluate ReAct-related blocking findings before managed Run/Stream execution starts.

When ReAct blocking finding is present, admission decision MUST be `deny` and deny path MUST remain side-effect free.

#### Scenario: Managed Run is denied by ReAct blocking finding
- **WHEN** admission evaluates readiness output containing blocking `react.provider_tool_calling_unsupported`
- **THEN** admission decision is `deny` and runtime performs no scheduler enqueue, mailbox publish, or lifecycle mutation

#### Scenario: Managed Stream is denied by ReAct blocking finding
- **WHEN** admission evaluates readiness output containing blocking `react.stream_dispatch_unavailable`
- **THEN** admission decision is `deny` with semantically equivalent side-effect-free semantics

### Requirement: Degraded ReAct readiness SHALL follow policy-controlled admission mapping
When ReAct readiness status is `degraded`, admission guard MUST apply configured degraded policy:
- `allow_and_record`
- `fail_fast`

#### Scenario: Degraded ReAct finding with allow-and-record policy
- **WHEN** ReAct readiness is degraded and policy is `allow_and_record`
- **THEN** admission decision is `allow` and runtime records canonical degraded-admission markers

#### Scenario: Degraded ReAct finding with fail-fast policy
- **WHEN** ReAct readiness is degraded and policy is `fail_fast`
- **THEN** admission decision is `deny` with deterministic degraded-class reason

### Requirement: Run and Stream admission SHALL preserve ReAct reason taxonomy equivalence
For equivalent readiness input and policy, Run and Stream admission decisions MUST preserve equivalent ReAct reason taxonomy and explainability fields.

#### Scenario: Equivalent degraded ReAct admission under Run and Stream
- **WHEN** equivalent Run and Stream requests receive the same degraded ReAct readiness output
- **THEN** both paths produce semantically equivalent admission decision and reason taxonomy

#### Scenario: Equivalent blocked ReAct admission under Run and Stream
- **WHEN** equivalent Run and Stream requests receive the same blocked ReAct readiness output
- **THEN** both paths produce semantically equivalent deny classification and explainability output
