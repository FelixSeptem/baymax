## 1. Current-pin evidence records

- [x] 1.1 Add the current-pin upstream finding record with legacy pin provenance, evidence class, deployment assumptions, and explicit unresolved child/team/source-history/mailbox/cancellation/retry boundaries.
- [x] 1.2 Add the Baymax contract evidence matrix separating execution wiring, identity/aggregation, continuation restore, source replay/projection, and provider live-smoke status.

## 2. Governance and documentation

- [x] 2.1 Add equivalent PowerShell and shell gates for current/legacy pin consistency, required matrix rows, limitation markers, and `skipped-unconfigured` live-smoke classification; wire them into quality/docs governance.
- [x] 2.2 Update README, roadmap, and contract index navigation to the current-pin evidence records and state that this refresh does not promote unsupported capabilities.

## 3. Verification and delivery

- [x] 3.1 Run the dedicated evidence gate, docs consistency, OpenSpec validators, roadmap/example/documentation impact checks, and `git diff --check`.

Verification evidence: PowerShell and Git Bash current-pin gates passed; `pwsh -File scripts/check-docs-consistency.ps1`, roadmap status consistency, example impact, documentation impact and `git diff --check` passed; `openspec validate --all` reported 154 passed/0 failed; the general quality gate passed repo hygiene, docs/current-pin/example gates, `go test ./...`, `go test -race ./...`, and `golangci-lint`.
- [x] 3.2 Archive with `scripts/openspec-archive-seq.ps1`, explicitly stage only this change and its synchronized documentation/gate files, commit, merge into latest `master`, push, verify merged state, and delete the merged branch/worktree.

Verification evidence: archive script assigned sequence 197 and updated `openspec/changes/archive/INDEX.md`; the change was committed, fast-forward merged into latest `master`, pushed to `origin/master`, and the merged feature branch was deleted. The root worktree was retained. No runtime test suite was required beyond the repository governance gates because this change has `skip_specs: true` and no code or behavior changes.

## Example Impact Assessment

无需示例变更（附理由）

理由：本 change 仅刷新 upstream evidence 与治理文档，不改变任何 example 的执行路径、输出 marker 或配置语义。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/baymax-contract-evidence.md`、`docs/upstream/baymax-findings.md`
- 维护 owner：upstream findings owner、runtime contracts owner、documentation maintainer
- 验证证据：`pwsh -File scripts/check-upstream-findings-current-pin-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/baymax-contract-evidence.md` | runtime contracts owner | current-pin matrix gate |
| components | 修改文档 | `docs/upstream/baymax-findings.md` | upstream findings owner | pin/provenance checks |
| configuration | 修改文档 | `docs/upstream/baymax-findings.md` | upstream findings owner | live-smoke skipped-unconfigured marker |
| contract/API | 修改文档 | `docs/baymax-contract-evidence.md` | runtime contracts owner | capability limitation matrix |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | dedicated evidence gate |
| examples | 无需文档变更（附理由） | examples unchanged | examples/documentation maintainer | no runtime or example semantic change; evidence-only update |
| CLI/integration | 修改文档 | `README.md` | documentation maintainer | navigation and status checks |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap status gate |
