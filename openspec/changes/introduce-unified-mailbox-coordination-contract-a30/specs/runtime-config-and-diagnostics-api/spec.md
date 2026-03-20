## ADDED Requirements

### Requirement: Runtime config SHALL expose mailbox domain with deterministic precedence
Runtime configuration MUST expose `mailbox.*` domain and resolve effective values using precedence `env > file > default`.

Mailbox domain MUST include at least backend, retry, ttl, dlq, and query pagination controls.

#### Scenario: Startup with mailbox overrides in file and env
- **WHEN** mailbox fields are configured in YAML and overlapping environment variables
- **THEN** effective mailbox config resolves with `env > file > default`

#### Scenario: Startup with invalid mailbox config
- **WHEN** mailbox config contains unsupported backend or invalid numeric range
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

### Requirement: Runtime SHALL expose mailbox diagnostics aggregates and query entrypoint
The runtime MUST expose mailbox diagnostics aggregates and a library-level mailbox query entrypoint for coordination observability.

Mailbox diagnostics MUST preserve correlation fields (`run_id`, `task_id`, `workflow_id`, `team_id`) for composition with run/task board views.

#### Scenario: Consumer queries mailbox diagnostics
- **WHEN** application requests mailbox diagnostics or mailbox query API
- **THEN** runtime returns bounded records with normalized fields and correlation metadata

#### Scenario: Consumer inspects mailbox retries and dead-letter outcomes
- **WHEN** mailbox path contains retry and dead-letter events
- **THEN** diagnostics expose deterministic aggregate counters and reason taxonomy for those outcomes
