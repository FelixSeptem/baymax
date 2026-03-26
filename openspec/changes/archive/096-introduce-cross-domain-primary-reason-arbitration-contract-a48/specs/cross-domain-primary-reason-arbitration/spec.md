## ADDED Requirements

### Requirement: Runtime SHALL apply deterministic cross-domain primary-reason precedence
Runtime MUST apply a deterministic primary-reason precedence when timeout, readiness, and adapter-health findings co-exist in one logical decision window.

Canonical precedence order:
1. timeout exhausted or timeout reject
2. readiness blocked
3. adapter required unavailable
4. readiness degraded or adapter optional unavailable
5. warning/info findings

#### Scenario: Multiple domains emit findings in one decision window
- **WHEN** timeout, readiness, and adapter-health findings are all present
- **THEN** runtime selects one primary reason using canonical precedence order

#### Scenario: No blocking-class finding exists
- **WHEN** only degraded or warning/info findings exist
- **THEN** runtime selects primary reason from highest available precedence bucket deterministically

### Requirement: Primary-reason tie-break SHALL be deterministic and observable
When multiple candidates exist at the same precedence level, runtime MUST apply deterministic tie-break using canonical code lexical order and MUST expose conflict observability counters.

#### Scenario: Two same-level candidates compete
- **WHEN** two findings in the same precedence bucket are both eligible as primary reason
- **THEN** runtime selects lexical-min canonical code as primary and records one arbitration conflict

#### Scenario: No same-level conflict occurs
- **WHEN** exactly one candidate exists at top precedence bucket
- **THEN** runtime selects it as primary without increasing arbitration conflict counter

### Requirement: Primary-reason arbitration SHALL remain mode-equivalent and replay-stable
For equivalent inputs and equivalent configuration, primary-reason arbitration output MUST remain semantically equivalent across Run and Stream and MUST remain replay-idempotent.

#### Scenario: Equivalent Run and Stream arbitration
- **WHEN** equivalent composite findings are evaluated in Run and Stream paths
- **THEN** primary domain/code/source are semantically equivalent

#### Scenario: Equivalent arbitration events are replayed
- **WHEN** equivalent arbitration events are replayed for one run
- **THEN** logical arbitration aggregate counters remain stable after first ingestion
