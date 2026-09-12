## Context

See `proposal.md` for the motivation and scope. The current source Runtime exposes an active Run registry with bounded Realtime ingress, while `RunRequest.Input` and `Messages` are startup-only inputs. Host commands can correlate lifecycle and HITL operations, but there is no source-owned contract for additional input during an active Run.

The design must preserve the existing owners: Runner owns execution and safe points, Realtime owns sequence/cursor/interrupt state, Session history and checkpoint owners retain reference-only lineage, terminal arbitration owns the first terminal outcome, policy/readiness/sandbox own authorization, and `RuntimeRecorder` remains the single diagnostic writer. The first profile is process-local and embedded; it does not add runtime configuration or durable input storage.

## Goals / Non-Goals

**Goals:**

- Define one versioned, transport-neutral input contract with explicit `steering` and `follow_up` kinds.
- Provide source-owned, bounded per-Run/per-causal-chain admission and deterministic duplicate, stale, terminal, and backpressure outcomes.
- Apply steering only at a safe decision boundary and promote follow-up only through an existing source-owned Run admission path.
- Preserve cancel, retry, Realtime interrupt/resume, HITL, terminal arbitration, checkpoint lineage, and Run/Stream parity.
- Make input admission, application, and non-application replayable without provider or tool execution.

**Non-Goals:**

- Injecting arbitrary content into an in-flight Provider call, tool execution, or HITL resolver.
- Adding a second Session history, checkpoint, cursor, terminal state machine, or hosted queue.
- Persisting pending input across process restart in the first profile.
- Introducing provider-specific steering APIs, remote gateways, RBAC, multi-tenancy, or new runtime configuration keys.

## Decisions

### 1. Use one transport-neutral input envelope with two explicit kinds

The new capability owns the input envelope and normalized admission outcome. `steering` and `follow_up` are not inferred from timing or payload shape. Every envelope carries a stable input identity, target Session/Run, request/correlation fields, bounded payload, and profile version.

Alternative considered: reuse `HostCommandKindAction` with an action string such as `send_message`. Rejected because it would hide input ordering and application semantics inside a generic lifecycle action, making replay and authorization ambiguous.

### 2. Keep admission and application separate

Admission is a source-owned decision and returns immediately. Application is a later source fact emitted at a safe point. The host receives a command response for admission and a runtime event for `applied`, `not_applied`, or a terminal/stale settlement. This follows the Embedded Host contract's command/event separation and prevents an accepted input from being mistaken for model progress.

Alternative considered: synchronously wait until steering is applied before responding. Rejected because it would block host transport, couple response latency to model/tool execution, and complicate cancellation and disconnect cleanup.

### 3. Use bounded source-owned lanes, not a global queue

The first profile uses source-owned bounded lanes associated with the active Run/causal follow-up chain:

- one pending steering lane per active Run, with stable duplicate detection;
- a bounded FIFO follow-up lane associated with the source-owned causal chain;
- no unbounded host queue, global queue, or diagnostics-backed queue.

Capacity is an embedded source option/profile limit rather than a new runtime configuration key. The effective limit is included in bounded admission facts and fixtures so backpressure remains deterministic. If a terminal transition occurs, the source atomically settles or promotes accepted follow-ups before removing the active control; process shutdown settles them as `disconnected`/`not_applied` and does not persist them in a new store.

Alternative considered: coalesce multiple steering messages into the latest value. Rejected for the first profile because coalescing changes user-visible ordering and makes replay identity ambiguous. Each accepted identity remains individually observable.

### 4. Apply steering only between atomic Runner operations

The Runner checks the steering lane after an atomic model call, tool dispatch, or HITL resolution and before the next model decision. The next model request is assembled with the accepted steering input at that boundary. No code path mutates an in-flight Provider request, tool arguments, Realtime sequence, or pending resolver.

Realtime interrupt/resume and cancel retain precedence over steering at the same boundary: an accepted cancel or terminal commit settles the steering as not-applied; a Realtime interrupt freezes progression and steering waits until the existing resume boundary is valid.

