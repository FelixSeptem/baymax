# Workflow orchestration

Workflow owns deterministic graph planning, step dependencies, retry/timeout and source-native continuation facts. The workflow-first continuation contract persists graph identity/version/digest, parent/run/session/causation, authority/plan/write-scope/effect/occurrence bindings, pending and completed steps, consumed budgets, pause boundary, completed effect references, terminal arbitration and checkpoint lineage. A new Engine may restore the same native graph only after manifest, binding, digest, expiry and atomic claim validation; it never re-runs the parent or completed effects.

`NewFileWorkflowContinuationStore` is the durable profile for same-host single-writer sequential reopen. `Engine.ContinuationCapabilities` advertises workflow checkpoint/restore only when this store and the `workflow` owner namespace are configured. Memory stores are test/local-only and return unavailable for cross-process capability advertisement. Child/fanout, team, mailbox and planbridge resumed execution must remain explicitly unsupported until their own source owner has equivalent restore material and evidence.

The supported pause matrix is evidence-driven for `input_required`, `approval_required` and `budget_exhausted`, with Run/Stream parity and cancel/timeout/retry race semantics. Child/fanout, team parallel/vote, mailbox host-plan and planbridge resumed forms remain `unsupported_capability`. The default deployment scope is same-host single-writer sequential reopen; this README does not claim distributed CAS or multi-instance recovery.

Workflow continuation capability evidence includes owner fence and transaction
scope. Reopening a FileStore on the same host is supported only after the
previous writer has released ownership; concurrent, cross-machine and
multi-instance restore remains explicitly unsupported.

After reopening, any persisted `claimed` or `running` resume attempt without a
committed result is `recovery_required`. The workflow owner never automatically
redispatches a step whose effect may have occurred. Exact duplicates return a
committed result when one exists; otherwise they return the retained recovery
state. History append failures are separately surfaced as degraded/unavailable
source evidence and do not authorize rerunning the workflow.
