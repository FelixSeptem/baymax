# Host context projection contract

`context/hostprojection` is the provider-neutral, reference-only semantic
anchor for explicit host context assembly. `Admit` accepts only a detached
`host_context_projection.v1` descriptor with canonical SHA-256 evidence;
`Resolve` calls the host-owned current-authority resolver only after admission;
it returns transient material for the current assembly or correlated restore and
never persists a resolved body.

The fixed constructor/API limits are 32 KiB serialized descriptor, 64 opaque
references, 256 UTF-8 bytes per reference, and 128 UTF-8 bytes per digest.
There is no environment, file, or hot-update control. `Snapshot` and
`Resolution` return defensive copies; source materials are transient and are
never an authorization decision, a body store, or a handoff payload.

Stable rejection classifications are finite: `host_context.unsupported_version`,
`host_context.invalid_descriptor`, `host_context.limit_exceeded`,
`host_context.duplicate_reference`, `host_context.invalid_capability`,
`host_context.digest_mismatch`, `host_context.resolver_denied`,
`host_context.source_not_found`, `host_context.capability_expansion`,
`host_context.parent_evidence_missing`, and
`host_context.invalid_plan_correlation`. Invalid structure, bounds, digest drift,
parent/child capability expansion, resolver denial, and missing sources fail
closed. Callback errors and source bodies are not exposed through the
classification result.

Current supported use is direct `Run`/`Stream` assembly through the opt-in
non-serialized `types.RunRequest` fields `HostContextProjection` and
`HostContextResolver`, when the existing Context Assembler is enabled. When it
is disabled, the existing assembler bypass semantics apply and the projection
is not admitted or resolved. `handoff.v1` may retain only correlation evidence.
Legal finalized, checkpoint, tool-finalized, and flushed-stream `handoff.v1`
cuts can retain only bounded correlation evidence. Restore requires
`runner.RestoreHandoffWithHostProjection` and a fresh host resolver after the
existing handoff/source validation; it returns transient material for
caller-owned injection and does not itself inject, resume, or run. Resolver-independent restore,
cross-process continuation, mailbox, workflow/team recovery, and dynamic-action
continuation are unsupported. Dedicated replay, dual-platform contract gates,
their static standard-quality-path wiring, and the offline example are
delivered, but task 6.3's fresh successful suite execution remains pending.

To roll back, stop supplying the opt-in descriptor/resolver and continue using
the existing assembler path. This cannot roll back a host business effect.
