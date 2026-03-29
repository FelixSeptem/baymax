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

### Requirement: S3 deny alerts SHALL use S4 managed delivery path
When `decision=deny`, runtime MUST dispatch callback alerts through S4 managed delivery executor instead of unmanaged direct invocation.
S4 delivery outcomes MUST be observable via delivery diagnostics fields.

#### Scenario: Deny callback is dispatched through managed executor
- **WHEN** runtime emits a deny security event
- **THEN** callback dispatch uses configured delivery mode, timeout, retry, and circuit-breaker policy

#### Scenario: Managed delivery failure preserves deny semantics
- **WHEN** managed delivery returns timeout/retry-exhausted/circuit-open result
- **THEN** runtime keeps original deny decision outcome and records delivery failure diagnostics

### Requirement: S3 non-deny events SHALL remain non-alerting under S4
S4 integration MUST NOT change existing deny-only alert trigger policy.
`allow|match` decisions MUST remain non-alerting regardless of delivery executor mode.

#### Scenario: Match event does not enter delivery executor
- **WHEN** runtime emits a security event with `decision=match`
- **THEN** runtime records observability fields only and does not enqueue callback delivery

### Requirement: Sandbox deny outcomes SHALL emit normalized S3 security events
Sandbox-driven deny outcomes MUST emit normalized S3 security events with canonical taxonomy fields and deterministic reason code mapping.

Sandbox taxonomy MUST include policy kind support for sandbox governance outcomes.

#### Scenario: Sandbox policy deny emits S3 event
- **WHEN** runtime denies tool execution by sandbox policy
- **THEN** runtime emits normalized S3 security event with sandbox policy kind and canonical deny reason code

#### Scenario: Sandbox fallback deny emits S3 event
- **WHEN** sandbox launch fails and fallback policy resolves to deny
- **THEN** runtime emits normalized S3 security event with canonical fallback-deny reason code

#### Scenario: Sandbox capability mismatch deny emits S3 event
- **WHEN** required sandbox capability is not satisfied and enforce path denies execution
- **THEN** runtime emits normalized S3 security event with canonical capability-mismatch reason code

### Requirement: Observe-mode sandbox outcomes SHALL remain non-alerting unless deny is enforced
Sandbox decisions observed under non-enforcing mode MUST preserve S3 deny-only trigger policy and MUST NOT trigger callback alerts unless effective decision is deny.

#### Scenario: Sandbox observe decision does not trigger callback
- **WHEN** sandbox mode is `observe` and observed decision is non-terminal
- **THEN** runtime records observability fields and does not dispatch deny alert callback

#### Scenario: Sandbox enforced deny triggers callback
- **WHEN** sandbox mode is `enforce` and effective decision is deny
- **THEN** runtime dispatches deny-only callback alert through managed delivery path

