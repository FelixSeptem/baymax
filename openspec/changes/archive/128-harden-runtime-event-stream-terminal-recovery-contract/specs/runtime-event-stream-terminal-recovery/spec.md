## ADDED Requirements

### Requirement: Runtime SHALL expose an observer recovery lifecycle without changing Run state

The Runtime SHALL represent observer lifecycle outcomes for an event stream using source-owned results such as `catching_up`, `live`, `disconnected`, `stopped`, `terminal_available`, `expired`, `gap`, and `backpressure`. Observer disconnect or consumer stop MUST NOT mutate the business Run state, trigger retry/resume actions, or move a terminal Run back to `working`.

#### Scenario: Observer disconnect preserves business execution
- **WHEN** a consumer disconnects while the source Run remains active
- **THEN** the Runtime records a recoverable observer outcome and leaves the source Run lifecycle unchanged

#### Scenario: Consumer stop after terminal outcome is idempotent
- **WHEN** a consumer stops after a terminal event has been accepted
- **THEN** the recovery result exposes the existing terminal outcome and does not create a second terminal transition

### Requirement: Recovery SHALL use an accepted source cursor and preserve catch-up/live-tail ordering

Recovery MUST request history only from a source-accepted cursor and MUST validate the source-provided catch-up boundary before entering live tail. Overlapping catch-up and live events SHALL reuse existing event ID, sequence, and deduplication semantics. The recovery projection MUST NOT synthesize a cursor, sequence, history event, or exactly-once guarantee.

#### Scenario: Valid cursor reconnects with bounded overlap
- **WHEN** a host reconnects with a valid cursor and the source returns history followed by an overlapping live event
- **THEN** the normalized stream reports catch-up then live and delivers the overlapping event at most once according to source deduplication

#### Scenario: Handoff gap is rejected deterministically
- **WHEN** the source-provided live handoff omits a required monotonic sequence progression
- **THEN** recovery returns a deterministic `stream_sequence_gap` classification and does not claim a live subscription

#### Scenario: Expired cursor is not silently replaced
- **WHEN** a reconnect cursor is outside source retention
- **THEN** recovery returns an explicit `expired` outcome and does not start an unrelated latest subscription

### Requirement: Recovery SHALL converge on one authoritative terminal outcome

When a terminal event, terminal snapshot, or late conflicting outcome is observed during recovery, the Runtime MUST normalize it through the existing terminal outcome arbiter. The first accepted business terminal outcome MUST remain authoritative; equivalent repeats MUST be idempotent; late conflicts MUST be recorded as diagnostics without overwriting the business result. Recovery MUST retain partial output and completed tool-call facts already owned by the source.

#### Scenario: Terminal snapshot arrives before terminal event
- **WHEN** recovery queries a source-owned terminal snapshot before the corresponding terminal event is delivered
- **THEN** the snapshot is exposed as the authoritative terminal outcome and a later equivalent event is deduplicated

#### Scenario: Late conflicting terminal is observed
- **WHEN** recovery observes a terminal outcome that conflicts with the first accepted business terminal
- **THEN** the Runtime records a terminal conflict classification and preserves the first business terminal outcome

#### Scenario: Partial facts survive observer recovery
- **WHEN** a stream disconnects after partial text and one or more tool calls have completed
- **THEN** catch-up or terminal projection retains those facts with the original run/step correlation

### Requirement: Recovery outcomes SHALL remain correlated across protocol, diagnostics, and tracing

Every normalized recovery result MUST preserve known `session_id`, `run_id`, `step_id`, `event_id`, and source identity, and MUST reference the observed cursor or terminal outcome when available. Diagnostics writes MUST use `observability/event.RuntimeRecorder` as the single writer. New fields MUST be additive, nullable, and defaultable; high-cardinality cursor and causation values MUST NOT be emitted as OTel metric dimensions.

#### Scenario: Recovered terminal is query-equivalent
- **WHEN** a host obtains a terminal outcome through catch-up and then queries runtime diagnostics
- **THEN** the query exposes semantically equivalent terminal classification, correlation, and retained facts

#### Scenario: Historical consumer omits recovery fields
- **WHEN** an older consumer reads a recovery result without the additive recovery fields
- **THEN** parsing and default behavior remain backward compatible

### Requirement: Run and Stream recovery SHALL be semantically equivalent

Equivalent Run and Stream executions with the same source history, cursor, delivery policy, interruption, cancellation, timeout, and Provider outcome MUST produce equivalent normalized recovery state, terminal outcome, correlation, retained facts, and reason classifications after permitted event-order normalization.

#### Scenario: Equivalent Run and Stream reconnect
- **WHEN** equivalent Run and Stream paths reconnect using the same valid cursor and source history
- **THEN** both paths expose equivalent catch-up/live state, deduplication, terminal outcome, and recovery reason

#### Scenario: Provider failure is classified consistently
- **WHEN** an equivalent Provider stream failure occurs on Run and Stream paths after partial output
- **THEN** both paths preserve the same partial facts and expose the same terminal failure classification

### Requirement: Recovery contract SHALL be replayable and enforce library-first ownership

The Runtime SHALL provide a versioned `runtime_event_stream_terminal_recovery.v1` fixture and contract gate covering observer stop, disconnect, cursor catch-up, overlap deduplication, handoff gap, retention expiry, backpressure, cancel, timeout, Provider failure, terminal conflict, and Run/Stream parity. Replay and gate execution MUST be deterministic, shell/PowerShell equivalent, and MUST fail if implementation introduces a transport listener, hosted connection manager, external event store, global binding queue, or second terminal state machine.

#### Scenario: Canonical recovery fixture replays successfully
- **WHEN** replay processes a valid recovery fixture whose normalized output matches the expected source-owned results
- **THEN** replay succeeds deterministically without live runtime or network connectivity

#### Scenario: Recovery drift blocks the gate
- **WHEN** normalized output differs in cursor handoff, deduplication, terminal convergence, retained facts, or Run/Stream parity
- **THEN** replay returns a stable recovery drift classification and the contract gate fails

#### Scenario: Hosted ownership is rejected
- **WHEN** recovery implementation adds a transport server, hosted event store, or binding-owned global queue
- **THEN** the gate fails with a deterministic library-first boundary classification
