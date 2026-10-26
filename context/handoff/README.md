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

`establish-host-context-assembly-handoff-contract` is archived as the direct-path baseline. Its implemented `host_context_projection.v1` correlation is additive and reference-only: schema/identity/version/digest, capability evidence, optional plan correlation, opaque source references, and optional source-binding identity accompany the legal finalized, checkpoint, tool-finalized and flushed-stream cuts. A source binding carries only form, owner/source/node, logical continuation identity/digest and generation; it never carries a resolved body or an authorization decision. The admitted descriptor is bounded by the fixed constructor/API limits (32 KiB serialized, 64 references, 256 UTF-8 bytes/reference and 128 UTF-8 bytes/digest). Resolved bodies, private memory/knowledge, credentials, business authorization and effect payloads remain excluded; legacy consumers continue to see a nullable zero-value field.

Correlated restore retains existing schema/cut/reference validation first, then validates projection correlation/capability narrowing and requires `runner.RestoreHandoffWithHostProjection` with a fresh current-authority host resolver. It returns transient resolution for caller-owned injection only; the runtime does not inject context, resume, or run itself. `host_context.*` failures are finite and fail closed; they never disclose a body or raw resolver error. A historical descriptor or handoff digest is not authorization, replay is not restore, and neither operation may create a business commit or a parallel recovery owner. Resolver-independent correlated restore is rejected. Dedicated replay and dual-platform gates remain the direct-path baseline. Composite source/node/checkpoint binding is an active OpenSpec: child attempt, team pending local member, and workflow pending-step owner APIs have binding/fresh-resolver evidence, but default manifest and planbridge cells remain unsupported until every promoted owner has complete evidence. Mailbox continuation only projects lifecycle outcome and has no handler execution safe point, so it cannot borrow direct correlation or be treated as context execution restore.
