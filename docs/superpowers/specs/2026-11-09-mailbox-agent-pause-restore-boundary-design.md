# Mailbox Agent Pause Restore Boundary

The mailbox is already the source owner for message delivery and has an opt-in
handler continuation implementation. The approved design makes that path a
declared public capability only when a durable FileStore, registered resumer,
revalidation callback, and owner evidence are present. The default worker stays
lifecycle-only, and mailbox lifecycle replay stays distinct from Agent
execution restore.

The checkpoint is reference-only and binds the original message, delivery,
lease, attempt, host-plan identity, pause reason, pending action, completed
effect references, budget, deadline, expiry, and generation lineage. Fresh
process restore claims the original source record and invokes a registered
resumer once. A second pause writes a successor checkpoint instead of rerunning
the original handler. Ack, requeue, nack, retry, DLQ, lease loss and recovery
required remain source lifecycle outcomes and never become a host business
commit.

The capability is explicitly limited to same-host, single-writer, sequential
reopen. Manifest promotion is per pause reason and per owner; planbridge only
delegates to the owner and cannot create a worker, queue, scheduler, or restore
loop. Evidence consists of fresh-process tests for all three pause reasons,
negative admission tests, no-effect-rerun counters, duplicate convergence,
terminal policy tests, a replay fixture, and a dedicated gate.
