## ADDED Requirements

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
