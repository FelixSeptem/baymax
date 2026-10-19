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
