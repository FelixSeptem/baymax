## 1. Source History Failure Semantics

- [x] 1.1 Define the typed owner-facing append health/error contract and verify positive, optional-observation failure, and unavailable-source unit cases.
- [x] 1.2 Route runner, child, team, and workflow history append outcomes to owner health/result surfaces; verify injected persistence failures are never silently reported as complete history.
- [x] 1.3 Verify callback-history failure is reported as degraded/unavailable without claiming effect gating; verify workflow continuation persistence failures/reopened incomplete claims remain fail-closed and never redispatch uncertain effects.
- [x] 1.4 Add replay tests proving only committed facts are returned and history replay never activates Run, Stream, model, tool, retry, resume, or business commit.

## 2. Workflow Claim Recovery and Store Ownership

- [x] 2.1 Reclassify reopened `claimed` and `running` workflow claims without results as recovery-required; verify exact duplicates return the same typed state and dispatch no steps.
- [x] 2.2 Preserve completed workflow resume projections across reopen; verify duplicate resume returns the committed result without repeating completed effects.
- [x] 2.3 Add process-scoped writer fencing to the workflow FileStore and verify concurrent open is rejected and sequential reopen succeeds on Windows and Unix.
- [x] 2.4 Gate workflow continuation capability advertisement on an active valid fence; verify unfenced/unavailable stores cannot admit resume.
- [x] 2.5 Add deterministic crash-window tests around claim, running marker, effect dispatch, and terminal result persistence; verify no window causes automatic re-execution.

## 3. Public Contract and Verification

- [x] 3.1 Update `README.md`, `core/types/README.md`, `orchestration/workflow/README.md`, and `docs/development-roadmap.md`; verify links and documented unsupported/recovery boundaries.
- [x] 3.2 Update contract-test index and dedicated history/workflow gates; verify fixture classifications for degraded, unavailable, recovery-required, duplicate, and conflicting outcomes.
- [x] 3.3 Remediate the reachable Go standard-library security advisory by updating the strict validation toolchain floor and public prerequisite documents, then run focused package tests, subprocess restart/fence tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; record all results and any environment-blocked command.

### Verification Record

- Passed before the final Go 1.26.10 rerun: focused history/workflow packages; native history and workflow continuation contract gates; `go test ./...`; `go test -race ./...`; `golangci-lint run --config .golangci.yml`; `pwsh -File scripts/check-docs-consistency.ps1`; `openspec validate --all`; `git diff --check`.
- Security remediation: the reachable `GO-2026-6809` scan finding in `crypto/x509@go1.26.9` was removed by changing the validation `toolchain` floor to Go `1.26.10`; `go version` resolves to `go1.26.10 windows/amd64`, and a fresh `govulncheck ./...` reports `Your code is affected by 0 vulnerabilities.`
- Writer-fence evidence: Windows runs cover same-process and cross-process contention plus sequential reopen. Unix runs executed in `docker.1ms.run/library/golang:1.26.10` with the host module cache mounted read-only and `GOPROXY=off`: `go test ./orchestration/workflow -run '^(TestFileWorkflowContinuationStoreReopensSequentially|TestFileWorkflowContinuationStoreRejectsConcurrentOwner|TestFileWorkflowContinuationStoreFencesAcrossProcesses)$' -count=1` passed on Linux.
- Final post-remediation validation: `pwsh -File scripts/check-quality-gate.ps1` completed 86/86 steps successfully in 2094.58s under Go `1.26.10`; its full-chain example smoke passed and strict `govulncheck` reported zero reachable vulnerabilities. Earlier fresh post-remediation runs of `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-docs-consistency.ps1`, `openspec validate --all`, and `git diff --check` all exited 0. No required verification command remains environment-blocked.

## Example Impact Assessment

无需示例变更（附理由）：本 change 只新增失败与恢复边界，不新增正常执行形态或配置；负向和崩溃窗口由确定性测试与 contract fixture 覆盖。

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
|---|---|---|---|---|
| architecture | 修改文档 | `orchestration/workflow/README.md` | Workflow owner | 明确 incomplete claim、recovery-required 与单写者 fence 行为 |
| components | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确显式 history 注入、append failure 与 capability health 边界 |
| configuration | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确本 change 不新增配置键或默认值 |
| validation toolchain | 修改文档 | `CONTRIBUTING.md`, `docs/versioning-and-compatibility.md`, `docs/runtime-config-diagnostics.md` | Runtime contracts owner | 将 strict security validation toolchain floor 同步为 Go 1.26.10 |
| contract/API | 修改文档 | `core/types/README.md` | Runtime contracts owner | 记录 typed history failure 和 workflow recovery 语义 |
| diagnostics | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确不增加诊断 schema 或 recorder 字段 |
| examples | 修改文档 | `core/types/README.md` | Runtime contracts owner | 说明负向/崩溃窗口由测试覆盖，不增加示例 |
| CLI/integration | 无需文档变更（附理由） | — | Runtime contracts owner | 不改变 CLI 或 integration 调用流程 |
| best practices | 无需文档变更（附理由） | — | Runtime contracts owner | 由 owner package 文档说明 failure/recovery 约束，不引入新惯例 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | Runtime roadmap owner | 标记本 change 为 active 并保留剩余 source-owner gaps |
| release status snapshot | 修改文档 | `README.md` | Runtime contracts owner | 将 active OpenSpec 状态与 `openspec list --json` 对齐 |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `core/types/README.md`, `orchestration/workflow/README.md`, `CONTRIBUTING.md`, `docs/versioning-and-compatibility.md`, `docs/runtime-config-diagnostics.md`
- 维护 owner：Runtime contracts owner and Workflow owner
- 验证证据：`openspec validate --all`; `pwsh -File scripts/check-docs-consistency.ps1`; targeted history/workflow tests
