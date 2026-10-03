## Why

`ResumeDynamicAction` currently emits `run.dynamic_action.resolved` only on deny/timeout, and the helper that emits it also emits a canceled `run.finished` event. This conflates decision admission with terminal cancellation and leaves confirm resumes without a canonical resolution event, which makes SSE consumers and `RuntimeRecorder` unable to distinguish an accepted confirmation from a canceled action. The fix is needed now because the archived dynamic-action contract promises explicit resolution, terminal projection, and Run/Stream parity.

## What Changes

- Split dynamic-action event emission into a resolution event and a canceled-terminal event.
- Emit exactly one `run.dynamic_action.resolved` event for accepted confirm, deny, and timeout decisions.
- Emit `run.finished(state=canceled)` only for deny and timeout decisions.
- Preserve the existing duplicate-equivalent decision short-circuit before event emission so retries with the same idempotency key do not duplicate resolution events.
- Add event-collector regression coverage for confirm Run/Stream, deny, timeout, and duplicate confirm behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `action-gate-hitl`: clarify that dynamic action resolution is emitted for every accepted decision, while only deny/timeout produce a canceled terminal event; preserve Run/Stream equivalence.
- `embedded-host-command-response-and-event-correlation-contract`: clarify that an accepted action-resume decision produces one source-owned resolution event and that terminal cancellation is emitted only for deny/timeout.

## Impact

- Affected implementation: `core/runner/dynamic_action_resume.go`.
- Affected tests: `core/runner/dynamic_action_resume_test.go` and event-capture helpers in the runner package.
- Affected observability consumers: `run.dynamic_action.resolved` and `run.finished` event semantics; no field removal or diagnostic schema break is intended.
- No provider, host transport, business PendingAction payload, or executor ownership changes.
- Example Impact Assessment: 无需示例变更（附理由）：本 change only corrects event classification and cardinality for an existing dynamic-action contract; existing examples do not assert the conflated event sequence.

## Example Impact Assessment

无需示例变更（附理由）

## Documentation Impact Assessment

- architecture: 无需文档变更；source ownership and module boundaries remain unchanged。
- components: 修改文档；the archived dynamic-action contract and host event-correlation contract receive requirement clarifications。
- configuration: 无需文档变更；no configuration keys or defaults change。
- contract/API: 修改文档；accepted resolution and canceled terminal event cardinality are clarified in the modified specs。
- diagnostics: 修改文档；RuntimeRecorder consumers now receive the intended resolution/terminal distinction without changing additive fields。
- examples: 无需示例变更（附理由）；the behavior is an event-classification correction covered by runner tests。
- CLI/integration: 无需文档变更；host command admission remains source-owned and transport-neutral。
- best practices: 无需文档变更；no new integration pattern is introduced。
- roadmap: 修改文档；README 与 development roadmap 需要反映该 active follow-up change，并在归档后更新状态。

Affected paths: `core/runner/dynamic_action_resume.go`, `core/runner/dynamic_action_resume_test.go`, `README.md`, `docs/development-roadmap.md`, `openspec/changes/fix-dynamic-action-resolution-terminal-event-semantics/`, and the two modified capability specs.
Owner: core/runner with observability contract ownership retained by existing RuntimeRecorder/event paths.
Verification: targeted runner tests, `go test ./core/runner`, full Go tests, race tests, lint, OpenSpec validation, docs consistency, and quality gate.

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `openspec/changes/fix-dynamic-action-resolution-terminal-event-semantics/design.md` | core/runtime maintainers | docs consistency |
| components | 修改文档 | `core/runner/dynamic_action_resume.go` | core/runner owners | focused runner tests |
| configuration | 修改文档 | `openspec/changes/fix-dynamic-action-resolution-terminal-event-semantics/proposal.md` | governance maintainers | documentation impact gate |
| contract/API | 修改文档 | modified action-gate and host correlation specs | contract owners | OpenSpec validation |
| diagnostics | 修改文档 | resolution/terminal event payload contract | observability owners | event collector tests |
| examples | 修改文档 | proposal Example Impact Assessment; no example source change | example owners | docs consistency |
| CLI/integration | 修改文档 | host action-resume correlation contract | host owners | host contract tests |
| best practices | 修改文档 | design.md migration and consumer guidance | maintainers | docs consistency |
| roadmap | 修改文档 | `README.md`, `docs/development-roadmap.md` | release owner | status parity |
