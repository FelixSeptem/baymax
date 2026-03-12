## ADDED Requirements

### Requirement: Multi-provider model adapters SHALL provide aligned streaming semantics
OpenAI, Anthropic, and Gemini adapters MUST provide aligned external streaming semantics for text delta, complete tool call emission, completion, and error termination.

#### Scenario: Same prompt streamed via different providers
- **WHEN** one prompt is streamed through OpenAI, Anthropic, and Gemini adapters
- **THEN** each provider emits events that can be interpreted under the same external semantic contract

### Requirement: ModelEvent type set SHALL allow minimal extensible enum additions
The runtime MUST allow adding minimal new `ModelEvent.Type` values required for multi-provider streaming alignment while preserving compatibility for existing event consumers.

#### Scenario: New normalized streaming event is introduced
- **WHEN** an additional event type is required to normalize provider streams
- **THEN** the type can be added without removing previously supported event types

### Requirement: Tool call streaming SHALL remain complete-only externally
External tool call events MUST be emitted only when a complete tool call is available; argument fragments MUST NOT be exposed externally.

#### Scenario: Provider emits partial tool arguments
- **WHEN** streaming includes incremental tool argument fragments
- **THEN** adapter buffers fragments internally and emits one external complete tool call event when ready

### Requirement: Streaming failures SHALL terminate fail-fast with aligned classification
On non-recoverable stream failures, adapters MUST stop streaming immediately and map failures to aligned baseline error classes.

#### Scenario: Provider stream error occurs
- **WHEN** a provider stream raises an unrecoverable error
- **THEN** stream terminates immediately and returns classified model failure without emitting subsequent events

### Requirement: Provider error taxonomy SHALL include normalized reason categories
Provider failures MUST include normalized reason categories in diagnostics/details with at least: `auth`, `rate_limit`, `timeout`, `request`, `server`, `unknown`.

#### Scenario: Rate limit error occurs
- **WHEN** provider returns a rate limit error during generate or stream
- **THEN** error mapping includes baseline class plus normalized reason `rate_limit`
