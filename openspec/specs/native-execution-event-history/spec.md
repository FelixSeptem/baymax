# native-execution-event-history Specification

## Purpose
Defines a source-owned, durable, bounded and read-only native execution event
history contract so a consumer can resume observation after a confirmed cursor
without treating callback projections, host records, or execution recovery as
event replay.

## Requirements

### Requirement: Native event history SHALL declare source-scoped capabilities

The runtime SHALL expose source-scoped event-history capabilities for runner,
isolated child, team, workflow and mailbox sources. A declaration MUST identify
the source and event identity namespaces, whether source sequence is defined and
its exact ordering scope, retention window/floor, opaque cursor scope and
format version, durable terminal recovery semantics, and current availability.
A capability missing from a source declaration MUST NOT be inferred from a
callback adapter, DTO, diagnostics record, package, Run ID, timestamp or host
observation sequence. A source without retained history SHALL return
`ErrHistoryUnsupported` before any read.

#### Scenario: Unsupported runner source is explicit
- **WHEN** a caller asks for capabilities or reads history for a direct runner
  source without a declared retained journal
- **THEN** the source reports no durable history and returns
  `ErrHistoryUnsupported` for the read
- **AND** it does not start a Run, subscribe to callbacks, create a host cursor
  or substitute a latest event

#### Scenario: Parallel source order remains local
- **WHEN** independent child or team members have separate source histories
- **THEN** each declaration states only its own source-local sequence scope
- **AND** the runtime does not claim a global natural order across the members

### Requirement: Read-after-cursor SHALL be bounded and source-owned

`NativeEventHistoryReader.ReadAfter` SHALL accept a source reference, an
optional opaque cursor and a bounded positive limit, then return a detached page
containing ordered source events, an optional next cursor, an optional retention
floor and a complete flag. The returned cursor SHALL be interpretable only by
the declaring source namespace and source instance. The reader SHALL reject a
malformed cursor with `ErrCursorMalformed`, a cursor for another source or
namespace with `ErrCursorForeign`, an expired cursor with `ErrCursorExpired`, a
retention discontinuity with `ErrHistoryGap`, and an unavailable declared source
with `ErrSourceUnavailable`. It MUST NOT silently begin at latest or at the
retention floor after any cursor error.

#### Scenario: Confirmed cursor resumes after consumer restart
- **WHEN** a consumer restarts with a cursor previously returned by a retained
  mailbox source
- **THEN** the reader returns the subsequent bounded source events and a new
  cursor according to that mailbox source's order
- **AND** it does not republish, claim, acknowledge, retry, dead-letter or run
  the message

#### Scenario: Foreign or expired cursor fails closed
- **WHEN** a reader receives a mailbox cursor for a different mailbox source,
  or a cursor whose referenced history is outside retention
- **THEN** it returns `ErrCursorForeign` or `ErrCursorExpired` respectively
- **AND** it does not return an unrelated page or silently move to latest

### Requirement: Native history replay SHALL be read-only and terminal-safe

History replay SHALL only read retained source evidence. It MUST NOT invoke a
model, tool, handler, host callback, scheduler, retry, resume, restore, Run,
Stream, business write or diagnostics write. Source terminal events and their
source identities SHALL remain recoverable from a retained source after a
consumer projection crashes. A replay cursor is not an execution checkpoint and
MUST NOT be accepted as authority to continue execution.

#### Scenario: Projection crashes before terminal consumption
- **WHEN** a retained mailbox source writes a terminal lifecycle event and the
  consumer crashes before projecting it
- **THEN** a later read after the last confirmed cursor returns the same
  source-owned terminal evidence according to its source order
- **AND** the replay performs no mailbox lifecycle transition or host business
  commit

#### Scenario: Callback identity is not a journal identity
- **WHEN** a direct callback EventID or mapped trace identifier resembles an
  event history identifier
- **THEN** the runtime does not accept it as a history cursor or deduplication
  key
- **AND** the direct callback source remains unsupported until its owner
  declares a retained history

### Requirement: History conformance SHALL provide reproducible offline evidence

The delivered contract SHALL include source-capability matrices, bounded valid
and invalid cursor fixtures, retained terminal/restart evidence and equivalent
shell/PowerShell gates. Offline history replay MUST parse and validate reference
fixtures without invoking a source owner, model, tool, handler, retry, resume,
restore, network or business write. Fixtures SHALL expose source-local ordering,
foreign cursor, malformed cursor, expired cursor, history gap, unsupported and
unavailable outcomes.

#### Scenario: Offline replay detects unsafe fallback
- **WHEN** a fixture represents an expired or gapped cursor but an
  implementation returns a latest page
- **THEN** the history contract gate fails with a fixture-local classification
- **AND** no execution source is activated during the check

### Requirement: Source-owner history expansion SHALL declare actual boundaries

