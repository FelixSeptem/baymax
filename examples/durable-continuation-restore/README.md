# Durable continuation restore

Status: offline direct-run example using the delivered `native_continuation.v1` API. No credentials or live provider are required.

Run from the repository root: `go run ./examples/durable-continuation-restore`.

## Semantic anchor

One direct native Run is paused, its source-owned checkpoint and bounded continuation material are durably saved, and a newly constructed Engine restores that same logical run only after binding validation and an atomic resume claim. No new parent Run is created and completed tools are not invoked again.

## Runtime path

The example uses a temporary local FileStore, synthetic model/tool dependencies, direct pause, store close/reopen with a new Engine, and source restore. It exercises `input_required`, `approval_required`, and `budget_exhausted` in Run and Stream, exact duplicate projection, conflict/foreign rejection, store unavailability, and explicit composite capability rejection. Workflow-first graph continuation is a separate source-owner profile: a workflow Engine with its own FileStore exposes its graph checkpoint/restore and pause cells through `CapabilityManifestWithWorkflowContinuation`; it preserves graph lineage, completed steps/effects and Run/Stream parity. Child/team/mailbox/planbridge resumed remain unsupported. Cross-process, corrupt/expired checkpoint, host revalidation denial and crash-window coverage live in `core/runner` and `core/checkpoint` tests. This example closes/reopens source handles in one process; `TestNativeContinuationCrossProcess` uses three independent OS processes per reason/mode.

The original namespace, dependency profile/revision and authority/plan/scope/effect binding are explicitly injected into each Engine. The source budget owner approves only an increased limit with a new revision and unchanged consumed counters/deadline. Production hosts also inject `WithHostResumeVerifier` and enforce current authority and business commit at their own boundaries; synthetic digests in this example are not authorization.

History replay is read-only and not restore. A claim or accepted cancellation request is not terminal evidence. A crash after claim/start without source result commit is `in-progress` or `recovery-required`, never an automatic retry.

## Expected markers

- Original run/session/parent and host correlation remain unchanged after Engine reopen.
- Completed tool outcomes are reused; no completed tool is dispatched again.
- Exact duplicate resume returns source-retained projection; conflict fails closed.
- Every supported pause reason has source evidence and current admission checks.
- Child/fanout, team, workflow, mailbox pause, and planbridge resumed paths reject before activation.
- Replay invokes no model/tool/resolver and mutates no source state.

The executable prints `<reason>_<run|stream>_restored` for all six cases, followed by `duplicate_projection`, `conflict_rejected`, `foreign_rejected`, `store_unavailable`, `composite_unsupported`, and `run_stream_parity`. Each marker is printed only after its corresponding invariant is checked. Read-only replay is exercised by the dedicated lifecycle gate, independently of the example.

FileStore holds a single writer lock and retains bounded private normalized requests/tool outcomes. Treat its directory as execution data. Do not use it as a shared distributed queue or copy its material into host telemetry.

## Rollback notes

Disable durable capability advertisement and stop accepting new durable pauses. Preserve persisted checkpoints and in-flight attempt records for inspection; do not delete, reinterpret, retry, or promote them to fresh Runs. Existing same-Engine dynamic-action behavior remains a separate legacy profile.
## Child/fanout boundary

The durable continuation example remains direct/workflow-first. Child/fanout
restore requires an explicit Composer `ChildContinuationStore`, source owner,
and scheduler task/attempt evidence; planbridge refuses terminal-only or
owner-less child continuation. Team, mailbox and resumed-plan cells remain
unsupported until their source owners provide equivalent evidence.
