## ADDED Requirements

### Requirement: Runtime SHALL resolve arbitration rule version deterministically
Runtime MUST resolve arbitration rule version deterministically for each decision window and MUST expose requested/effective/source version metadata.

Resolution order MUST be:
1. requested rule version (when provided and valid),
2. runtime default rule version.

#### Scenario: Requested version is supported
- **WHEN** caller provides `requested_version` and it is supported in current runtime registry
- **THEN** runtime selects requested version as effective arbitration rule and emits deterministic source metadata

#### Scenario: Requested version is absent
- **WHEN** caller does not provide rule-version override
- **THEN** runtime selects configured default rule version and emits deterministic source metadata

### Requirement: Runtime SHALL enforce compatibility window and unsupported-version policy
Runtime MUST validate requested/default arbitration versions against runtime registry and compatibility window before arbitration is applied.

Unsupported or out-of-window versions MUST follow configured policy, and default policy MUST be `fail_fast`.

#### Scenario: Requested version is unsupported with fail-fast policy
- **WHEN** caller requests arbitration version that is not registered and `on_unsupported=fail_fast`
- **THEN** runtime rejects evaluation with deterministic unsupported-version classification

#### Scenario: Requested version mismatches compatibility window with fail-fast policy
- **WHEN** caller requests registered version outside configured compatibility window and `on_mismatch=fail_fast`
- **THEN** runtime rejects evaluation with deterministic compatibility-mismatch classification

### Requirement: Version-governed arbitration SHALL remain mode-equivalent and replay-stable
For equivalent inputs, equivalent effective configuration, and equivalent version request context, version-governed arbitration outputs MUST remain semantically equivalent across Run and Stream and MUST remain replay-idempotent.

#### Scenario: Equivalent Run and Stream arbitration under same requested version
- **WHEN** Run and Stream evaluate equivalent findings under the same requested version
- **THEN** requested/effective version outputs and primary arbitration semantics are equivalent

#### Scenario: Equivalent version-governance events are replayed
- **WHEN** recorder replays duplicate version-governance arbitration events for one run
- **THEN** logical version-governance aggregates remain stable after first ingestion
