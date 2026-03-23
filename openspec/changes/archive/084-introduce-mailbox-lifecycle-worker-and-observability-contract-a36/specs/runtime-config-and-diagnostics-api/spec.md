## MODIFIED Requirements

### Requirement: Runtime config SHALL expose mailbox domain with deterministic precedence
Runtime configuration MUST expose `mailbox.*` domain and resolve effective values using precedence `env > file > default`.

Mailbox domain MUST include at least backend, retry, ttl, dlq, query pagination controls, and mailbox worker controls.

Minimum required mailbox worker controls:
- `mailbox.worker.enabled`
- `mailbox.worker.poll_interval`
- `mailbox.worker.handler_error_policy`

Default mailbox worker values MUST be:
- `mailbox.worker.enabled=false`
- `mailbox.worker.poll_interval=100ms`
- `mailbox.worker.handler_error_policy=requeue`

Invalid mailbox worker values in startup or hot reload MUST fail fast and keep the previous valid snapshot unchanged.

#### Scenario: Startup with mailbox overrides in file and env
- **WHEN** mailbox fields are configured in YAML and overlapping environment variables
- **THEN** effective mailbox config resolves with `env > file > default`

#### Scenario: Startup with invalid mailbox config
- **WHEN** mailbox config contains unsupported backend or invalid numeric range
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

#### Scenario: Hot reload provides invalid worker policy
- **WHEN** hot reload sets unsupported `mailbox.worker.handler_error_policy` or non-positive `mailbox.worker.poll_interval`
- **THEN** runtime rejects update and keeps previous valid snapshot

### Requirement: Runtime SHALL expose mailbox diagnostics aggregates and query entrypoint
The runtime MUST expose mailbox diagnostics aggregates and a library-level mailbox query entrypoint for coordination observability.

Mailbox diagnostics MUST preserve correlation fields (`run_id`, `task_id`, `workflow_id`, `team_id`) for composition with run/task board views.

Mailbox diagnostics MUST include lifecycle records for:
- consume
- ack
- nack
- requeue
- dead-letter transition
- expiration transition

Mailbox diagnostics MUST use canonical lifecycle reason taxonomy and preserve additive compatibility (`additive + nullable + default`) for new lifecycle fields.

#### Scenario: Consumer queries mailbox diagnostics
- **WHEN** application requests mailbox diagnostics or mailbox query API
- **THEN** runtime returns bounded records with normalized fields and correlation metadata

#### Scenario: Consumer inspects mailbox retries and dead-letter outcomes
- **WHEN** mailbox path contains retry and dead-letter events
- **THEN** diagnostics expose deterministic aggregate counters and reason taxonomy for those outcomes

#### Scenario: Consumer inspects worker lifecycle transitions
- **WHEN** mailbox worker executes consume/ack/nack/requeue lifecycle transitions
- **THEN** query and aggregate results include corresponding lifecycle records and canonical reason codes
