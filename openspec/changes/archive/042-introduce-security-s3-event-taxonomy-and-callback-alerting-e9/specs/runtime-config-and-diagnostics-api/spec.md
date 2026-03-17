## ADDED Requirements

### Requirement: Runtime config SHALL expose S3 security-event and alert callback controls with deterministic precedence
Runtime configuration MUST expose S3 security-event controls with precedence `env > file > default`, including event enablement, deny-alert trigger policy, severity mapping controls, and callback registration constraints.

Invalid S3 event config values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default S3 event config
- **WHEN** runtime starts without explicit S3 security-event overrides
- **THEN** effective config resolves valid defaults and deny-alert policy

#### Scenario: Invalid S3 event config update arrives
- **WHEN** watched config changes to malformed S3 event settings
- **THEN** runtime rejects update and preserves previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive S3 security-event fields
Runtime diagnostics MUST expose additive S3 event fields at minimum:
- `policy_kind`,
- `namespace_tool`,
- `filter_stage`,
- `decision`,
- `reason_code`,
- `severity`,
- alert-dispatch status marker.

These fields MUST remain backward-compatible with existing consumers.

#### Scenario: Consumer inspects deny alert diagnostics
- **WHEN** runtime dispatches a deny alert callback
- **THEN** diagnostics include S3 taxonomy fields and alert-dispatch status

#### Scenario: Consumer inspects callback failure diagnostics
- **WHEN** callback dispatch fails
- **THEN** diagnostics include failure marker and normalized failure reason without changing deny decision outcome

### Requirement: Run and Stream SHALL preserve S3 diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent S3 diagnostics payload fields.

#### Scenario: Equivalent S3 diagnostics in Run and Stream
- **WHEN** equivalent deny decisions occur in Run and Stream
- **THEN** diagnostics include equivalent S3 taxonomy and alert-dispatch semantics
