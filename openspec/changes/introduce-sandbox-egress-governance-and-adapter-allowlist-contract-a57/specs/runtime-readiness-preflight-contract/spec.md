## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate sandbox egress policy safety findings
Runtime readiness preflight MUST evaluate effective sandbox egress policy and emit canonical findings for unsafe or invalid configurations.

Canonical finding codes for this milestone MUST include:
- `sandbox.egress.policy_invalid`
- `sandbox.egress.allowlist_invalid`
- `sandbox.egress.rule_conflict`
- `sandbox.egress.violation_budget_breached`

#### Scenario: Preflight detects malformed egress allowlist
- **WHEN** preflight evaluates effective config with malformed egress allowlist entries
- **THEN** readiness emits canonical `sandbox.egress.allowlist_invalid` finding

#### Scenario: Preflight detects egress violation budget breach
- **WHEN** preflight evaluates runtime state with egress violation budget in breached state
- **THEN** readiness emits canonical `sandbox.egress.violation_budget_breached` finding

### Requirement: Readiness preflight SHALL evaluate adapter allowlist readiness findings
Runtime readiness preflight MUST evaluate adapter allowlist activation readiness and emit canonical findings.

Canonical finding codes for this milestone MUST include:
- `adapter.allowlist.missing_entry`
- `adapter.allowlist.signature_invalid`
- `adapter.allowlist.policy_conflict`

#### Scenario: Preflight detects missing allowlist entry for required adapter
- **WHEN** required adapter metadata has no matching allowlist entry
- **THEN** readiness emits canonical `adapter.allowlist.missing_entry` finding

#### Scenario: Preflight detects invalid signature state under enforce mode
- **WHEN** adapter signature state is invalid and enforcement mode requires blocking
- **THEN** readiness emits canonical `adapter.allowlist.signature_invalid` finding

### Requirement: Egress and allowlist findings SHALL preserve strict and non-strict mapping semantics
Readiness preflight MUST map A57 findings using existing strict/non-strict rules:
- non-strict mode MAY classify recoverable findings as `degraded`,
- strict mode MUST escalate equivalent blocking findings to `blocked`.

#### Scenario: Non-strict mode with recoverable egress policy conflict
- **WHEN** preflight detects recoverable `sandbox.egress.rule_conflict` under non-strict policy
- **THEN** readiness status is `degraded` with canonical finding

#### Scenario: Strict mode with equivalent allowlist missing entry finding
- **WHEN** preflight evaluates equivalent `adapter.allowlist.missing_entry` under strict policy
- **THEN** readiness status escalates to `blocked`
