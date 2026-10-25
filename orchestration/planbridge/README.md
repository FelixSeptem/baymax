# Host execution plan bridge

`planbridge` is an opt-in compiler and thin native-owner delegation API for
`host_execution_plan.v1`. It does not own authorization, revision acceptance,
business commit, retries, budgets, terminal reduction, or durable recovery.

The public API is `Describe(Descriptor, Binding, Limits)` to populate normalized
reference-only `SourceFacts`, `Compile(Descriptor, Binding, Limits)` to validate
those independently supplied facts, and `Compiled.Run/Stream(ctx, Execution)`.
`Identity()` and `Canonical()` return detached identity and canonical bytes.
`Binding` selects exactly one of `Direct`, `Team`, `Workflow`, `Child` or the
source-owned `Mailbox` binding and supplies
a mandatory live `Admit` callback. Compilation never invokes callbacks.

Axes are independent: direct/subagent/team/workflow, inline/mailbox, and
manual/scheduled/resumed/callback. Only inline direct, terminal-only parallel/vote
team, flat workflow, isolated local child, source-owner child continuation, and manifest-approved `mailbox_async`
are supported. Mailbox requires a compatible source-owned manifest and native
binding; resumed, durable continuation, subgraphs and shared child scheduler
bindings fail before activation. Scheduled/callback labels create no workers.
`ChildBinding` accepts
no queue, scheduler or composer: activation constructs a private memory store,
scheduler and composer, used by one dispatch and never exposed. No bool supplied
by a caller can assert isolation. Child dispatch has no token streaming;
`token_streaming` requirements are supported only for direct. Existing owners
remain the sole validators and execution state machines.

Default constructor-only limits are 64KiB descriptor canonical bytes, 128
structural items, 1024 dependencies, 1024 UTF-8 bytes per reference, nesting 8,
and a separate 64KiB copied binding budget. Zero chooses defaults; positive
values may only tighten them. JSON-like payloads support primitive values,
string-keyed maps and slices; pointers, custom structs/MarshalJSON, cycles and
excessive nesting fail before cloning. Maps and slices are detached. Services,
stores, engines and callback closures cannot be cloned: the host must not mutate
their semantic configuration concurrently and must report current semantics in
`Admission`. Input/handler digests are declarations, not authentication.

`Admission` returns current authority/tool/semantic digests and, for children,
current parent depth, active children and remaining budget. Denial errors pass
through unchanged. Digest drift requires recompilation. Source readiness/action
gates still execute. Children also reuse native guardrails/timeout resolution and
native parent clamp; an exhausted live budget fails before enqueue, avoiding the
composer's historical missing-budget fallback. A synchronous local callback must
honor its source timeout contract: composer does not forcibly interrupt it.

`Execution` may supply a native `EventHandler` and independent observer sink.
Actual source callbacks are forwarded through `host/eventadapter`; host plan
version/digest are separate references. No parent or terminal event is created
from a result. Each activation has its own observer. Result fields are tagged
native direct/team/workflow/child results, never a cross-owner terminal enum.
Observer failures do not rewrite outcomes. RuntimeRecorder remains the sole
diagnostic writer. `TerminalOnly` guards runner adapters against unexpected
input_required and retains the native result/checkpoint through `ContinuationError`.
For child restore, `ChildBinding.TerminalOnly=false` is accepted only with a
source-owned `ChildResumeRequest` and `ChildContinuationOwner`; activation calls
that owner once and returns its native projection. Missing owner or incompatible
identity is rejected before activation. Team, mailbox and resumed-plan
continuation cells remain explicitly unsupported.
Only direct preserves input_required as native pause. Resume uses native APIs
separately; this compiled snapshot cannot restore a Run.

Errors expose `Code`, a safe field path and optional native `Unwrap`, never
payloads: unsupported_version, invalid_plan, limit_exceeded, binding_mismatch,
unsupported_capability and source_unavailable. The capability manifest is
versioned/evidence-linked and explicitly marks source-history, checkpoint and
pause dimensions with their evidenced supported/unsupported state. Canonical SHA256 covers declared refs/digests and matched
source-normalized structural/default/budget facts, with sorted map keys,
preserved semantic list order and durations in integer nanoseconds.

For `mailbox_async`, `MailboxBinding` requires a compatible immutable manifest,
the existing mailbox owner and a complete `mailbox.HostPlanBinding`. Compile
derives the binding's `PlanDigest` from the descriptor/source-fact projection;
a supplied differing digest is a pre-publish `binding_mismatch`. Activation
rechecks the manifest and owner, then calls `AdmitHostPlan` once. `Result.Mailbox`
is a detached source projection: original run/parent run, delivery/lease IDs, attempt causation,
lifecycle result/reason and terminal outcome stay distinct from an application
business commit. `Run` and `Stream` can return duplicate projections for the
same logical message but never manufacture a second parent Run.

Rollback: stop calling the bridge and call native APIs explicitly. Preserve
task/attempt/checkpoint/outbox and business facts; no storage migration or undo
of already executed effects is provided. See the offline
[example](../../examples/host-execution-plan/README.md).

`DefaultCapabilityManifest` remains a conservative source-unbound declaration.
Use `CapabilityManifestWithHistory(ctx, reader, source)` to obtain concrete
mailbox source facts from the existing owner: only a matching source-bound
capability can enable after-cursor replay, and durable history additionally
requires durable persistence. `ExecutionFormCapability.SourceHistory` preserves
the declared source/event namespace, sequence scope, retention/floor,
cursor scope/version and terminal-recovery limits. MemoryStore replay remains
process-local; unavailable or incompatible source facts fail closed. Runner,
child, team and workflow history is enabled only by an explicit source-owned
journal fact; the default manifest remains unsupported until that owner is
configured. No history capability enables checkpoint creation, cross-process
execution restore or composite pause. The bridge neither owns nor reads a
journal during plan dispatch.

## Team and mailbox continuation

Team parallel/vote and `mailbox_async` continuation are admitted only when the
corresponding source owner contributes a valid continuation capability for the
requested checkpoint, restore and pause-reason cells. A File-backed owner has
the limited `same_host_single_writer_sequential_reopen` profile; a memory-only
owner remains process-local and planbridge rejects cross-process continuation
before member activation or mailbox mutation.

The bridge delegates team restore to the configured team owner and mailbox
claim/projection to the existing mailbox owner. It does not restore a plan on
its own, publish or consume a message, make a second mailbox worker, re-run a
parent or treat history replay as a checkpoint. Lease loss, retry, DLQ, timeout
and terminal state remain source-owned evidence. Resumed planbridge execution is
still a distinct unsupported capability until it has its own source graph
material.

## Deployment scope admission

`Descriptor.Deployment` is an additive request for an explicitly evidenced
deployment boundary. Compilation and admission reject multi-writer,
cross-machine or multi-instance requests when the bound source only exposes
same-host single-writer sequential reopen. Rejection occurs before owner
activation, publish, enqueue, claim, restore or fallback direct execution.
