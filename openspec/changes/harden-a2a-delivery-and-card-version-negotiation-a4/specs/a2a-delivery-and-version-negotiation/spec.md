## ADDED Requirements

### Requirement: A2A runtime SHALL negotiate delivery mode deterministically
A2A runtime MUST negotiate result delivery mode between requester and peer using normalized modes `callback` and `sse`.

Negotiation MUST follow deterministic order and MUST emit explicit fallback outcome when preferred mode is unsupported.

#### Scenario: Preferred SSE mode is supported
- **WHEN** requester prefers `sse` and peer advertises `sse` support
- **THEN** runtime selects `sse` without fallback and records successful negotiation

#### Scenario: Preferred SSE mode falls back to callback
- **WHEN** requester prefers `sse` but peer only supports `callback`
- **THEN** runtime falls back to `callback` using deterministic policy and records fallback reason

### Requirement: A2A runtime SHALL apply bounded delivery retry and reconnect policies
A2A runtime MUST enforce bounded retry/reconnect controls for delivery operations.

For this milestone:
- callback delivery retries MUST be bounded by configured retry budget,
- sse subscription reconnects MUST be bounded by configured reconnect budget.

#### Scenario: Callback delivery exhausts retry budget
- **WHEN** callback delivery repeatedly fails until retry budget is exhausted
- **THEN** runtime transitions to terminal delivery failure with normalized reason

#### Scenario: SSE subscription reconnect succeeds within budget
- **WHEN** initial SSE subscribe fails and reconnect attempts are available
- **THEN** runtime reconnects and resumes delivery before budget exhaustion

### Requirement: A2A runtime SHALL negotiate Agent Card schema version with strict-major compatibility
A2A runtime MUST negotiate Agent Card schema versions using strict-major compatibility and bounded minor compatibility policy.

For this milestone:
- major version mismatch MUST be treated as incompatible,
- major match with accepted minor range MUST be treated as compatible.

#### Scenario: Major version mismatch is rejected
- **WHEN** local card major version differs from peer card major version
- **THEN** runtime rejects peer negotiation with normalized `version_mismatch` classification

#### Scenario: Minor version compatible path is accepted
- **WHEN** local and peer card major versions match and peer minor version is within accepted policy
- **THEN** runtime accepts negotiation and records normalized compatibility result

### Requirement: A2A version and delivery failures SHALL map to normalized runtime taxonomy
A2A delivery/version failures MUST map to normalized runtime error semantics and MUST remain diagnosable across subsystems.

Minimum normalized reason classes for this milestone:
- `a2a.delivery_unsupported`
- `a2a.delivery_retry_exhausted`
- `a2a.sse_reconnect_exhausted`
- `a2a.version_mismatch`

#### Scenario: Unsupported delivery mode mapping
- **WHEN** peer supports neither requested mode nor fallback mode
- **THEN** runtime reports normalized `a2a.delivery_unsupported` classification

#### Scenario: Version mismatch mapping
- **WHEN** card major version mismatch occurs during handshake
- **THEN** runtime reports normalized `a2a.version_mismatch` classification

### Requirement: A2A delivery and version semantics SHALL remain Run and Stream equivalent
For equivalent A2A interactions and effective configuration, Run and Stream paths MUST produce semantically equivalent delivery-mode decisions, negotiation outcomes, and terminal status.

#### Scenario: Equivalent delivery fallback in Run and Stream
- **WHEN** equivalent requests encounter unsupported preferred delivery mode
- **THEN** Run and Stream both select the same fallback mode and equivalent terminal semantics

#### Scenario: Equivalent version mismatch in Run and Stream
- **WHEN** equivalent requests encounter card major version mismatch
- **THEN** Run and Stream both emit semantically equivalent mismatch outcome and terminal classification
