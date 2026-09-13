## ADDED Requirements

### Requirement: Replay tooling SHALL validate continuity comparison fixtures

Diagnostics replay MUST accept a versioned `eval_continuity_comparison.v1` fixture containing bounded baseline/candidate projections and expected normalized outcome. It MUST validate schema, bounds, privacy rules, fixed-axis coverage, and deterministic comparison output without live runtime connectivity.

#### Scenario: Continuity fixture replays successfully
- **WHEN** a valid fixture contains equivalent baseline and candidate references and the expected pass outcome
- **THEN** replay emits deterministic normalized output and a pass result

#### Scenario: Continuity fixture schema is invalid
- **WHEN** a fixture has unsupported version, missing phase/run identity, unbounded fields, or body-bearing content
- **THEN** replay fails fast with the corresponding `continuity_schema_drift` or `continuity_privacy_violation` reason

### Requirement: Replay tooling SHALL preserve canonical continuity drift classes

Replay MUST expose stable classifications for identity, objective, task association, attempt/lease, workspace, pending request, checkpoint, artifact reference, owner, missing/duplicate reference, Run/Stream parity, and recovery idempotency drift. Adding this fixture MUST preserve all historical replay fixture behavior.

#### Scenario: Replay detects axis-specific drift
- **WHEN** candidate continuity differs from baseline on one fixed axis
- **THEN** replay reports the canonical drift class for that axis and identifies the affected reference key

#### Scenario: Mixed historical and continuity fixtures replay together
- **WHEN** the gate executes archived fixture versions alongside `eval_continuity_comparison.v1`
- **THEN** all fixtures are validated deterministically without parser or output regressions

