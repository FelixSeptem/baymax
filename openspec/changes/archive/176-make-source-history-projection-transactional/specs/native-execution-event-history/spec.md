## ADDED Requirements

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