For runner, isolated child, team, workflow, and mailbox sources, Baymax SHALL
publish a source-scoped history declaration identifying the event identity
namespace, whether a source-local sequence exists and its ordering scope,
retention window and floor, opaque cursor scope/version, terminal recovery
semantics, and current availability. A missing declaration SHALL remain
unsupported rather than being inferred from callbacks, DTOs, package presence,
Run IDs, timestamps, or host observation order.

#### Scenario: Unsupported source is explicit
- **WHEN** a caller requests history for a source without a retained journal
- **THEN** the source returns `ErrHistoryUnsupported` before reading or
  activating execution
- **AND** its capability declaration includes a stable unsupported reason

#### Scenario: Source availability is explicit
- **WHEN** a declared retained source cannot be opened or its store is
  unavailable
- **THEN** the reader returns `ErrSourceUnavailable`
- **AND** it does not downgrade the source to unsupported or start a new run

### Requirement: Source-owner replay SHALL retain bounded cursor semantics

A supported source SHALL provide bounded, detached pages after an optional
opaque cursor, including events, an optional next cursor, an optional retention
floor, and a complete flag. Cursors SHALL be interpretable only by their source
namespace and source instance. Malformed, foreign, expired, and gapped cursors
MUST return `ErrCursorMalformed`, `ErrCursorForeign`, `ErrCursorExpired`, or
`ErrHistoryGap` respectively; the reader MUST NOT silently use latest or the
retention floor after an error.

#### Scenario: Consumer restart continues from a confirmed cursor
- **WHEN** a new consumer process reads a retained source with its last
  confirmed cursor
- **THEN** it receives only the subsequent bounded source-local events and a
  valid next cursor
- **AND** no source lifecycle transition is performed

#### Scenario: Retention loss fails closed
- **WHEN** a cursor points before the retained floor or a retained sequence has
  an unrecoverable gap
- **THEN** the reader returns the corresponding typed expiry or gap error
- **AND** it does not return an unrelated latest page

### Requirement: Source-owner replay SHALL remain projection-safe

History reads SHALL expose source-owned facts without invoking Run, Stream,
model, tool, handler, retry, resume, restore, ack, nack, lease, requeue, DLQ,
scheduler, business commit, or diagnostics writes. Source journal progress and
consumer projection progress SHALL be independent. Overlapping pages SHALL be
idempotently deduplicated by stable source event identity; conflicting payloads
MUST fail closed.

#### Scenario: Projection crashes before consuming terminal evidence
- **WHEN** the source has retained a terminal event but its consumer crashes
  before projection
- **THEN** the next read after the confirmed cursor returns the same terminal
  evidence
- **AND** replay does not execute or mutate the source

#### Scenario: Overlap and payload conflict are handled safely
- **WHEN** a consumer reads an overlapping page twice
- **THEN** identical source events are accepted once
- **AND** the same source identity with a different payload fails closed

### Requirement: Source-local ordering SHALL not imply a global order

Each source SHALL state the exact scope of its monotonic sequence. Parallel
child or team sources MUST retain their own local order without claiming a
global natural order, and a history cursor MUST NOT be accepted as an execution
checkpoint or restore authority.

#### Scenario: Parallel sources remain independently ordered
- **WHEN** two child or team member sources emit events concurrently
- **THEN** each source can be replayed in its declared local order
- **AND** the runtime does not synthesize a cross-source sequence

### Requirement: History capability SHALL state writer and cursor transaction scope

Each retained source journal declaration MUST state whether append, retention
floor advancement, and cursor reads are protected only by one writer or by a
source-owned multi-writer transaction. A file rename, process reopen, or
overlapping read MUST NOT be represented as multi-writer CAS or distributed
cursor advancement.

#### Scenario: File history advertises local ownership
- **WHEN** a consumer reads a persisted same-host FileStore history manifest
- **THEN** append and floor advancement are declared single-writer and
  sequential-reopen scoped
- **AND** a second writer or distributed cursor transaction is unsupported

#### Scenario: Concurrent cursor writer fails closed
- **WHEN** a second process attempts to advance a cursor without the declared
  source transaction/fence
- **THEN** the source returns a typed stale-owner or unavailable error
- **AND** it does not merge pages, skip a gap, or fall back to latest

### Requirement: History replay SHALL not imply distributed coordination

Successful read-after-cursor replay after a process restart MUST be reported
only as retained source evidence. It MUST NOT be used as evidence of concurrent
writer safety, cross-machine journal replication, atomic reader/writer
transactions, or multi-instance ownership.

#### Scenario: Restart replay has bounded meaning
- **WHEN** a new process reads a persisted source journal from a confirmed
  cursor
- **THEN** replay succeeds within the declared local ownership boundary
- **AND** capability evidence continues to mark distributed coordination as
  unsupported

### Requirement: Owner append failures SHALL be observable

