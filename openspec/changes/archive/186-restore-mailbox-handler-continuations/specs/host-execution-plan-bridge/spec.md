## ADDED Requirements

### Requirement: Mailbox handler continuation admission SHALL delegate to matching owner evidence

For a `mailbox_async` plan that requires host-handler continuation, the bridge
SHALL validate compatible manifest version, form, pause reason, owner namespace,
deployment scope, and current mailbox handler continuation availability before
publish, consume, handler dispatch, checkpoint claim, Run, or Stream. It SHALL
delegate admission and restore only to the configured mailbox owner and preserve
the owner's original message/delivery/lease/attempt, causal, and terminal
semantics.

The bridge MUST NOT create a worker, queue, scheduler, handler executor,
continuation store, replacement parent Run, or lifecycle fallback. Missing,
foreign, stale, unavailable, unsupported, or version-incompatible evidence
MUST fail closed before source side effects.

#### Scenario: Run and Stream share one mailbox continuation decision
- **WHEN** equivalent Run and Stream requests require a configured mailbox
  handler continuation cell
- **THEN** both validate and delegate the same owner admission/restore decision
- **AND** their normalized terminal/correlation projection is equivalent while
  Stream only changes source event delivery

#### Scenario: Unsupported handler continuation rejects before mailbox mutation
- **WHEN** a mailbox plan requires a pause reason without matching handler
  safe-point evidence
- **THEN** the bridge returns typed `unsupported_capability` or unavailable
  before publishing, consuming, invoking a handler, or claiming a checkpoint
- **AND** it does not degrade to direct execution or lifecycle replay
