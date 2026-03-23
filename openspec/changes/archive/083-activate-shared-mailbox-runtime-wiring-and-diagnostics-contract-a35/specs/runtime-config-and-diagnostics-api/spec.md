## MODIFIED Requirements

### Requirement: Runtime config SHALL expose mailbox domain with deterministic precedence
Runtime configuration MUST expose `mailbox.*` domain and resolve effective values using precedence `env > file > default`.

Mailbox domain MUST include at least backend, retry, ttl, dlq, and query pagination controls.

Mailbox configuration resolution MUST be wired to runtime mailbox initialization behavior:
- `mailbox.enabled=false` MUST use shared memory mailbox runtime wiring,
- `mailbox.enabled=true` MUST initialize mailbox runtime using resolved backend configuration,
- invalid mailbox configuration MUST fail fast for startup/hot reload,
- `backend=file` initialization failure MAY fallback to memory backend only with deterministic fallback reason diagnostics.

#### Scenario: Startup with mailbox overrides in file and env
- **WHEN** mailbox fields are configured in YAML and overlapping environment variables
- **THEN** effective mailbox config resolves with `env > file > default`

#### Scenario: Startup with invalid mailbox config
- **WHEN** mailbox config contains unsupported backend or invalid numeric range
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

#### Scenario: Mailbox wiring uses shared memory when disabled
- **WHEN** runtime resolves `mailbox.enabled=false`
- **THEN** managed runtime paths still wire a shared memory mailbox instance rather than bypassing mailbox contract

### Requirement: Runtime SHALL expose mailbox diagnostics aggregates and query entrypoint
The runtime MUST expose mailbox diagnostics aggregates and a library-level mailbox query entrypoint for coordination observability.

Mailbox diagnostics MUST preserve correlation fields (`run_id`, `task_id`, `workflow_id`, `team_id`) for composition with run/task board views.

Mailbox diagnostics records MUST be written from runtime mailbox publish paths used by managed orchestration flows, so query/aggregate results reflect real sync/async/delayed execution traffic rather than test-only synthetic records.

#### Scenario: Consumer queries mailbox diagnostics
- **WHEN** application requests mailbox diagnostics or mailbox query API
- **THEN** runtime returns bounded records with normalized fields and correlation metadata

#### Scenario: Consumer inspects mailbox retries and dead-letter outcomes
- **WHEN** mailbox path contains retry and dead-letter events
- **THEN** diagnostics expose deterministic aggregate counters and reason taxonomy for those outcomes

#### Scenario: Managed orchestration traffic appears in mailbox query
- **WHEN** managed runtime executes mailbox-backed sync/async/delayed orchestration operations
- **THEN** `QueryMailbox` and `MailboxAggregates` include corresponding publish records and correlation identifiers
