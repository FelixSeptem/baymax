# tool-security-governance-s2 Specification

## Purpose
TBD - created by archiving change harden-security-s2-tool-permission-rate-limit-and-io-filter-e8. Update Purpose after archive.
## Requirements
### Requirement: Runner SHALL enforce namespace+tool permission policy before tool dispatch
The runtime MUST evaluate tool permission policy keyed by `namespace+tool` before executing a tool call.

If a matching policy resolves to `deny`, execution MUST be fail-fast denied and the tool MUST NOT be dispatched.

#### Scenario: Permission allow for matched namespace+tool
- **WHEN** a tool call matches configured `namespace+tool` policy with `allow`
- **THEN** runtime permits tool dispatch and continues normal execution flow

#### Scenario: Permission deny for matched namespace+tool
- **WHEN** a tool call matches configured `namespace+tool` policy with `deny`
- **THEN** runtime fail-fast denies execution and does not dispatch the tool

### Requirement: Runner SHALL enforce process-scoped rate limit on namespace+tool
The runtime MUST enforce rate limits using process-scoped counters keyed by `namespace+tool` and configured time windows.

When rate limit is exceeded, runtime MUST return `deny` outcome and MUST NOT execute the tool call.

#### Scenario: Tool call within configured quota
- **WHEN** call count for a `namespace+tool` key remains within configured limit for the current window
- **THEN** runtime allows tool dispatch

#### Scenario: Tool call exceeds configured quota
- **WHEN** call count for a `namespace+tool` key exceeds configured limit in the current window
- **THEN** runtime returns fail-fast `deny` and blocks tool dispatch

### Requirement: Tool security governance SHALL emit normalized deny diagnostics
Permission-denied and rate-limit-denied outcomes MUST emit normalized diagnostics fields and reason codes for audit and triage.

#### Scenario: Permission deny is recorded
- **WHEN** runtime denies a tool call by permission policy
- **THEN** diagnostics include policy type `permission`, matched `namespace+tool`, and normalized deny reason code

#### Scenario: Rate limit deny is recorded
- **WHEN** runtime denies a tool call due to rate-limit exceedance
- **THEN** diagnostics include policy type `rate_limit`, matched `namespace+tool`, window context, and normalized deny reason code

### Requirement: Run and Stream SHALL keep tool-security governance semantic equivalence
For equivalent input and effective configuration, Run and Stream MUST produce semantically equivalent permission and rate-limit decisions.

#### Scenario: Equivalent permission deny in Run and Stream
- **WHEN** equivalent requests hit the same deny permission policy in both Run and Stream
- **THEN** both paths deny tool execution with semantically equivalent outcome and diagnostics semantics

#### Scenario: Equivalent rate-limit deny in Run and Stream
- **WHEN** equivalent request patterns exceed the same process-scoped rate limit in both Run and Stream
- **THEN** both paths deny tool execution with semantically equivalent outcome and diagnostics semantics

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

### Requirement: Tool-security governance SHALL compose with sandbox action policy deterministically
Tool governance and sandbox action policy MUST compose with deterministic precedence under equivalent effective configuration.

At minimum:
- permission deny MUST remain terminal deny,
- rate-limit deny MUST remain terminal deny,
- sandbox action resolution MUST apply only after governance allow path.

#### Scenario: Permission deny short-circuits sandbox resolution
- **WHEN** tool governance resolves permission decision to `deny`
- **THEN** runtime denies execution without evaluating sandbox action path

#### Scenario: Governance allow proceeds to sandbox action
- **WHEN** tool governance resolves permission and rate-limit checks as allow
- **THEN** runtime proceeds to sandbox action resolution for final execution path

### Requirement: Sandbox deny outcomes SHALL align with security-event taxonomy
Sandbox-driven deny outcomes MUST emit normalized security-event taxonomy compatible with existing S3/S4 delivery semantics.

#### Scenario: Sandbox policy deny emits normalized security event
- **WHEN** runtime denies a tool call by sandbox policy
- **THEN** emitted security event includes canonical policy kind, selector context, normalized reason code, and severity

#### Scenario: Sandbox fallback deny emits normalized security event
- **WHEN** runtime denies execution due to sandbox launch failure and deny fallback policy
- **THEN** emitted security event includes canonical fallback deny reason code and dispatch semantics

### Requirement: High-risk sandbox fallback SHALL default to deny
For high-risk selector baseline, sandbox fallback policy MUST default to `deny` unless explicitly overridden by per-selector configuration.

High-risk selector baseline for this milestone:
- `local+shell`
- `local+process_exec`
- `local+fs_write`
- `mcp+stdio_command`

#### Scenario: High-risk selector without explicit fallback override
- **WHEN** sandbox launch fails for a high-risk selector and no explicit override exists
- **THEN** runtime denies execution deterministically

#### Scenario: High-risk selector with explicit allow override
- **WHEN** sandbox launch fails for a high-risk selector and explicit `allow_and_record` override is configured
- **THEN** runtime executes host fallback and records override metadata for audit

