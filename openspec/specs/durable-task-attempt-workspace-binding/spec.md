# durable-task-attempt-workspace-binding Specification

## Purpose
This capability makes scheduler task and attempt execution auditable against bounded workspace provenance without owning workspace contents or lifecycle. It prevents stale lease, retry, and recovery paths from silently applying results to an unrelated workspace.

## Requirements

### Requirement: Task and attempt workspace binding SHALL be reference-only and additive

Scheduler task and attempt records MUST be able to carry an optional workspace binding containing workspace identity, change-set identity, integrity references, and producing Run/Step correlation. The binding MUST remain reference-only, nullable, defaultable, and MUST NOT embed workspace contents or create a workspace store.

#### Scenario: Legacy task without workspace binding remains valid
- **WHEN** a task or snapshot omits workspace binding fields
- **THEN** existing enqueue, claim, retry, terminal commit, query, and restore behavior remains valid with workspace binding absent

#### Scenario: Valid binding is retained across claim
- **WHEN** a queued task with a valid workspace binding is claimed
- **THEN** the active attempt retains the same workspace and change-set identity and the binding remains associated with the current task/attempt correlation

### Requirement: Lease rollover and retry SHALL isolate stale workspace results

When an attempt expires, is requeued, or is superseded by a newer attempt, a terminal result carrying the stale attempt or stale workspace binding MUST be rejected or classified as stale/conflict before mutating the current task result. Retry semantics MUST explicitly identify whether the binding is reused after integrity validation or replaced by a new binding.

#### Scenario: Expired attempt cannot commit against a newer attempt
- **WHEN** attempt A expires, attempt B is claimed, and a terminal commit arrives for attempt A
- **THEN** the commit is classified as stale and MUST NOT change attempt B or the task's current result

#### Scenario: Workspace rollover mismatch is detected
- **WHEN** a retry presents a workspace binding different from the binding selected for the current attempt
- **THEN** the scheduler rejects or classifies the mismatch deterministically and records the bounded reason through the existing diagnostics owner

### Requirement: Workspace integrity SHALL be validated at checkpoint and recovery boundaries

Before restoring or reconciling scheduler state that references workspace provenance, the source recovery path MUST validate workspace association and observed integrity. Missing, dirty, conflict, checkpoint-association mismatch, and integrity-drift outcomes MUST be distinguishable, deterministic, and fail-fast under strict restore policy; compatible restore MAY continue only with a bounded downgrade classification.

#### Scenario: Strict restore rejects integrity drift
- **WHEN** observed workspace integrity differs from the provenance expected before restore
- **THEN** strict restore fails before mutating scheduler, mailbox, or recovery state and emits the existing workspace integrity drift classification

#### Scenario: Compatible restore records bounded downgrade
- **WHEN** a compatible restore encounters a recoverable workspace mismatch within the configured compatibility policy
- **THEN** restore skips or bounds the affected segment, records a deterministic downgrade classification, and does not silently claim workspace equivalence

### Requirement: Workspace binding SHALL survive snapshot and replay normalization

Scheduler snapshots, composer recovery snapshots, protocol projections, and diagnostics replay MUST preserve workspace binding references and classify binding, association, and integrity drift without persisting raw workspace content or changing source-of-truth ownership.

#### Scenario: Snapshot restore preserves binding identity
- **WHEN** a scheduler/recovery snapshot containing workspace-bound task attempts is exported and restored
- **THEN** task, attempt, lease, workspace, change-set, and checkpoint correlations remain equivalent after normalization

#### Scenario: Replay detects workspace binding drift
- **WHEN** replay input changes workspace identity, change-set identity, checkpoint association, or integrity reference
- **THEN** replay returns the corresponding bounded drift classification without invoking providers, tools, Git, or workspace mutation
