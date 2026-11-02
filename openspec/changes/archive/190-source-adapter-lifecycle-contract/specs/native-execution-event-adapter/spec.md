## ADDED Requirements

### Requirement: Source delivery lifecycle SHALL remain independent of callback observation
The synchronous callback Observer SHALL retain no durable cursor, pending
receipt, consumer disposition or source delivery retry ownership. Restartable
external delivery MUST use explicit source lifecycle dependencies; observer
status, callback EventID and TraceID MUST NOT become durable source identity.
Source adapter lifecycle MUST NOT change native callback delivery semantics.

#### Scenario: Observer is not a durable source lifecycle
- **WHEN** an integration attempts to use only the callback Observer as a restartable polling source or state store
- **THEN** that capability is unsupported without manufacturing cursor state
