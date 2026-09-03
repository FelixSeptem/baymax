## 1. Contract baseline and documentation

## Example Impact Assessment

修改示例

- [x] 1.1 Audit `tool/local`, `core/runner`, middleware, policy, sandbox, terminal outcome, diagnostics, and replay owners; record source-of-truth and non-goals in the change notes
- [x] 1.2 Freeze lifecycle stage vocabulary (`prepare`, `validate`, `authorize`, `execute`, `finalize`), skipped-stage semantics, failure-origin mapping, attempt scope, and bounded field limits
- [x] 1.3 Update `docs/development-roadmap.md` with the P1 tool lifecycle and failure-isolation contract, dependencies, startup signals, and rollback boundary
- [x] 1.4 Update `README.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and `docs/mainline-contract-test-index.md` with the capability, owner boundaries, and gate mapping
- [x] 1.5 Update the selected tool-loop/agent-mode README and `examples/agent-modes/MATRIX.md` with semantic anchor, runtime path, expected markers, deny/panic/timeout/finalize assertions, parallel ordering, and rollback notes before code changes

## 2. Lifecycle projection model

- [x] 2.1 Add transport-neutral lifecycle stage and failure-isolation projection types with additive nullable/defaultable JSON fields
- [x] 2.2 Implement deterministic stage transition normalization and explicit `skipped`/`not_applicable` handling without introducing a second execution state machine
- [x] 2.3 Preserve `call_id`, tool name, source identity, run/step correlation, original input position, attempt count, and started/executed markers
- [x] 2.4 Reuse existing `ClassifiedError`, security/reason taxonomy, ReAct termination mapping, and terminal arbiter; add only bounded lifecycle origin/subreason fields
- [x] 2.5 Add unit tests for successful stage order, unknown tool, schema failure, skipped authorization, blank/duplicate call IDs, and legacy field defaults

## 3. Dispatcher failure isolation and finalization

- [x] 3.1 Instrument existing dispatcher boundaries to project prepare and validate outcomes without changing lookup or schema behavior
- [x] 3.2 Project policy, adapter allowlist, sandbox capability, and egress decisions as authorization outcomes that cannot be overridden by middleware
- [x] 3.3 Project middleware short-circuit, middleware error, sandbox failure, local execution error, timeout, cancellation, and panic recovery with existing classifications
- [x] 3.4 Preserve retry ownership and record one attempt scope with deterministic retry count and retry-exhaustion mapping
- [x] 3.5 Guarantee idempotent finalize projection for every call entering dispatch processing, including abnormal exits and nil/partial results
- [x] 3.6 Preserve source-owned partial content, structured facts, completed sub-operations, resource-release status, and terminal outcome without compensation or exactly-once claims
- [x] 3.7 Add focused dispatcher tests for panic isolation, pre-start versus started timeout/cancel, retry exhaustion, middleware short-circuit, sandbox deny, and finalize-on-error
- [x] 3.8 Add parallel-dispatch tests proving completion timing does not reorder normalized outcomes or ReAct feedback and that duplicate IDs classify deterministically

## 4. Diagnostics and observability

- [x] 4.1 Add additive lifecycle fields to `RuntimeRecorder` payload normalization and `runtime/diagnostics` records with nullable/default compatibility
- [x] 4.2 Route all lifecycle writes through `observability/event.RuntimeRecorder` and keep raw arguments, stack traces, and causation values out of OTel metric dimensions
- [x] 4.3 Add diagnostics tests for stage order, failure origin, attempt metadata, idempotent finalize, conflict markers, retained facts, and historical parser compatibility
- [x] 4.4 Add bounded-cardinality and serialization tests for lifecycle fields and verify no global lifecycle queue or hosted owner is introduced

## 5. Replay, integration, and Run/Stream parity

- [x] 5.1 Add canonical `tool_lifecycle_failure_isolation.v1` replay fixture covering success, validation rejection, policy denial, sandbox failure, middleware short-circuit, panic, timeout, cancellation, retry exhaustion, partial facts, and parallel ordering
- [x] 5.2 Add malformed, unsupported-version, stage-order drift, failure-origin drift, duplicate-finalize, duplicate-call-id, and hosted-ownership negative fixtures with stable classifications
- [x] 5.3 Extend `tool/diagnosticsreplay` normalization and tests for lifecycle stage, attempt, finalize idempotency, retained facts, and input-order drift
- [x] 5.4 Add Run/Stream integration tests for successful tool loops, deny/timeout/panic paths, retry exhaustion, cancellation, provider failure after completed tool facts, and equivalent terminal mapping
- [x] 5.5 Add integration tests proving policy/sandbox ownership cannot be bypassed and lifecycle projection does not mutate Run state or create a second terminal outcome
- [x] 5.6 Add replay idempotency and mixed-version compatibility tests for old diagnostics records without lifecycle fields
- [x] 5.7 Add executable agent-mode smoke assertions for denial, panic/timeout finalization, partial facts, and parallel `call_id` ordering

## 6. Gates and quality verification

- [x] 6.1 Add shell and PowerShell `check-tool-lifecycle-failure-isolation-contract` gates with identical deterministic classifications and fail-fast behavior
- [x] 6.2 Add dependency scans rejecting transport listeners, hosted tool/session stores, global invocation queues, external event stores, and alternate terminal state machines
- [x] 6.3 Wire the new gate into `scripts/check-quality-gate.sh/.ps1` and update required-check and mainline contract documentation
- [x] 6.4 Run `openspec validate harden-tool-lifecycle-and-failure-isolation-contract` and resolve all artifact/schema drift
- [x] 6.5 Run focused package tests, integration/replay suites, and agent-mode smoke with `GOSUMDB=sum.golang.org`
- [x] 6.6 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`
- [x] 6.7 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`, plus shell parity gates
- [x] 6.8 Review final diff for additive compatibility, module-boundary compliance, example evidence, rollback notes, absence of proposal-number identifiers, and no unrelated file changes
