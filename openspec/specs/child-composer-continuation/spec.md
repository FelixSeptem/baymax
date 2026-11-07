# child-composer-continuation Specification

## Purpose
This capability defines how a Composer child owns bounded pause checkpoints and safely restores the same child execution after a process boundary using explicit native source evidence.

## Requirements

### Requirement: Child continuation checkpoints bind native source position

A child owner that advertises continuation SHALL persist a bounded checkpoint containing parent and child identities, assignment/task and attempt identity, pause reason, consumed budget and original deadline, completed effect references, and a stable child source reference and opaque cursor. The checkpoint SHALL bind its deployment scope and SHALL NOT contain credentials, callbacks, or unbounded application payloads.

#### Scenario: Paused child retains source position
- **WHEN** a child reaches an advertised pause safe point
- **THEN** its checkpoint retains parent/child identity, assignment, budget/effect boundary, and source-local cursor

#### Scenario: Cursor is foreign or malformed
- **WHEN** a restore request supplies a cursor from another source or a malformed cursor
- **THEN** the owner returns the corresponding typed history error before claiming or activating the child

### Requirement: Child restore requires fresh-process source evidence

A child continuation capability SHALL be promoted only from an explicitly configured child owner whose checkpoint and source history can be reopened by a fresh Composer process under the declared deployment scope. Restore SHALL use the original checkpoint and child task/attempt lineage, and SHALL NOT invoke a new parent Run, enqueue a replacement child, or treat history replay as executable state.

#### Scenario: Fresh Composer restores the original child
- **WHEN** a new Composer opens the configured owner store and receives a valid child checkpoint reference
- **THEN** it restores the same child lineage and only the persisted pending boundary

#### Scenario: Missing source facts fail closed
- **WHEN** the checkpoint's source facts are unavailable, expired, or contain a sequence gap
- **THEN** restore returns a typed unavailable, expired, or gap error and performs no child/tool effect

### Requirement: Child pause capabilities are independent evidence cells

The child capability declaration SHALL report `input_required`, `approval_required`, and `budget_exhausted` independently. A cell SHALL be supported only when its safe point, durable checkpoint, fresh-process restore, source cursor compatibility, terminal behavior, and completed-effect non-replay are evidenced for that reason. Undeclared or incomplete cells SHALL remain `unsupported_capability`; child execution wiring SHALL NOT imply continuation support or direct fallback.

#### Scenario: One pause cell does not promote another
- **WHEN** a child owner has fresh-process evidence for `input_required` but not `approval_required`
- **THEN** only `input_required` may be advertised and approval admission is rejected before activation

#### Scenario: Completed effect is not repeated after restore
- **WHEN** a child checkpoint records a completed tool/effect reference and the child is restored
- **THEN** the completed effect is not invoked again and the original effect identity remains in the resulting evidence

#### Scenario: Incomplete capability does not fall back to direct
- **WHEN** a subagent/fanout plan requests a pause cell without complete child evidence
- **THEN** planning returns `unsupported_capability` without converting the request to direct execution

### Requirement: Child history replay is bounded and read-only

Child source history SHALL expose a stable source-local cursor, retention floor, and bounded `ReadAfter`. A cursor preceding retained history SHALL return a typed expired error; a discontinuity SHALL return a typed gap error. Replay SHALL NOT resume, retry, enqueue, invoke tools, or mutate child terminal state.

#### Scenario: Retained cursor replays bounded facts
- **WHEN** a caller reads after a retained child cursor with a valid page limit
- **THEN** the source returns only bounded child facts, the next cursor, and retention floor without executing work

#### Scenario: Expired cursor is not advanced silently
- **WHEN** a requested cursor is older than the source retention floor
- **THEN** the source returns a typed expired error rather than returning the latest position
