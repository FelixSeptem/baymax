# Host execution plan bridge

`planbridge` is an opt-in compiler and thin native-owner delegation API for
`host_execution_plan.v1`. It does not own authorization, revision acceptance,
business commit, retries, budgets, terminal reduction, or durable recovery.

The public API is `Describe(Descriptor, Binding, Limits)` to populate normalized
reference-only `SourceFacts`, `Compile(Descriptor, Binding, Limits)` to validate
those independently supplied facts, and `Compiled.Run/Stream(ctx, Execution)`.
`Identity()` and `Canonical()` return detached identity and canonical bytes.
`Binding` selects exactly one of `Direct`, `Team`, `Workflow`, `Child` and supplies
a mandatory live `Admit` callback. Compilation never invokes callbacks.

Axes are independent: direct/subagent/team/workflow, inline/mailbox, and
manual/scheduled/resumed/callback. Only inline direct, terminal-only parallel/vote
team, flat workflow, and isolated local child are supported. Mailbox, resumed,
durable continuation, subgraphs and shared child scheduler bindings fail before
activation. Scheduled/callback labels create no workers. `ChildBinding` accepts
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
Only direct preserves input_required as native pause. Resume uses native APIs
separately; this compiled snapshot cannot restore a Run.

Errors expose `Code`, a safe field path and optional native `Unwrap`, never
payloads: unsupported_version, invalid_plan, limit_exceeded, binding_mismatch,
unsupported_capability. Canonical SHA256 covers declared refs/digests and matched
source-normalized structural/default/budget facts, with sorted map keys,
preserved semantic list order and durations in integer nanoseconds.

Rollback: stop calling the bridge and call native APIs explicitly. Preserve
task/attempt/checkpoint/outbox and business facts; no storage migration or undo
of already executed effects is provided. See the offline
[example](../../examples/host-execution-plan/README.md).
