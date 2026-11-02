## Purpose

Provide source-bound external fact delivery with explicit consumer confirmation,
durable cursor progress, bounded evidence, and cooperative host-owned lifecycle,
without changing execution, native history, or business-effect ownership.

## ADDED Requirements

### Requirement: Source admission SHALL bind metadata and external facts
The lifecycle MUST use explicit source, consumer and state dependencies.
Source identity, identity namespace and cursor version MUST be nonempty,
bounded UTF-8 references. Cursor envelopes, pages and facts MUST match the
admitted metadata; malformed or foreign values MUST fail before delivery.
External source identity MUST NOT be inferred from native callback IDs.

#### Scenario: Foreign metadata is rejected before consumption
- **WHEN** a page, fact or saved state belongs to a different source or cursor version
- **THEN** the lifecycle rejects it and preserves confirmed progress without invoking the consumer

### Requirement: Immutable identities SHALL have bounded conflict evidence
Facts MUST contain bounded external identity, reference, digest, payload and
source-bound next cursor. The digest MUST match the payload. Before delivery,
the lifecycle MUST persist a pending receipt without advancing the confirmed
cursor. Pending and the bounded recent confirmed receipt window MUST detect
same-identity conflicting facts before effect. Identical repeats MUST be
offered with the same identity; host effect CAS remains the idempotency owner.
Evidence outside the declared bounded window MUST NOT imply global deduplication.

#### Scenario: Crash before cursor commit preserves immutable pending evidence
- **WHEN** a consumer has performed its effect but the confirmed state write fails
- **THEN** reopening retains the prior cursor and pending identity so an identical offer can be retried through host CAS and a changed fact is rejected

#### Scenario: Page conflict prevents partial effects
- **WHEN** a bounded page contains duplicate external identity with changed facts
- **THEN** validation rejects the whole page before any consumer call

### Requirement: Cursor progress SHALL follow finite consumer confirmation
Only explicit consumer dispositions `sent` and `gone` SHALL permit cursor
advance. `gone` MUST mean the consumer established permanent inability to
deliver, never an inferred transient timeout. `retryable`, errors and
cancellation MUST leave that fact's confirmed cursor unchanged and stop the
current page. Earlier confirmed facts in that page MUST remain committed.
Polling and callback offers MUST share these semantics.

#### Scenario: Retryable stops after the confirmed prefix
- **WHEN** a page has a sent fact followed by a retryable fact and another fact
- **THEN** only the first cursor is confirmed, the last fact is not consumed, and retry starts from the first cursor

#### Scenario: Gone confirms progress without claiming effect
- **WHEN** a consumer confirms permanent non-deliverability with gone
- **THEN** the lifecycle commits that source cursor without claiming business success

### Requirement: Lifecycle health and interruption SHALL remain bounded
Health MUST expose finite classification, bounded identity/reference and
saturating counts without raw payload/error bodies. Polling and consumer work
MUST have bounded cooperative contexts. Dispose and Abort MUST be idempotent,
cancel only lifecycle operations, reject later operations and never themselves
write cursor state or control a Run. Concurrent operations MUST fail explicitly.
An already admitted atomic store commit may complete during cancellation; its
truth MUST be determined by reloading the store, not inferred from cancellation.

#### Scenario: Abort interrupts unconfirmed delivery
- **WHEN** abort cancels a cooperative consumer before confirmation
- **THEN** confirmed cursor is unchanged and repeated shutdown has no new side effects

### Requirement: Durable state SHALL declare its deployment and restart scope
State MUST include versioned source metadata, revision, confirmed cursor and
bounded identity evidence. Memory state MUST claim process-local scope only.
The supplied file store MUST support atomic same-host single-writer sequential
reopen, reject corrupt, foreign or over-bound envelopes without overwrite, and
MUST NOT claim cross-process or cross-machine CAS guarantees.

#### Scenario: Fresh process reloads the confirmed prefix
- **WHEN** a new process opens valid file state after a confirmed prefix and an unconfirmed fact
- **THEN** it starts at the confirmed cursor and preserves pending evidence for the next offer

### Requirement: Adapter conformance SHALL preserve existing owners and replay purity
The lifecycle MUST NOT become a native history writer, execution terminal owner,
global source registry, background polling worker, business-effect ledger or
diagnostics writer. Source-local order MUST NOT imply global order. Offline
versioned replay MUST validate bounded evidence and reject cursor/disposition
drift without source polling, consumers, tools, models, network or effects.

#### Scenario: Replay detects unconfirmed cursor advance
- **WHEN** a fixture advances cursor after retryable or interrupted delivery
- **THEN** replay reports deterministic drift without executing delivery
