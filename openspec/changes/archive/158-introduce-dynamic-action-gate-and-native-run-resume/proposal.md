## Why

Application adapters can already identify an unconfirmed `PendingAction`, prevent the business executor from running, and continue after confirmation. Baymax cannot yet represent that post-tool fact as a native `input_required` Run outcome or resume the original Runner iteration; the current Action Gate runs before tool dispatch, while existing retry/follow-up paths create a new causally related Run. This blocks evidence for a native same-Run `input_required`/resume contract without blocking adapter-owned application safety boundaries.

## What Changes

- Add a source-owned dynamic action registration contract that lets a tool result or dispatcher return an opaque action token/reference after execution.
- Pause the current Run at the tool-result boundary when a resumable action is registered, expose an additive `input_required` terminal/runtime projection, and prevent subsequent model or executor work until a valid decision is admitted.
- Add a bounded, versioned Runner checkpoint/resume contract that restores the same `run_id`, `session_id`, iteration, tool-result references, pending action reference, and terminal correlation.
- Add idempotent resume admission with fail-closed handling for missing, stale, duplicate, mismatched, or terminal checkpoints.
- Keep PendingAction business payload, authorization, confirmation, and executor ownership in the Application/adapter layer; Baymax stores only bounded opaque references and Runner recovery metadata.
- Keep Realtime interrupt/resume, retry, and follow-up promotion as distinct contracts; do not reinterpret them as dynamic action resume.
- Extend host command/event correlation so action-resume admission is separate from the eventual source-owned runtime and terminal outcome.
- Add Run/Stream parity, integration, replay, diagnostics, and gate coverage for dynamic action pause/resume and negative cases.

## Capabilities

### New Capabilities

- `dynamic-action-gate-and-native-run-resume`: Opaque post-tool action registration, native `input_required` pause, same-Run checkpoint/resume, idempotent admission, and bounded source-owned recovery metadata.

### Modified Capabilities

- `action-gate-hitl`: Extend Action Gate from pre-dispatch confirmation to an explicit post-tool dynamic action pause while preserving fail-closed behavior and Run/Stream semantic equivalence.
- `embedded-host-command-response-and-event-correlation-contract`: Add action-resume command admission and correlated source-owned pause/resume outcome projection without giving the host adapter terminal-state ownership.

## Impact

- Affected code: `core/types` tool/result and runtime protocol contracts, `core/runner` dispatch/state/checkpoint paths, host command correlation, diagnostics/replay and contract gates.
- Affected APIs: additive dynamic action token/reference, checkpoint snapshot/resume interfaces, and host action-resume command envelopes; existing `ModelClient`, retry, follow-up, and Realtime APIs remain source-compatible.
- Affected integrations: Application/REST/MCP adapters can bridge their PendingAction through an opaque token without moving business state into Baymax.
- Persistence/dependencies: no provider SDK or hosted control plane; checkpoint storage is an injected bounded source-owned dependency and must not become a business action store.
- Example Impact Assessment: 新增示例。

## Example Impact Assessment

新增示例

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | core/runtime maintainers | docs consistency |
| components | 新增文档 | `examples/dynamic-action-resume/README.md` | runner/adapter owners | example smoke run |
| configuration | 修改文档 | `openspec/governance/go-file-line-budget-exceptions.csv` | governance maintainers | quality gate |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | contract owners | focused tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability owners | diagnostics tests |
| examples | 新增文档 | `examples/dynamic-action-resume` | example owners | `go run` |
| CLI/integration | 修改文档 | host action-resume contract | host owners | host tests |
| best practices | 修改文档 | runtime boundary docs | maintainer | docs gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap parity |
