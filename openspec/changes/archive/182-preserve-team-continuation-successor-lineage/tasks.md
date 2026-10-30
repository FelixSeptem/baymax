## 1. Source-Owned Successor Contract

- [x] 1.1 Add a failing team test that resumes a paused team, pauses at a second safe point, and asserts successor generation/predecessor plus the predecessor's retained successor reference; verify it fails on current generation-1 behavior.
- [x] 1.2 Add failing store tests for fresh FileStore reopen, exact duplicate projection, and foreign/conflicting successor digest, generation, predecessor, graph, and authority; verify each fails for the missing lineage transition rather than test setup.

## 2. Team Owner Implementation

- [x] 2.1 Carry the consumed checkpoint reference and next generation through the resumed team execution into a later pause; verify the successor envelope names the original logical team/run and immediate predecessor.
- [x] 2.2 Extend the existing team resume projection and store commit transition to publish the sealed successor reference atomically with the predecessor result; verify failed persistence leaves no partially published in-memory or file state.
- [x] 2.3 Validate successor namespace, digest, graph/team/run/session/authority binding, generation, and predecessor relation before claim; verify malformed, foreign, stale, or conflicting lineage fails closed before member dispatch.
- [x] 2.4 Verify a fresh engine/store resumes only successor pending members, settled effects do not rerun, duplicate resume returns the same projection, and Run/Stream terminal semantics remain equivalent.

## 3. Public Documentation and Verification

- [x] 3.1 Update `orchestration/teams/README.md` with repeated-pause lineage and recovery boundaries; verify it describes same-host single-writer scope and does not claim distributed recovery.
- [x] 3.2 Update `docs/development-roadmap.md` so the graph-material candidate reflects the team lineage gap and its resolved evidence; verify other candidates and owner restrictions remain listed.
- [x] 3.3 Run `go test ./orchestration/teams ./orchestration/mailbox ./orchestration/planbridge -count=1`, `go test -race ./orchestration/teams ./orchestration/mailbox ./orchestration/planbridge -count=1`, and the PowerShell/shell team-mailbox continuation gates; verify no second mailbox owner is introduced.
- [x] 3.4 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; record any environment-blocked command with its evidence and reason.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | orchestration/teams maintainers | Existing owner boundary remains unchanged |
| components | 修改文档 | `orchestration/teams/README.md` | orchestration/teams maintainers | Team continuation contract gate |
| configuration | 修改文档 | `orchestration/teams/README.md` | orchestration/teams maintainers | README records that no configuration key or default changes |
| contract/API | 修改文档 | `orchestration/teams/README.md` | orchestration/teams maintainers | Successor lineage and duplicate projection tests |
| diagnostics | 修改文档 | `orchestration/teams/README.md` | diagnostics owner | README records that no diagnostic schema changes |
| examples | 修改文档 | `orchestration/teams/README.md` | examples owner | README records that example runtime paths and markers are unchanged |
| CLI/integration | 无需文档变更（附理由） | — | integration owner | No CLI or integration entry point changes |
| best practices | 无需文档变更（附理由） | — | orchestration/teams maintainers | README records recovery boundary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | Roadmap status consistency gate |

## Example Impact Assessment

- 无需示例变更（附理由）

The checkpoint lineage behavior is verified by source-owner tests and replay/gates; no example runtime path or marker changes.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`, `orchestration/teams/README.md`
- 维护 owner：orchestration/teams maintainers
- 验证证据：`pwsh -File scripts/check-team-mailbox-continuation-contract.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`
