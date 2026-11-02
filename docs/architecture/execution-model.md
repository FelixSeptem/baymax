# Execution Model

This page gives external integrators one mental model for Baymax. It explains
how the nouns used by the runtime relate to one another and which owner is
allowed to make each decision. It is intentionally a semantic guide, not a
replacement for the typed contracts in `core/types`, package tests, or
`openspec/specs`.

## The mental model

```text
host request
    |
    v
admission and capability check
    |
    v
logical Run / graph
    |
    +--> Task or workflow step
    |       |
    |       +--> Attempt (model/tool/delivery execution)
    |                  |
    |                  +--> source event facts
    |                  +--> checkpoint at an owner safe point
    |
    +--> source terminal outcome
                |
                v
       host authorization and business effect commit
```

The important boundary is at the bottom: a Baymax source can finish its own
execution, but only the host can authorize and commit a business effect.
Observing an event, acknowledging a mailbox message, or receiving a terminal
`RunResult` does not cross that boundary.

## Core nouns and owners

| Noun | Meaning | Primary owner | What it does not prove |
| --- | --- | --- | --- |
| Run | One logical execution identified by a source/run identity. Run and Stream use the same decision and terminal semantics. | runner or an orchestration source | It is not a business transaction or a durable restore handle by itself. |
| Session | An optional caller correlation scope that can relate runs and host references. | host plus the relevant source | It does not grant authority, define global event order, or make a run recoverable. |
| Task | A unit owned by scheduler, child, team, or workflow orchestration. | the orchestration owner | It is not interchangeable with a runner Run or a mailbox message. |
| Attempt | One execution, delivery, lease, or retry try for a task or message. | the source owner | A retry is not an invisible continuation; it has its own attempt/causation evidence. |
| Event | A source fact or a bounded projection of a source fact. | source owner; observer only projects | A callback EventID or host sequence is not automatically a durable cursor. |
| Checkpoint | Source-owned continuation material at a safe pause boundary. | original execution owner | A metadata snapshot, replay page, or reused RunID is not restore material. |
| Terminal outcome | The authoritative source result such as succeeded, failed, canceled, expired, or dead-lettered. | source owner | It does not imply that a host business effect was committed. |
| Business effect | An external write or domain decision that changes host-owned state. | host authorization/effect/commit owner | It is not owned by Baymax diagnostics, replay, ack, digest, or terminal projection. |

There is no universal cross-source sequence. A runner, child, team, workflow,
and mailbox may each expose a source-local order, but parallel children and
team members must not be presented as one naturally ordered stream.

## One request through the system

1. **The host chooses the operation.** It supplies the request, authority,
   provider/tool selection, policy, write scope, and any host references. Baymax
   does not discover a provider, infer authorization from a DTO, or create a
   hidden global registry.
2. **Admission validates capability and identity.** Unsupported, unavailable,
   stale, foreign, malformed, or expired inputs fail before execution side
   effects. A plan digest or capability manifest proves descriptor consistency;
   it is not an authorization grant.
3. **The source owner executes.** The runner performs model/tool steps. An
   orchestration owner performs graph, task, lease, retry, or mailbox work.
   Each attempt and transition keeps its source identity and causation chain.
4. **Observers project facts.** `RuntimeRecorder` is the diagnostics write
   entry. The native event adapter is an opt-in callback projection. Source
   history readers are bounded and read-only; replay cannot call a model,
   tool, retry, resume, ack, nack, or business commit.
5. **The source publishes terminal evidence.** Run/Stream, mailbox, workflow,
   scheduler, and team terminal states remain source-specific. Host code must
   evaluate the source result rather than reconstructing it from observer
   status or a partial event page.
6. **The host decides effects.** The host rechecks current authorization,
   revision and idempotency/effect key, then commits or rejects the business
   effect. A source terminal can be retained while its business commit is
   pending or rejected.

## Pause, retry, replay, and restore

These operations answer different questions:

| Operation | Question answered | Side effects | Identity rule |
| --- | --- | --- | --- |
| Pause | Where did the source stop safely? | None after the safe point | Preserve the source run/graph and pending action. |
| Retry | Which attempt should be tried again? | Starts a new source attempt | Record retry causation; do not hide it as the original attempt. |
| History replay | What retained facts existed after this cursor? | Read-only | Use only the source's opaque cursor and retention rules. |
| Continuation restore | How does the original graph continue? | Source execution resumes | Claim the original checkpoint atomically; do not start a new parent. |
| Business retry | Should the host effect be attempted again? | Host-owned effect | Use host idempotency/CAS rules; do not reuse source ack as proof. |

`input_required`, `approval_required`, and `budget_exhausted` are capability
cells, not universal behavior. A form is supported only when its source owner
can persist the real pending state, consumed budget/deadline, completed
outcomes, terminal arbitration and restore evidence. Otherwise admission must
return a typed `unsupported_capability` or unavailable result.

## How to use this model

When integrating a new capability, answer these questions in order:

1. Which source owns the fact and terminal outcome?
2. Is the requested operation execution, observation, history replay,
   continuation restore, or host business commit?
3. Which identity must remain stable, and which attempt/causation identity must
   advance?
4. What is the retention, cursor, checkpoint, lease, or expiry boundary?
5. Which capability manifest cell and contract/replay gate prove support?
6. What does the host do when the source is unavailable or unsupported?

Start with the [architecture overview](overview.md), then read the
[runtime lifecycle](runtime-lifecycle.md), [data and event flow](data-event-flow.md),
and [integration capability matrix](../guides/integration.md). For field-level
rules, follow the source links in the [documentation ownership map](../documentation-ownership.md).

## Non-goals

This page does not define a global workflow engine, a remote control plane, an
exactly-once guarantee, or a new persistence layer. It does not make a
terminal-only form recoverable, turn callback events into durable history, or
move authorization and business effects into Baymax.
