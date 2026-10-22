# Composer child continuation

Composer owns child dispatch admission while scheduler remains the owner of
task, attempt, lease, retry and terminal facts. `ChildContinuationStore` is an
explicit source-owned safe-point store: MemoryStore is process-local and
FileStore supports same-host single-writer sequential reopen only.

`ResumeChildContinuation` validates graph/run/session/causation and authority
bindings, atomically claims one logical continuation, checks the existing
scheduler task and attempt, and delegates pending work to the injected source
executor. It never calls `SpawnChild`, enqueues a replacement task, creates a
parent run, or replays history. Duplicate terminal claims return the retained
projection; lease-loss, stale, corrupt, expired, unavailable and in-progress
states fail closed.

The three pause reasons are independent capability cells. A host must provide
source evidence for `input_required`, `approval_required`, or
`budget_exhausted`; scheduler retry and recovery snapshots are not pause
continuations. Team, mailbox and resumed-plan support remains unsupported.

Child continuation stores expose the same-host single-writer sequential-reopen
deployment boundary. The owner namespace and fence are source facts; a
foreign/stale writer or distributed graph-claim request is rejected before
member dispatch and is never converted into a fresh parent or replay.
