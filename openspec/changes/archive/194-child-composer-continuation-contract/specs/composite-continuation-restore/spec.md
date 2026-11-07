## MODIFIED Requirements

### Requirement: Child and fanout restore SHALL not re-enqueue or rerun completed effects

When the child/fanout source advertises restore, a new Composer SHALL reopen the configured source store, restore the same native child graph, task/attempt lineage, and checkpoint-bound source position, skip completed members/effects, and continue only pending work. Restore MUST NOT call a new parent Run, create a replacement task, enqueue a duplicate, or use history replay as execution state. A missing, expired, foreign, or gapped source cursor MUST fail closed with a typed outcome before activation.

#### Scenario: New Composer restores pending local child
- **WHEN** a local child continuation and its source cursor are persisted and a new Composer opens the same compatible store
- **THEN** it resumes the original task/attempt lineage from the checkpoint-bound position and does not spawn a second task or repeat completed effect outcomes

#### Scenario: Child lease loss is source evidence
- **WHEN** a child lease expires before restore
- **THEN** admission returns the native lease/recovery outcome and does not silently reclaim, retry, or re-run the parent

#### Scenario: Child history cursor has expired or a gap
- **WHEN** a restore cursor is below the retention floor or source history has a discontinuity
- **THEN** restore returns typed expired or gap evidence before activating child work
