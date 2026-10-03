## 1. Regression Tests

- [x] 1.1 Add an event-collector test for confirm Run and verify exactly one accepted `run.dynamic_action.resolved`, normal completed/failed continuation, and no canceled `run.finished` event.
- [x] 1.2 Add an equivalent event-collector test for confirm Stream and verify Run/Stream resolution payload and cardinality parity.
- [x] 1.3 Add deny, timeout, and duplicate-confirm assertions and verify one resolution plus one canceled terminal event for deny/timeout and no duplicate events for an idempotent retry.

## 2. Runner Event Semantics

- [x] 2.1 Split dynamic-action resolution and canceled-terminal event emitters and verify each helper emits only its own semantic event.
- [x] 2.2 Emit the resolution event for accepted confirm, deny, and timeout after checkpoint mutation and lock release; preserve the duplicate short-circuit and verify targeted runner tests pass.
- [x] 2.3 Emit canceled terminal projection only for deny/timeout and verify normal confirm continuation preserves the original Run/Stream correlation and terminal state.

## 3. Contract and Verification

- [x] 3.1 Run `gofmt` and targeted `go test ./core/runner -run DynamicAction -count=1` with all new event assertions passing.
- [x] 3.2 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml` with no regressions.
- [x] 3.3 Run `openspec validate --all`, `pwsh -File scripts/check-docs-consistency.ps1`, and the quality gate; confirm the modified event contract remains additive and documented.

## Example Impact Assessment

无需示例变更（附理由）

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `design.md` | core/runtime maintainers | docs consistency |
| components | 修改文档 | runner implementation and tests | core/runner owners | focused runner tests |
| configuration | 修改文档 | proposal/task governance declarations only; no runtime keys change | governance maintainers | documentation impact gate |
| contract/API | 修改文档 | delta specs for action-gate and host correlation | contract owners | OpenSpec validation |
| diagnostics | 修改文档 | resolution and canceled event payload assertions | observability owners | event collector tests |
| examples | 无需文档变更（附理由） | no example source or invocation changes | example owners | docs consistency |
| CLI/integration | 修改文档 | host event projection contract | host owners | host contract tests |
| best practices | 修改文档 | design migration guidance | maintainers | docs consistency |
| roadmap | 修改文档 | README and development roadmap active status | release owner | status parity |
