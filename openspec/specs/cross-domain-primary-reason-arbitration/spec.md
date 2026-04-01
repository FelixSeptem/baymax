# cross-domain-primary-reason-arbitration Specification

## Purpose
TBD - created by archiving change introduce-cross-domain-primary-reason-arbitration-contract-a48. Update Purpose after archive.
## Requirements
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

### Requirement: Primary-reason arbitration SHALL include version-governance traceability
Cross-domain primary-reason arbitration MUST include version-governance traceability fields for each arbitration decision:
- requested rule version,
- effective rule version,
- rule-version source,
- policy action.

These fields MUST remain deterministic for equivalent inputs and equivalent configuration.

#### Scenario: Arbitration uses caller-requested version
- **WHEN** caller provides supported requested arbitration version
- **THEN** arbitration output preserves canonical primary reason and includes requested/effective/source version fields

#### Scenario: Arbitration falls back to runtime default version
- **WHEN** caller does not provide requested arbitration version
- **THEN** arbitration output includes effective default version and deterministic source classification

### Requirement: Primary-reason arbitration SHALL enforce version-policy fail-fast semantics
When arbitration version policy is configured as fail-fast, arbitration MUST reject unsupported or compatibility-mismatch versions before producing primary reason output.

#### Scenario: Unsupported version is requested
- **WHEN** requested version is not in runtime arbitration rule registry and `on_unsupported=fail_fast`
- **THEN** arbitration returns deterministic unsupported-version failure without emitting non-canonical primary reason

#### Scenario: Compatibility-mismatch version is requested
- **WHEN** requested version exists but violates configured compatibility window and `on_mismatch=fail_fast`
- **THEN** arbitration returns deterministic mismatch failure and does not silently downgrade to another version

### Requirement: Primary-reason arbitration SHALL align with policy-stack winner semantics
Cross-domain primary-reason arbitration MUST preserve alignment between arbitration output and policy precedence winner fields.

When policy-stack winner exists, arbitration explainability output MUST expose consistent stage/source semantics and MUST NOT remap winner source to a conflicting taxonomy.

#### Scenario: Arbitration receives policy winner from higher-precedence stage
- **WHEN** policy evaluator marks `security_s2` as winner and readiness also reports blocked
- **THEN** primary-reason arbitration output preserves winner-stage/source alignment without conflicting remap

#### Scenario: Arbitration output is replayed with equivalent winner input
- **WHEN** equivalent arbitration events with identical policy winner are replayed
- **THEN** primary reason and policy winner alignment remains deterministic and idempotent

