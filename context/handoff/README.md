# Runtime handoff contract

`context/handoff` is a bounded projection, not a source-of-truth store.

| Handoff field | Source owner |
| --- | --- |
| `run_id`, `session_id`, cut identity | Runner / protocol Run and Session projections |
| objective, completed, pending, failed attempts | Run/Step/Event timeline and Context Assembler |
| file changes, tool results | Artifact and tool lifecycle owners |
| policy, sandbox, admission state | Policy precedence, sandbox, and admission owners |
| checkpoint/history/snapshot references | Checkpoint, Session History, and snapshot manifest owners |
| facts and provenance | Observed events and source metadata only |
| inferences and confirmation needs | Handoff projection, always carrying source IDs for inferences |

The package does not load or persist referenced bodies. Restore callers resolve references through the existing owner and inject the bounded projection through the Context Assembler/reference-first path.
