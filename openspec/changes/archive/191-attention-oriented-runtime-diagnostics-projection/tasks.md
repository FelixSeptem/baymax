## 1. Projection contract

- [x] 1.1 Add runtime/diagnostics/attention_test.go for mixed/healthy/retrying, all finite mappings, atomic invalid input, bounds, dedup/order/truncation and detached result; run go test ./runtime/diagnostics -run TestAttention -count=1 and capture RED.
- [x] 1.2 Implement runtime/diagnostics/attention.go with pure validated snapshot projection; verify TestAttention passes and race checks.
- [x] 1.3 Add integration/attention_diagnostics_test.go for real RuntimeRecorder -> QueryRuns -> projection store immutability and explicit host effect independence; verify focused integration tests.

## 2. Replay and example

- [x] 2.1 Write failing tool/diagnosticsreplay/attention_test.go then implement strict 1MiB bounded attention.v1 replay with golden/invalid fixtures, owner/action/evidence drift; verify positive and negative classification.
- [x] 2.2 Add examples/attention-diagnostics and README with synthetic offline markers; add shell/PowerShell dedicated gates and contribution ownership/wiring checks; verify both dedicated gates and negative drift.

## 3. Review and delivery preparation

- [x] 3.1 Check all declared docs against exact semantics, run documentation impact and docs consistency gates; retain actual command evidence.
- [x] 3.2 Complete independent review, full go test ./..., go test -race ./..., golangci-lint, quality/docs gates and openspec validate --all; record evidence in verification.md.
- [x] 3.3 Prepare archive-ready artifacts and sync main spec through archive script; commit/merge/push/remote proof/clean are subsequent delivery steps.

## Example Impact Assessment

新增示例

新增离线 `examples/attention-diagnostics`，用合成引用展示 mixed page、
healthy/retrying 无行动项、显式 effect/source evidence 和截断，不调用外部系统。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/attention-diagnostics.md`、`runtime/diagnostics/README.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`
- 维护 owner：runtime/diagnostics maintainers
- 验证证据：`pwsh -File scripts/check-attention-diagnostics-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | diagnostics maintainers | pure read model dependency boundary |
| components | 修改文档 | `runtime/diagnostics/README.md` | diagnostics maintainers | projection unit tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics maintainers | API defaults and limits; no new config |
| contract/API | 修改文档 | `docs/guides/attention-diagnostics.md` | diagnostics maintainers | finite mappings and negative tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics maintainers | no writer/query schema change |
| examples | 修改文档 | `docs/guides/attention-diagnostics.md` | examples maintainers | synthetic offline example |
| CLI/integration | 修改文档 | `README.md` | diagnostics maintainers | library integration entry |
| best practices | 修改文档 | `docs/guides/attention-diagnostics.md` | diagnostics maintainers | no terminal-to-effect inference |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap consistency gate |
