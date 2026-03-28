## ADDED Requirements

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
