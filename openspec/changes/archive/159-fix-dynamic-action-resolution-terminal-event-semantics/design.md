## Context

The archived dynamic-action implementation stores the decision before releasing `dynamicActionMu`, but its event helper combines the resolution fact and canceled terminal projection. The confirm branch does not call that helper, while deny/timeout receive both events. The fix must preserve checkpoint idempotency and the existing source-owned event path.

## Goals / Non-Goals

**Goals:**

- Make `run.dynamic_action.resolved` a decision-admission fact for confirm, deny, and timeout.
- Keep canceled `run.finished` terminal projection exclusive to deny/timeout.
- Preserve duplicate-equivalent short-circuiting before any event emission.
- Exercise Run/Stream parity and event cardinality through deterministic event collectors.

**Non-Goals:**

- No new checkpoint fields, transport envelopes, business PendingAction storage, or executor behavior.
- No changes to provider adapters, host authorization, or retry/follow-up semantics.

## Decisions

1. **Split event helpers by semantic fact.** Introduce separate internal emitters for the resolution event and canceled terminal event. This prevents a future confirm path from accidentally inheriting canceled semantics and keeps each helper single-purpose.

2. **Emit resolution after checkpoint mutation and lock release.** `ResumeDynamicAction` will set `cp.resumed` and `cp.decision`, persist the checkpoint, release `dynamicActionMu`, and then emit the resolution event. The deny/timeout branch emits the canceled terminal event after resolution; confirm proceeds into the normal Run/Stream continuation.

3. **Keep idempotency before emission.** The existing equivalent duplicate check remains the first checkpoint outcome check. It returns the stored result without calling either emitter, so SSE and RuntimeRecorder observe one resolution per accepted idempotency identity.

4. **Test observable events, not helper calls.** Tests will provide an event handler, count event types, inspect resolution payload fields, and assert terminal state/cardinality for confirm Run, confirm Stream, deny, timeout, and duplicate confirm.

## Risks / Trade-offs

- [Risk] Existing consumers may have inferred cancellation from every resolution event. → The corrected contract makes cancellation explicit in `run.finished`; the modified specs and targeted tests document the distinction.
- [Risk] A continuation may emit its own normal `run.finished` after confirm. → Tests assert the absence of a canceled terminal event while allowing the normal completed/failed terminal outcome.
- [Risk] Event handlers can be nil in existing callers. → Emitters retain the current nil-handler no-op behavior.

## Migration Plan

Deploy as a backward-compatible event semantic correction. Consumers should key cancellation on `run.finished` state rather than treating `run.dynamic_action.resolved` as terminal. Rollback is a single commit revert if downstream consumers require the previous conflated behavior.

## Example Impact Assessment

无需示例变更（附理由）

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `design.md` ownership and event-boundary decisions | core/runtime maintainers | docs consistency |
| components | 修改文档 | `core/runner/dynamic_action_resume.go` | core/runner owners | focused runner tests |
| configuration | 修改文档 | proposal/task governance declarations only; no runtime keys change | governance maintainers | documentation impact gate |
| contract/API | 修改文档 | modified action-gate and host correlation delta specs | contract owners | OpenSpec validation |
| diagnostics | 修改文档 | resolution versus canceled event semantics | observability owners | event collector tests |
| examples | 无需文档变更（附理由） | no example behavior or invocation changes; event semantics are covered by runner tests | example owners | docs consistency |
| CLI/integration | 修改文档 | host action-resume event projection contract | host owners | host contract tests |
| best practices | 修改文档 | migration guidance for consumers | maintainers | docs consistency |
| roadmap | 修改文档 | `README.md`, `docs/development-roadmap.md` | release owner | status parity |
