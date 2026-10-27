# Source History Projection Ledger

`core/history` owns source-local retained facts and the optional explicit
`ProjectionLedger`. A ledger stores bounded source commit identity, digest,
terminal evidence and opaque projection acknowledgements; it does not own
business writes, execution, retry, resume, or a Danny projection database.

`Journal` remains usable without a ledger and then advertises retained history
only. `WithProjectionLedger` is opt-in. A file-backed journal plus a
file-backed ledger can advertise same-host single-writer sequential-reopen
projection recovery. Memory stores and history-only journals keep recovery
unsupported with a reason. If the source journal is committed but ledger
persistence cannot be confirmed, append returns `ErrProjectionRecoveryRequired`
and callers must recover from source evidence or escalate; they must not rerun
the uncertain effect.

After a fresh-process reopen, bind the matching file-backed ledger explicitly
and call `Journal.ReconcileProjection` with the event read from that journal.
The first call persists one `SourceCommitRecord`; an identical call returns
`ProjectionRecoveryResult.Duplicate=true` without advancing sequence or
retention. The method validates source identity, canonical event identity,
sequence, detached payload, terminal evidence and source presence before any
ledger write. It returns typed foreign/conflict/expired/gap/unavailable errors
and never runs, retries, resumes, acknowledges delivery, or writes a business
projection. Mailbox lifecycle history is not a substitute for this binding.
