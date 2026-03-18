## ADDED Requirements

### Requirement: Action timeline SHALL normalize A2A delivery negotiation reason semantics
Action timeline events for A2A delivery negotiation MUST expose normalized reason codes under `a2a.*` namespace.

For this milestone, minimum reason codes MUST include:
- `a2a.sse_subscribe`
- `a2a.sse_reconnect`
- `a2a.delivery_fallback`
- `a2a.version_mismatch`

#### Scenario: SSE subscribe path emits reason
- **WHEN** runtime starts A2A delivery using SSE mode
- **THEN** timeline event includes reason `a2a.sse_subscribe`

#### Scenario: Delivery fallback emits reason
- **WHEN** runtime falls back from preferred delivery mode to fallback mode
- **THEN** timeline event includes reason `a2a.delivery_fallback`

### Requirement: Action timeline SHALL include A2A delivery/version correlation metadata
Timeline events for A2A delivery/version paths MUST include correlation metadata sufficient for cross-run tracing.

At minimum, payload MUST include:
- `task_id`
- `agent_id`
- `peer_id`
- `delivery_mode`
- `version_local`
- `version_peer`

#### Scenario: Version mismatch includes local and peer metadata
- **WHEN** runtime detects card version mismatch during handshake
- **THEN** timeline event payload includes local/peer version metadata and canonical `peer_id`

#### Scenario: SSE reconnect includes delivery metadata
- **WHEN** runtime retries SSE subscription
- **THEN** timeline event payload includes `delivery_mode=sse` and canonical A2A correlation fields

### Requirement: A2A delivery/version timeline semantics SHALL remain Run and Stream equivalent
For equivalent A2A interactions and effective configuration, Run and Stream MUST emit semantically equivalent A2A delivery/version reason and correlation semantics.

#### Scenario: Equivalent fallback path in Run and Stream
- **WHEN** equivalent requests trigger delivery fallback in both Run and Stream
- **THEN** both paths emit semantically equivalent `a2a.delivery_fallback` timeline semantics

#### Scenario: Equivalent version mismatch path in Run and Stream
- **WHEN** equivalent requests trigger version mismatch in both Run and Stream
- **THEN** both paths emit semantically equivalent `a2a.version_mismatch` timeline semantics
