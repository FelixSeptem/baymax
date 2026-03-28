## ADDED Requirements

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
