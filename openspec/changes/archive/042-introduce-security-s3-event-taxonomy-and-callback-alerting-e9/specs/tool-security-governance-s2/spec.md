## ADDED Requirements

### Requirement: Tool-security deny outcomes SHALL map to S3 security-event taxonomy
Permission-denied and rate-limit-denied outcomes from tool governance MUST map to S3 security-event taxonomy with normalized `reason_code` and `severity`.

#### Scenario: Permission deny maps to S3 event
- **WHEN** runtime denies tool execution by permission policy
- **THEN** emitted S3 security event includes policy kind, selector context, normalized reason code, and severity

#### Scenario: Rate-limit deny maps to S3 event
- **WHEN** runtime denies tool execution due to process-scoped rate limit
- **THEN** emitted S3 security event includes policy kind, selector context, normalized reason code, and severity

### Requirement: Tool-security deny outcomes SHALL trigger deny-only callback alerts
Tool governance deny outcomes MUST invoke registered callback alert sink and non-deny outcomes MUST NOT invoke callback.

#### Scenario: Tool permission deny triggers callback
- **WHEN** permission decision is `deny`
- **THEN** runtime dispatches callback alert with normalized S3 event payload

#### Scenario: Tool allow does not trigger callback
- **WHEN** permission decision is `allow`
- **THEN** runtime does not dispatch callback alert