When an execution owner is configured with a native event history writer, append failures MUST NOT be silently discarded while capability state continues to claim complete retained source history. The owner MUST expose a typed health/availability outcome and preserve whether source facts may be incomplete. This requirement does not make event history and external consumer projection one transaction.

#### Scenario: Journal write failure is surfaced
- **WHEN** an explicitly configured runner, child, team, or workflow journal cannot persist an appended source event
- **THEN** the owner records or returns a typed history unavailable/degraded outcome associated with the source
- **AND** it does not claim that the failed event is replayable

#### Scenario: No history writer is configured
- **WHEN** the owner runs without an explicitly injected native history writer
- **THEN** its history capability remains unsupported
- **AND** it does not create an implicit journal or derive history from callback identifiers

### Requirement: History replay SHALL NOT repair uncertain execution

Source history replay MUST return only facts whose source append committed. If a source transition may have occurred without a committed fact, the owner MUST report recovery-required/unavailable and MUST NOT use replay, parent Run, child Run, retry, or resume to synthesize the missing fact.

#### Scenario: Append failure leaves a source gap
- **WHEN** a source journal append fails and a later read observes unavailable or incomplete history
- **THEN** the reader reports a typed unavailable or gap outcome
- **AND** it does not silently return latest or invent a terminal event

### Requirement: History capabilities SHALL distinguish commit recovery from replay

When a source advertises retained history, its capability declaration MUST
state whether committed source records and projection recovery are available,
the cursor/commit transaction scope, and the typed outcome for an uncertain
source append. Read-after-cursor support alone MUST NOT be interpreted as an
atomic source/projection transaction or execution recovery capability.

#### Scenario: History is durable but projection recovery is unavailable
- **WHEN** a file-backed source can replay retained events but has no declared
  source commit/projection recovery store
- **THEN** the history capability remains explicit about that limitation
- **AND** a consumer cannot infer atomic projection or resume authority

#### Scenario: Recovery-required source gap is explicit
- **WHEN** a source append may have occurred but the commit record is missing
  or unreadable
- **THEN** history access returns a typed unavailable or recovery-required
  outcome
- **AND** it does not silently start at the latest retained event

### Requirement: Projection recovery SHALL use committed source facts only

After a consumer crash, recovery MUST read only source commit records whose
source append is durably committed. Consumer cursor advancement MUST be
independent and idempotent by source event identity; an append-health callback
or callback EventID MUST NOT serve as a commit acknowledgement or a projection
cursor.

#### Scenario: Append health is not a commit proof
- **WHEN** `NativeEventHistoryError` reports no error after an asynchronous
  callback
- **THEN** the runtime does not claim that an external consumer projection was
  committed atomically
- **AND** recovery still relies on the source-owned commit record

#### Scenario: Recovery never reruns uncertain effects
- **WHEN** a committed source fact is unavailable during projection recovery
- **THEN** the owner returns a typed recovery-required/unavailable result
- **AND** it does not call Run, retry, resume, or replay a parent/child effect

### Requirement: Source history recovery SHALL bind to an explicit projection owner

Read-after-cursor history MAY supply facts to projection recovery only when the
caller explicitly binds the declaring source journal to a matching projection
ledger. The recovery result MUST preserve source-local ordering, retention
floor, cursor scope, event identity, and terminal evidence; a history reader
alone MUST NOT imply projection recovery or acknowledgement support.

#### Scenario: Durable history without ledger remains limited

- **WHEN** a file-backed source journal is opened without a matching projection
  ledger
- **THEN** its manifest continues to report retained history without projection
  recovery
- **AND** a consumer cannot infer an acknowledgement or atomic projection
  transaction from `ReadAfter`

#### Scenario: Reopened source facts feed the matching ledger

- **WHEN** a fresh process reads a confirmed cursor from a source journal and
  supplies the matching ledger owner
- **THEN** bounded facts can be reconciled without changing source lifecycle
  state
- **AND** the returned cursor and terminal evidence remain source-owned

### Requirement: History recovery SHALL preserve source-local ordering and overlap semantics

Recovery across bounded pages MUST accept identical overlap by source event
identity and digest, reject conflicting overlap with `ErrHistoryConflict`, and
reject gaps, foreign cursors, and expired cursors without silently changing
source position. Parallel child and team sources MUST retain independent source
order and MUST NOT be merged into a synthetic global cursor.

#### Scenario: Overlapping pages are safe to retry

- **WHEN** a consumer retries a recovery page after an acknowledgement timeout
- **THEN** identical source facts deduplicate by source event identity
- **AND** the next source cursor is not advanced twice

#### Scenario: Parallel sources do not share a cursor

- **WHEN** recovery reads are interleaved for two child or team sources
- **THEN** each source validates and advances only its own cursor
- **AND** a cursor from one source is rejected as foreign by the other