Alternative considered: poll input concurrently inside the Provider stream. Rejected because it would create provider-specific interruption behavior and violate existing Run/Stream and stream-error semantics.

### 5. Promote follow-up through the existing Run owner

Follow-up is never treated as steering for the current Run. It remains pending until the source reaches its declared terminal/idle boundary. At that boundary the source either:

1. rejects or settles the follow-up when policy does not allow promotion; or
2. starts a distinct causal Run/Stream through the existing admission path, preserving Session and input causation while leaving the original terminal Run immutable.

Promotion is source-owned and occurs before the prior active control is released. It does not mutate a terminal Run back to `working` and does not create a second Session history source.

Alternative considered: append follow-up to the current Run's message history after terminal publication. Rejected because it would mutate an immutable terminal Run and bypass existing retry/branch/admission semantics.

### 6. Reuse existing authorization and observability owners

Input admission passes through readiness, policy, sandbox, capability, and host authorization before source mutation. Diagnostics contain only bounded kind, outcome, reason, queue, boundary, and correlation fields. Raw input content, prompt text, or arbitrary metadata is never emitted as a diagnostic or OTel dimension. All facts enter through `observability/event.RuntimeRecorder`.

### 7. Make replay side-effect-free and parity-first

The transcript fixture records command, admission response, source input fact, safe-point/terminal boundary, and normalized terminal outcome. Replay verifies duplicate identity, target correlation, ordering, queue outcome, apply/not-applied classification, and Run/Stream equivalence without invoking a model, tool, provider, or host connection.

The gate scans for global queues, direct Provider prompt injection, host-owned input state, and new terminal/cursor stores. Shell and PowerShell scripts must classify the same failures.

## Risks / Trade-offs

- **[Input timing differs between Run and Stream]** → Define the same semantic safe-point contract and compare normalized boundaries rather than raw callback timing; add explicit parity fixtures for model, tool, HITL, interrupt, and terminal races.
- **[Follow-up promotion creates an unexpected second Run]** → Require explicit source admission and causal correlation; expose promotion as a source event and keep the original terminal Run immutable.
- **[Accepted input is lost during process shutdown]** → Do not claim durability in the first profile; settle the input as `disconnected`/`not_applied` and document that persistence requires a future source-owned contract.
- **[Steering starves a long-running Run]** → Bound the steering lane to one pending item and apply it only at existing Runner boundaries; never poll or interrupt a provider stream indefinitely.
- **[Sensitive input leaks into diagnostics]** → Emit only bounded reason/kind/correlation facts, reject raw payloads in recorder projections, and add high-cardinality scans to the contract gate.
- **[Host capability and authorization drift]** → Treat capability advertisement as availability only and re-run readiness/policy/sandbox authorization for every input command.

## Migration Plan

1. Add the new capability and capability advertisement as opt-in; existing hosts that do not advertise or request it continue using the 134 host contract unchanged.
2. Add source-owned DTOs, admission/apply events, and bounded lanes behind the new capability before enabling host commands.
3. Add Run/Stream tests, replay fixtures, subprocess cases, and shell/PowerShell gates before changing the example markers.
4. Roll back by disabling the capability and rejecting steering/follow-up commands as unsupported; existing Run, Stream, Realtime, HITL, cancel, retry, and terminal behavior remains unchanged.
5. Because pending input is not persisted in the first profile, rollback does not require a storage migration or history rewrite.

## Example Impact Assessment

修改示例。The existing `realtime-interrupt-resume` mode documentation must describe steering/follow-up anchors, safe-point behavior, expected admission/apply markers, and rollback notes before example code or task checkboxes are changed.

## Open Questions

None. The first-profile input kinds, ownership, safe-point boundary, follow-up promotion rule, boundedness, replay behavior, and rollback strategy are fixed by this design; changes to them require an OpenSpec update before implementation.
