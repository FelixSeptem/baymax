## ADDED Requirements

### Requirement: CA3 governance-enabled threshold path SHALL preserve policy semantics
When CA3 threshold governance is enabled, failure handling MUST preserve existing policy semantics:
- `best_effort`: fallback to pre-governance threshold path with deterministic fallback marker.
- `fail_fast`: terminate assembly before model execution.

#### Scenario: Governance evaluation failure under best-effort
- **WHEN** CA3 governance evaluation fails and compaction policy is `best_effort`
- **THEN** CA3 falls back to pre-governance threshold path and continues assembly

#### Scenario: Governance evaluation failure under fail-fast
- **WHEN** CA3 governance evaluation fails and compaction policy is `fail_fast`
- **THEN** CA3 aborts assembly before model execution

### Requirement: CA3 governance-enabled semantics SHALL remain equivalent between Run and Stream
For equivalent inputs and effective config, governance mode selection, provider:model rollout matching, and fallback outcomes MUST remain semantically equivalent between Run and Stream.

#### Scenario: Equivalent Run and Stream with enforce mode
- **WHEN** equivalent requests run through Run and Stream with same governance config in `enforce` mode
- **THEN** both paths produce semantically equivalent threshold-governed gate outcomes

#### Scenario: Equivalent Run and Stream with dry-run mode
- **WHEN** equivalent requests run through Run and Stream with same governance config in `dry_run` mode
- **THEN** both paths preserve semantically equivalent non-enforcing final gate outcomes
