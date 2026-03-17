# security-event-governance-s3 Specification

## Purpose
TBD - created by archiving change introduce-security-s3-event-taxonomy-and-callback-alerting-e9. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL emit normalized S3 security event taxonomy
Runtime MUST emit a normalized S3 security event envelope for security decisions, including at minimum:
- `policy_kind` (`permission|rate_limit|io_filter`),
- `namespace_tool` (when applicable),
- `filter_stage` (`input|output` when applicable),
- `decision` (`allow|match|deny`),
- `reason_code`,
- `severity` (`low|medium|high`).

#### Scenario: Permission deny emits normalized security event
- **WHEN** tool execution is denied by permission policy
- **THEN** runtime emits a security event with normalized taxonomy fields and severity value

#### Scenario: I/O filter deny emits normalized security event
- **WHEN** model input or output path is denied by security filter
- **THEN** runtime emits a security event with `policy_kind=io_filter`, proper `filter_stage`, and normalized severity

### Requirement: Runtime SHALL trigger alerts only for deny decisions
Runtime MUST trigger S3 alert dispatch only when `decision=deny`.
`allow` and `match` decisions MUST NOT trigger alert callbacks.

#### Scenario: Deny decision triggers alert callback
- **WHEN** runtime emits a security event with `decision=deny`
- **THEN** registered callback sink is invoked with the normalized event payload

#### Scenario: Match decision does not trigger callback
- **WHEN** runtime emits a security event with `decision=match`
- **THEN** runtime records observability fields but does not invoke alert callback

### Requirement: Runtime SHALL support host-provided callback alert sink
Runtime MUST expose callback registration contract for security alerts and validate callback wiring during runtime setup.

#### Scenario: Host registers valid callback sink
- **WHEN** application registers a valid callback implementation
- **THEN** runtime accepts registration and dispatches deny alerts to callback sink

#### Scenario: Callback sink execution fails
- **WHEN** callback invocation returns error
- **THEN** runtime records alert delivery failure diagnostics and preserves existing security decision outcome

### Requirement: Run and Stream SHALL keep S3 security event semantic equivalence
For equivalent inputs and effective configuration, Run and Stream MUST produce semantically equivalent S3 security events and deny-alert behavior.

#### Scenario: Equivalent deny event in Run and Stream
- **WHEN** equivalent requests trigger the same deny decision in Run and Stream
- **THEN** emitted event taxonomy fields and callback trigger semantics remain equivalent

