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

## Active host-context assembly boundary

`establish-host-context-assembly-handoff-contract` is active. Its implemented `host_context_projection.v1` correlation is additive and reference-only: schema/identity/version/digest, capability evidence, optional plan correlation and opaque source references accompany the legal finalized, checkpoint, tool-finalized and flushed-stream cuts. The admitted descriptor is bounded by the fixed constructor/API limits (32 KiB serialized, 64 references, 256 UTF-8 bytes/reference and 128 UTF-8 bytes/digest). Resolved bodies, private memory/knowledge, credentials, business authorization and effect payloads remain excluded; legacy consumers continue to see a nullable zero-value field.

Correlated restore retains existing schema/cut/reference validation first, then validates projection correlation/capability narrowing and requires `runner.RestoreHandoffWithHostProjection` with a fresh current-authority host resolver. It returns transient resolution for caller-owned injection only; the runtime does not inject context, resume, or run itself. `host_context.*` failures are finite and fail closed; they never disclose a body or raw resolver error. A historical descriptor or handoff digest is not authorization, replay is not restore, and neither operation may create a business commit or a parallel recovery owner. Resolver-independent correlated restore is rejected. Dedicated replay, dual-platform contract gates, and their static standard-quality-path wiring are delivered, while task 6.3's fresh successful suite execution remains pending.
