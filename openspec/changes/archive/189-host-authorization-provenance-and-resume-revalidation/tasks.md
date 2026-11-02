## 1. Provenance validation

- [x] 1.1 Add JSON-based regression in `core/types/host_resume_admission_test.go`; run `go test ./core/types -run TestHostResumeProvenance -count=1` and observe stale policy/source accepted before the fix.
- [x] 1.2 Add optional provenance and bounded validation in `core/types/host_resume_admission.go`; verify matching, partial, invalid UTF-8, exact byte limits, drift reasons and authorized cancellation.

## 2. Runtime and replay integration

- [x] 2.1 Add `core/runner/host_resume_provenance_test.go` with Run/Stream rejection-then-retry; verify no resolution, model call or checkpoint consumption on rejection.
- [x] 2.2 Add cross-process reopened FileStore/fresh Engine restore test; verify source Load unchanged and ReadResumeClaim returns not-found after provenance rejection, then valid facts resume once.
- [x] 2.3 Extend bounded offline host-resume replay with optional provenance reason evidence and golden/drift tests; verify malformed or effectful rejection is rejected by the existing replay gate.
- [x] 2.4 Update offline example provenance setup and drift marker; verify `go run ./examples/host-resume-admission` preserves Run/Stream parity.

## 3. Delivery

- [x] 3.1 Synchronize declared public docs, example README, contract index and roadmap; verify documentation-impact and docs-consistency gates.
- [x] 3.2 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, quality gate, docs gate and `openspec validate --all`; record exact evidence.
- [x] 3.3 Complete archive preparation: review implementation/spec alignment, record gate evidence and prepare roadmap archive status. Archive via `scripts/openspec-archive-seq.ps1`; subsequent commit/merge/push/cleanup are delivery operations verified after archive.

## Example Impact Assessment

修改示例

更新 `examples/host-resume-admission`，展示可信暂停记录的 admission/policy/source-scope provenance；不新增执行入口。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/host-resume-admission.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/host-resume-admission/README.md`
- 维护 owner：`core/types`、`core/runner` maintainers
- 验证证据：`pwsh -File scripts/check-host-resume-admission-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | 保留 source/host ownership，未引入模块依赖或新状态 owner |
| components | 修改文档 | `docs/guides/host-resume-admission.md` | core maintainers | host resume contract gate |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | 说明新增字段默认省略，无配置键变化 |
| contract/API | 修改文档 | `docs/guides/host-resume-admission.md` | core maintainers | provenance validation and restore tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | 复用有限 reason，不增加诊断写入口 |
| examples | 修改文档 | `examples/host-resume-admission/README.md` | examples maintainers | offline example and markers |
| CLI/integration | 无需文档变更（附理由） | — | — | 既有 direct/Host guard 复用同一校验，不新增 wire command |
| best practices | 修改文档 | `docs/guides/host-resume-admission.md` | core maintainers | trusted records and effect CAS notes |
| roadmap | 修改文档 | `README.md`、`docs/development-roadmap.md` | documentation maintainers | roadmap status consistency |
