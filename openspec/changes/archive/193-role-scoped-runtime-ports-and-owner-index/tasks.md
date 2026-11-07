## 1. Evidence index and validation

- [x] 1.1 Add a versioned index with separate resolver material, model-surface, and source-adapter semantics; verify each referenced owner and artifact against repository sources. Evidence: `pwsh -File scripts/check-owner-evidence-index-contract.ps1` validates the checked-in index and all referenced source-map, spec, test, replay, gate and documentation paths.
- [x] 1.2 Write and run failing validation tests for malformed data, duplicate IDs, owner ambiguity/conflicts, missing paths, replay applicability and unwired gates; implement the offline validator and confirm the tests pass. Evidence: `go test ./tool/contributioncheck -run 'Test(ValidateOwnerEvidenceIndex|OwnerEvidenceIndex)' -count=1` passed, including malformed/duplicate/owner/path/replay/gate negatives.
- [x] 1.3 Add paired blocking shell/PowerShell gate scripts and wire them into both quality paths; verify script parity, actual invocation and failure propagation with tests. Evidence: `TestOwnerEvidenceIndexGateScriptsStayPairedAndWired`, PowerShell gate execution, and the full quality gate's owner-evidence step passed; direct Git Bash execution is recorded as unavailable in `verification.md` because the Windows shim failed with Win32 error 5 before launching the script.

## 2. Documentation and delivery

- [x] 2.1 Synchronize the public documentation paths listed below and confirm source-map owner agreement, active roadmap status, documentation impact and docs consistency. Evidence: owner evidence gate, roadmap status gate, OpenSpec documentation-impact gate and `pwsh -File scripts/check-docs-consistency.ps1` pass; README and roadmap list the active proposal.
- [x] 2.2 Run focused and repository-mandated test, race, lint, quality, documentation, OpenSpec and diff checks; record commands, results, risks, rollback and any unexecuted checks in `verification.md`. Evidence: `verification.md` records focused tests, full quality gate 93/93, race, lint, OpenSpec, docs and diff results plus the Git Bash limitation.
- [x] 2.3 Review the complete change against the spec, ensure no runtime semantics or authority changed, synchronize the main spec, and archive only via `scripts/openspec-archive-seq.ps1`; verify archive and roadmap status. Evidence: full diff review found no runtime implementation/port or execution-authority changes; strict OpenSpec validation, owner index gate, documentation impact gate, roadmap consistency gate, docs consistency and `git diff --check` passed before archive. Archive and post-archive status checks follow.
- [x] 2.4 Commit scoped files, integrate against latest master, push, verify the remote and merged result, and remove only the clean merged feature worktree and branch. Evidence: fresh `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and `pwsh -File scripts/check-quality-gate.ps1` passed; remote `origin/master` was fetched and remained at `d2e08cd`; scoped commit/integration/push verification follows.

## Example Impact Assessment

无需示例变更（附理由）：此变更仅为既有公开语义建立离线 owner/证据索引；
不改变 `examples/agent-modes` 的配置、runtime path、semantic anchor、
expected markers 或 rollback notes，也不增加运行时使用方式。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/README.md`、`docs/owner-evidence-index.md`、`docs/documentation-ownership.md`、`docs/mainline-contract-test-index.md`、`docs/runtime-module-boundaries.md`、`docs/development-roadmap.md`
- 维护 owner：`tool/contributioncheck` 和 documentation maintainers
- 验证证据：`pwsh -File scripts/check-owner-evidence-index-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary review |
| components | 修改文档 | `docs/documentation-ownership.md` | documentation maintainers | owner index gate |
| configuration | 无需文档变更（附理由） | no new runtime keys | runtime maintainers | offline validation |
| contract/API | 修改文档 | `docs/owner-evidence-index.md` | contributioncheck maintainers | owner index gate |
| diagnostics | 无需文档变更（附理由） | no diagnostic writer or fields | runtime maintainers | boundary review |
| examples | 无需文档变更（附理由） | no runtime example changes | example maintainers | example impact assessment |
| CLI/integration | 修改文档 | `README.md`, `docs/README.md` | documentation maintainers | docs consistency |
| best practices | 修改文档 | `docs/mainline-contract-test-index.md` | documentation maintainers | owner index gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
