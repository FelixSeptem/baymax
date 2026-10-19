## ADDED Requirements

### Requirement: Mailbox SHALL own retained source event history

For a mailbox source that declares retained history, the existing mailbox owner
SHALL persist bounded source lifecycle and terminal event evidence in its own
journal. Every retained event SHALL preserve mailbox source identity and its
source-local ordering, including message, delivery, lease, attempt/retry
causation and terminal/lifecycle classification when available. The mailbox
reader SHALL expose retention floor and terminal recovery facts without
transferring queue, worker, handler, authorization or business-commit ownership
to a host bridge or event adapter.

#### Scenario: Lifecycle evidence is retained by mailbox source
- **WHEN** a retained mailbox message is admitted, leased, retried, acknowledged,
  dead-lettered, expires or reaches a source terminal outcome
- **THEN** the mailbox source records its bounded lifecycle/terminal evidence in
  source-local order
- **AND** the record preserves original correlation and causation rather than
  creating a replacement parent Run

#### Scenario: Retained history excludes unsafe bodies
- **WHEN** mailbox lifecycle evidence is written to the retained history
- **THEN** bounded identity, lifecycle and terminal references are retained
- **AND** business payloads, credentials, callback errors and a host
  business-commit assertion are not retained as history facts

### Requirement: Mailbox cursor replay SHALL be read-only and fail closed

A retained mailbox source SHALL provide bounded read-after-cursor pages and
source-scoped opaque cursors. It SHALL distinguish malformed, foreign, expired,
gapped and unavailable cursor/source outcomes with the native history typed
errors. A cursor read MUST NOT publish, claim, renew a lease, acknowledge,
negative-acknowledge, requeue, retry, dead-letter, invoke a handler or mutate a
mailbox record.

#### Scenario: Restart replay returns later source events only
- **WHEN** a consumer restarts after confirming a mailbox cursor
- **THEN** reading after that cursor returns the later retained mailbox events
  in source-local order with a next cursor or complete indication
- **AND** the logical message/admission and delivery attempt counts do not
  increase

#### Scenario: Retention loss is not hidden
- **WHEN** a mailbox cursor falls before the retained history floor or crosses a
  declared retention discontinuity
- **THEN** the reader returns `ErrCursorExpired` or `ErrHistoryGap` with its
  applicable floor evidence
- **AND** it does not substitute the first retained event or the latest event
