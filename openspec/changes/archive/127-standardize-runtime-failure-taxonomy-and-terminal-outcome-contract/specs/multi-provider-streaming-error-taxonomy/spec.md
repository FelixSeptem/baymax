## MODIFIED Requirements

### Requirement: Streaming failures SHALL terminate fail-fast with aligned classification
On non-recoverable stream failures, adapters MUST stop streaming immediately and map failures to aligned baseline error classes. The mapping MUST distinguish failures before a stream is established from failures after valid events have been consumed. Post-start failures MUST preserve already emitted valid facts and MUST NOT trigger an automatic retry of consumed work.

#### Scenario: Provider stream error occurs before stream establishment
- **WHEN** request construction, authentication, or connection setup fails before any response event is consumed
- **THEN** the adapter returns the existing classified model failure through the Go error path with phase `pre_execution` and no partial-output claim

#### Scenario: Provider stream error occurs after valid events
- **WHEN** a provider stream emits valid text or complete tool-call events and then raises an unrecoverable error
- **THEN** the adapter terminates immediately, preserves prior events, maps the error to the aligned baseline class with phase `post_start`, and does not replay the consumed request

### Requirement: Provider error taxonomy SHALL include normalized reason categories
Provider failures MUST include normalized reason categories in diagnostics/details with at least: `auth`, `rate_limit`, `timeout`, `request`, `server`, `unknown`. The normalized terminal projection MUST retain this source reason and map it to the shared failure family without removing existing provider categories.

#### Scenario: Rate limit error occurs after stream start
- **WHEN** a provider reports a rate limit condition after valid stream events were consumed
- **THEN** diagnostics include reason `rate_limit`, phase `post_start`, preserved partial facts, and normalized terminal family `runtime_failed` or `retry_exhausted` according to the owning retry decision
