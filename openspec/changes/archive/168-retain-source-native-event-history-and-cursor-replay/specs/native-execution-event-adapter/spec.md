## ADDED Requirements

### Requirement: Event adapter SHALL not become a retained history owner

The native event adapter SHALL remain a real-time callback projection. A history
cursor or event returned by a source-owned history reader may be projected only
as an already-retained source fact, with its source identity, source sequence
scope and cursor outcome preserved separately from callback mapping references.
The adapter MUST NOT retain application history, mint a journal identity,
translate callback EventID into a cursor, create a replay worker, or invoke a
history read as part of Run or Stream.

#### Scenario: Retained mailbox event is projected without identity rewrite
- **WHEN** a host projects an event previously returned by a mailbox history
  reader
- **THEN** the projection retains the mailbox source identity and declared
  local sequence scope
- **AND** adapter callback references do not replace or become a replay cursor

#### Scenario: Adapter cannot satisfy a history request by itself
- **WHEN** a consumer asks an adapter-only direct callback source for
  read-after-cursor replay
- **THEN** the request fails with the source history unsupported outcome
- **AND** the adapter creates no in-memory history, subscription or latest
  fallback
