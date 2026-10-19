## ADDED Requirements

### Requirement: History replay fixtures SHALL remain source-reference-only

Diagnostics replay tooling SHALL support versioned, bounded native history
fixtures that verify source capability declarations, history identity, local
sequence, cursors, retention floors, typed failures and terminal recovery
evidence. It MUST treat fixture replay as a pure validation operation and MUST
NOT instantiate a mailbox owner, runner, team, workflow, handler, model, tool,
host callback, retry, restore or business write. A diagnostics record or adapter
callback EventID SHALL NOT be accepted as a synthetic history fixture identity.

#### Scenario: Fixture validates restart cursor evidence without execution
- **WHEN** offline tooling replays a mailbox history fixture with a confirmed
  cursor and later terminal event
- **THEN** it validates the returned source page and next cursor deterministically
- **AND** no execution, delivery lifecycle transition or diagnostics write occurs

#### Scenario: Fixture rejects cursor identity drift
- **WHEN** a fixture changes a cursor's source namespace, event identity,
  retention floor or terminal classification
- **THEN** tooling reports deterministic history/cursor drift
- **AND** it does not normalize that drift as callback fragmentation or a host
  observation difference
