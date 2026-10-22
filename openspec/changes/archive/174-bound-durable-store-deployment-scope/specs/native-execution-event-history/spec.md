## ADDED Requirements

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
