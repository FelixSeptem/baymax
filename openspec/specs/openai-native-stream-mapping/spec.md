# openai-native-stream-mapping Specification

## Purpose
TBD - created by archiving change upgrade-openai-native-stream-mapping. Update Purpose after archive.
## Requirements
### Requirement: OpenAI Streaming SHALL use native SDK event stream
`model/openai` MUST consume OpenAI Go official SDK Responses streaming events as the source of truth for stream output, instead of compatibility-only generate fallback behavior.

#### Scenario: Native stream initialization succeeds
- **WHEN** a streaming run starts with valid OpenAI credentials and model settings
- **THEN** the adapter opens a native SDK stream and emits model events derived from that stream

#### Scenario: Native stream emits textual deltas
- **WHEN** the SDK stream produces text delta events
- **THEN** the adapter emits corresponding `ModelEvent` items preserving causal order

### Requirement: ModelEvent type set SHALL be extensible and backward compatible
The system MUST allow adding new `ModelEvent.Type` values needed by native streaming semantics while preserving compatibility for existing event consumers.

#### Scenario: New event type is introduced
- **WHEN** native SDK emits an event requiring a new semantic type
- **THEN** the adapter emits the new type without removing existing supported event types

#### Scenario: Existing consumer receives unknown type
- **WHEN** an older consumer encounters an unrecognized `ModelEvent.Type`
- **THEN** the event payload remains structurally valid and the run can still be observed via existing correlation fields

### Requirement: Tool call streaming SHALL expose complete calls only
Streaming tool call output MUST be surfaced only when a complete tool call is available; partial argument fragments MUST NOT be emitted as external tool call events.

#### Scenario: Partial tool call arguments arrive
- **WHEN** the SDK emits incremental argument fragments for a tool call
- **THEN** the adapter buffers fragments internally and does not emit an external tool call event yet

#### Scenario: Tool call becomes complete
- **WHEN** the SDK stream reaches a complete tool call representation
- **THEN** the adapter emits exactly one complete tool call event for that call identifier

### Requirement: Stream execution SHALL fail fast on model streaming errors
Streaming execution MUST terminate immediately on non-recoverable stream errors and return classified failure information.

#### Scenario: Stream error occurs before completion
- **WHEN** the SDK stream returns an error during active streaming
- **THEN** the runner stops streaming immediately and returns an error classified as `ErrModel` or `ErrPolicyTimeout` by policy context

#### Scenario: Step timeout elapses during streaming
- **WHEN** streaming exceeds configured step timeout
- **THEN** the run terminates with `ErrPolicyTimeout` and no further stream events are emitted

### Requirement: Stream result SHALL be semantically consistent with generate result
For equivalent input and model settings, aggregated stream final answer MUST be semantically consistent with non-stream generate final answer.

#### Scenario: Same prompt in run and stream paths
- **WHEN** a prompt is executed once via `Run` and once via `Stream`
- **THEN** resulting final answers are semantically equivalent even if token segmentation differs

### Requirement: Streaming observability correlation SHALL remain complete
All emitted stream-related events MUST preserve `run_id`, `iteration`, and tracing correlation fields required for end-to-end observability.

#### Scenario: Stream run emits lifecycle events
- **WHEN** a stream run starts, produces deltas, and completes
- **THEN** all lifecycle events include stable run correlation fields and can be joined in one trace context

