## ADDED Requirements

### Requirement: Replay tooling SHALL validate first-error attribution fixtures

Diagnostics replay MUST accept a versioned `eval_first_error_attribution.v1` fixture containing evaluation correlation, baseline or candidate first-error attribution, trajectory decision boundary, bounded evidence references, and expected normalized outcome. Replay MUST validate schema, bounds, privacy rules, correlation, deterministic normalization, and comparison without live runtime connectivity or side effects.

#### Scenario: Canonical attribution fixture replays successfully
- **WHEN** a valid fixture contains correlated, bounded, reference-only first-error attribution and the expected normalized digest
- **THEN** replay emits deterministic normalized output and a pass result without invoking a provider, tool, memory backend, resolver, Git, workspace, or runtime execution path

#### Scenario: Attribution fixture is malformed
- **WHEN** a fixture uses an unsupported version, omits required first-error identity, exceeds a bound, conflicts on action boundary, or includes body-bearing evidence
- **THEN** replay fails fast with the corresponding schema, boundary, evidence, or privacy classification and emits no partial success

### Requirement: Replay tooling SHALL preserve canonical attribution drift taxonomy

Replay MUST expose stable classifications for first-error schema, step, kind, owner, cause, prefix, action boundary, required evidence, evidence reference, recoverability, confidence, correlation, privacy, and Run/Stream parity drift. Equivalent input ordering MUST normalize identically, duplicate conflicting input MUST fail rather than use last-write-wins behavior, and historical fixture behavior MUST remain compatible.

#### Scenario: Replay detects first-error semantic drift
- **WHEN** candidate attribution differs from baseline in first-error location, cause ownership, decision boundary, evidence, recoverability, confidence, or prefix identity
- **THEN** replay reports the corresponding canonical drift classification and affected stable reference

#### Scenario: Memory retrieval and application cases replay through the same contract
- **WHEN** a fixture classifies a memory retrieval miss, scope error, or application error using existing bounded diagnostic references
- **THEN** replay uses the canonical first-error attribution path and does not introduce a separate memory evaluation fixture owner

#### Scenario: Mixed historical and attribution fixtures replay together
- **WHEN** the replay gate executes archived fixtures together with `eval_first_error_attribution.v1`
- **THEN** all fixture generations remain parseable, deterministic, side-effect free, and semantically backward compatible
