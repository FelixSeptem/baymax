## Implementation Tasks

### Contract and owner implementation

- [x] Add successor continuation persistence and generation/lineage handling to the owner-native resume path; preserve source lifecycle semantics and fail closed on ambiguous claims.
- [x] Add table-driven fresh-process tests for `input_required`, `approval_required`, and `budget_exhausted`, including completed-effect counters and successor pause behavior.
- [x] Add negative/terminal tests for expiry, lease loss, stale deployment, revalidation denial, persistence recovery-required, duplicate resume, requeue, nack, retry exhaustion, and DLQ.
- [x] Verify lifecycle-only `NewWorker`, history replay, and planbridge bindings cannot promote or execute handler continuation; keep Run/Stream decisions equivalent.

### Evidence, documentation, and gates

- [x] Add the dedicated mailbox Agent pause/restore replay fixture and PowerShell/shell gate; enforce owner boundaries and stable unsupported reason.
- [x] Update public README, roadmap item I, integration guide, module boundaries, mailbox/planbridge READMEs, mainline contract index, and host execution plan example.
- [x] Extend `examples/host-execution-plan/main.go` to execute an actual configured handler pause, FileStore reopen, registered resumer, duplicate convergence, and no-handler-rerun assertion; run `go run ./examples/host-execution-plan` and verify the `mailbox-handler-pause-restore-evidence` marker.
- [x] Add `Public Semantic Documentation Commitment` and `Example Impact Assessment` consistency evidence to all change artifacts and run OpenSpec/documentation governance checks.
- [x] Run affected tests, race tests, lint, full quality/docs gates, `openspec validate --all`, and `git diff --check`; record results before archive.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`, `examples/host-execution-plan/README.md`
- 维护 owner：orchestration/runtime maintainers
- 验证证据：`pwsh -File scripts/check-mailbox-agent-pause-restore-contract.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `go test ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -count=1`

## Documentation Impact Assessment

| 文档域 | 变更 | 路径 | Owner | 证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | orchestration/runtime maintainers | owner and deployment boundary |
| components | 修改文档 | `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md` | orchestration maintainers | package contract and delegation boundary |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | states that no runtime keys/defaults changed |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | orchestration/runtime maintainers | public matrix and test index |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | states that no diagnostics or QueryRuns fields changed |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples maintainers | offline marker and support matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration maintainers | host integration boundary |
| best practices | 修改文档 | `README.md` | runtime maintainers | capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active status and scope |
