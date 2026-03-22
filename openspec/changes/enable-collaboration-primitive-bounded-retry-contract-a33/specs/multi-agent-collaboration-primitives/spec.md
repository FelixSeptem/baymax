## MODIFIED Requirements

### Requirement: Collaboration primitive retries SHALL be disabled by default
Collaboration primitive layer MUST keep retry disabled by default.

When `composer.collab.retry.enabled=true`, runtime MUST allow bounded primitive-layer retry under explicit retry governance and MUST keep the default-disabled behavior unchanged when the flag is not enabled.

Default retry governance for this milestone MUST be:
- `max_attempts=3`
- `backoff_initial=100ms`
- `backoff_max=2s`
- `multiplier=2.0`
- `jitter_ratio=0.2`
- `retry_on=transport_only`

#### Scenario: Delegation fails with retryable transport error under default config
- **WHEN** collaboration primitive receives retryable transport-class failure and `composer.collab.retry.enabled=false`
- **THEN** runtime does not perform primitive-layer retry and preserves downstream retry governance semantics

#### Scenario: Delegation retries within bounded policy when enabled
- **WHEN** collaboration primitive receives retryable transport-class failure and retry is enabled with bounded defaults
- **THEN** runtime retries up to configured attempt limit with exponential backoff+jitter and converges deterministically

## ADDED Requirements

### Requirement: Collaboration primitive retry scope SHALL exclude post-accept async-await convergence stage
Primitive-layer retry MUST apply to synchronous delegation path and async submit stage only.

After async request is accepted, terminal convergence MUST continue through existing async-await lifecycle contract and MUST NOT add primitive-layer retry on report/await/reconcile stages.

#### Scenario: Async request is accepted and later terminal convergence is delayed
- **WHEN** async delegation returns accepted acknowledgement and later report delivery is delayed
- **THEN** primitive retry is not re-entered and runtime uses existing async-await convergence contract

### Requirement: Collaboration primitive retry ownership SHALL avoid scheduler double-retry
For scheduler-managed collaboration execution, runtime MUST avoid layering primitive retry on top of scheduler retry budget.

Scheduler-managed paths MUST keep a single retry owner to prevent compounded retries.

#### Scenario: Scheduler-managed branch fails with retryable transport error
- **WHEN** scheduler-managed collaboration branch hits transport failure under retry-enabled configuration
- **THEN** runtime applies single-owner retry semantics and does not perform compounded primitive+scheduler retries

