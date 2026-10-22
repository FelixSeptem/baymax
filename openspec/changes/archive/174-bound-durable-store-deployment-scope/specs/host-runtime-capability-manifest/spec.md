## ADDED Requirements

### Requirement: Capability entries SHALL declare deployment and writer scope

Every source-history, checkpoint, continuation, and mailbox capability entry
MUST declare a bounded deployment profile, writer ownership mode, fencing
version, and transaction scope. The default supported profile SHALL be
`same_host_single_writer_sequential_reopen`; multi-writer, cross-machine,
distributed-journal, and multi-instance resume claims MUST be explicit cells
and MUST remain unsupported when no source evidence exists.

#### Scenario: Planner distinguishes sequential reopen from distributed restore
- **WHEN** a planner reads a FileStore-backed continuation capability
- **THEN** it receives the same-host single-writer profile and its fencing and
  transaction scope
- **AND** it does not infer cross-machine or concurrent restore support

#### Scenario: Unsupported deployment scope is explicit
- **WHEN** a host requests a multi-writer or cross-machine capability without
  source-owned evidence
- **THEN** the manifest returns `unsupported_capability` with a stable reason
- **AND** admission does not enqueue, publish, claim, resume, or replay

### Requirement: Capability evidence SHALL identify ownership and fencing

A supported deployment declaration MUST identify the source owner, writer
ownership token or equivalent bounded fence reference, snapshot/journal
transaction scope, and the evidence revision that proves the declaration.
These references MUST be detached and MUST NOT contain payloads, credentials,
or host business authority.

#### Scenario: Stale writer is rejected before activation
- **WHEN** a request presents a writer fence older than the source's current
  owner fence
- **THEN** admission returns a typed stale-owner or unavailable result
- **AND** no source lifecycle or restore transition is performed
