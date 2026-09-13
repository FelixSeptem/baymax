## 1. Contract and normalization

- [x] 1.1 Add versioned bounded continuity projection/fact/result types under `runtime/evalcontract`, including fixed semantic kinds, limits, stable identity, and additive nullable experiment linkage; verify new unit tests cover valid, missing, duplicate, unsupported-version, and body-bearing inputs.
- [x] 1.2 Implement deterministic normalization and axis-specific comparison with ordered drift entries for identity, objective, task, attempt/lease, workspace, pending request, checkpoint, artifact, owner, missing/duplicate references, parity, and recovery idempotency; verify unit tests assert stable digests, classifications, and no last-write-wins behavior.
- [x] 1.3 Add handoff and snapshot/checkpoint/workspace adapters that emit only source-owned references and preserve existing restore/source-of-truth semantics; verify adapter contract tests cover strict/compatible restore and Run/Stream equivalence without source mutation.

## 2. Replay and fixture gate

- [x] 2.1 Add `eval_continuity_comparison.v1` fixture schema, parser, and evaluator to `tool/diagnosticsreplay`; verify offline replay succeeds without live runtime/provider/tool/Git access and rejects privacy/schema violations deterministically.
- [x] 2.2 Add version-controlled continuity fixtures for equal projections, every required drift axis, duplicate/missing references, Run/Stream parity, and recovery idempotency; verify mixed historical and continuity fixture suites remain deterministic.
- [x] 2.3 Wire continuity replay into diagnostics contract/gate entry points and document canonical reason codes; verify shell and PowerShell gates fail fast on malformed fixtures and pass on canonical fixtures.

## 3. Integration and documentation

- [x] 3.1 Add additive continuity association coverage to evaluation corpus/experiment comparison and verify existing metric/rubric aggregation and local/distributed parity tests remain unchanged.
- [x] 3.2 Update `docs/mainline-contract-test-index.md` with comparator, adapter, and replay fixture mappings; verify docs consistency checks pass.
- [x] 3.3 Reconcile `docs/development-roadmap.md` to mark the bounded eval continuity comparator as the active proposal and record non-goals, ownership boundaries, and Example Impact Assessment; verify roadmap/OpenSpec consistency gate passes.
- [x] 3.4 Run focused tests, full `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `scripts/check-quality-gate.ps1`, `scripts/check-docs-consistency.ps1`, and `openspec validate --all`; all focused/full tests, race, lint, quality, docs, status, example-impact, semantic-stability, performance, smoke, and vulnerability gates pass with the verified MinGW GCC/CGO toolchain.

## Example Impact Assessment

无需示例变更（附理由）：implementation is limited to offline contract, adapter, fixture, replay, and documentation surfaces; `examples/agent-modes` behavior and runtime paths are unchanged.
