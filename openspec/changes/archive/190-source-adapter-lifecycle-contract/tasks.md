## 1. Admission, state and lifecycle

- [x] 1.1 Write failing source-bound admission, sent/gone/retryable and pending-conflict tests; run focused Go tests and record RED before production code.
- [x] 1.2 Implement bounded immutable facts/metadata, state CAS, sequential polling/callback offers, finite health and cooperative shutdown; verify positive/negative/boundary suites and race tests.
- [x] 1.3 Implement same-host single-writer file state with strict bounded envelopes and atomic writes; verify failed persistence leaves prior state and fresh subprocess reopen preserves confirmed/pending evidence.

## 2. Integration and conformance

- [x] 2.1 Add real injected source/consumer integration including host effect CAS crash/retry and callback Observer boundary; verify no source execution owner or history identity substitution.
- [x] 2.2 Add pure bounded source_adapter_lifecycle.v1 replay/fixtures, outcome and cursor drift negatives; verify replay does not instantiate source or consumer.
- [x] 2.3 Deliver offline example and markers plus equivalent PowerShell/shell dedicated gates and quality/contribution wiring; verify positive gates pass and deliberate drift fails.

## 3. Review, verification and archive

- [x] 3.1 Synchronize all declared docs and exact limits/failure/scope/rollback semantics; verify documentation impact and docs consistency gates.
- [x] 3.2 Complete implementation review and run go test ./..., go test -race ./..., golangci-lint run --config .golangci.yml, quality/docs gates and openspec validate --all; record exact evidence in verification.md.
- [x] 3.3 Prepare archive-ready artifacts and evidence and assess main-spec synchronization. Archive via scripts/openspec-archive-seq.ps1; subsequent commit, merge, push, remote verification and merged worktree cleanup are delivery steps, not pre-archive implementation assertions.

## Example Impact Assessment

新增示例

新增离线 `examples/source-adapter-lifecycle`，展示 sent/gone/retryable、
cursor-after-commit、abort 和 restart；不连接真实外部 source。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/source-adapter-lifecycle.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/source-adapter-lifecycle/README.md`
- 维护 owner：`host/sourceadapter` maintainers
- 验证证据：`pwsh -File scripts/check-source-adapter-lifecycle-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary tests and dedicated gate |
| components | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | source-adapter lifecycle suite |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | host integration maintainers | constructor bounds and timeout; no env/file/hot-update keys |
| contract/API | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | bounds and restart tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | host integration maintainers | finite API health; no RuntimeRecorder write or QueryRuns field |
| examples | 新增文档 | `examples/source-adapter-lifecycle/README.md` | examples maintainers | offline example marker suite |
| CLI/integration | 修改文档 | `README.md` | documentation maintainers | library-first integration entry |
| best practices | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | cursor-after-commit and CAS notes |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status consistency |
