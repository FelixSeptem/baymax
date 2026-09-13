## ADDED Requirements

### Requirement: Replay SHALL support cross-provider handoff and stream-edge fixtures

Diagnostics replay MUST parse versioned `provider_handoff_stream_edge.v1` fixtures covering canonical tool-call/tool-result handoff, thinking/reasoning projection, stream completion and abort, usage omission, empty/Unicode content, overflow, pre-step fallback, mid-stream failure, and Run/Stream parity. Replay MUST be offline, read-only, deterministic, and compatible with historical fixture versions.

#### Scenario: Valid cross-provider fixture replays
- **WHEN** replay receives a valid `provider_handoff_stream_edge.v1` fixture with matching normalized digest
- **THEN** replay succeeds without invoking a provider/tool or mutating runtime state

#### Scenario: Malformed fixture fails fast
- **WHEN** required provider, mode, correlation, boundary, or expected outcome fields are missing or invalid
- **THEN** replay fails with deterministic schema validation and produces no partial success

### Requirement: Replay SHALL classify provider conformance drift canonically

Replay MUST classify at minimum `provider_handoff_schema_drift`, `provider_tool_call_correlation_drift`, `provider_tool_result_feedback_drift`, `provider_thinking_projection_drift`, `provider_stream_boundary_drift`, `provider_abort_usage_drift`, `provider_overflow_drift`, `provider_unicode_empty_content_drift`, `provider_fallback_fence_drift`, and `provider_run_stream_parity_drift`.

#### Scenario: Replay detects handoff or stream drift
- **WHEN** normalized output differs from fixture expectation in correlation, feedback, thinking, boundary, usage/abort, overflow, content, fallback, or parity
- **THEN** replay returns the corresponding stable drift classification

#### Scenario: Replaying the same fixture is idempotent
- **WHEN** the same fixture is replayed repeatedly
- **THEN** normalized output and drift classification remain identical and no counters or source state inflate
