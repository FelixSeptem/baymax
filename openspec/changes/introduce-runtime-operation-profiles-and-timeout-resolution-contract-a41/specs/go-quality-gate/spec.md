## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include cross-domain timeout-resolution contract suites
Shared multi-agent gate MUST execute cross-domain timeout-resolution suites as blocking checks.

The suites MUST cover at minimum:
- operation-profile selection validation,
- layered precedence resolution (`profile -> domain -> request`),
- parent-child timeout clamp and exhausted-budget reject behavior,
- replay idempotency of timeout-resolution aggregates,
- Run/Stream equivalence and memory/file backend parity.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** cross-domain timeout-resolution suites run as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent cross-domain timeout-resolution suites run as required blocking checks

#### Scenario: Regression introduces precedence or clamp drift
- **WHEN** contract suites detect divergence in layered timeout precedence or parent-budget convergence semantics
- **THEN** shared gate fails fast and blocks merge

### Requirement: Quality gate SHALL preserve docs-consistency traceability for operation-profile timeout fields
Repository quality gate MUST ensure docs/config/spec alignment for newly introduced operation-profile timeout fields and diagnostics mappings.

#### Scenario: Config field introduced without docs mapping
- **WHEN** operation-profile timeout field exists in runtime config but docs mapping is missing or stale
- **THEN** docs consistency validation fails and quality gate returns non-zero status

#### Scenario: Docs and contract index are synchronized
- **WHEN** operation-profile timeout fields, diagnostics keys, and contract index mappings are aligned
- **THEN** quality gate proceeds without docs-consistency failure
