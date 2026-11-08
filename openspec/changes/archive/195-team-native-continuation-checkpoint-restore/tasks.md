## 1. Team continuation identity and aggregate provenance

- [x] 1.1 Extend canonical team plan identity and sealed continuation metadata with normalized aggregation policy, participant boundary, and aggregate provenance digest; reject malformed, stale, foreign, or oversized material.
- [x] 1.2 Verify parallel and vote pause checkpoints preserve settled member status, effect references, votes, pending IDs, tie-break, participant provenance, and terminal aggregate inputs without payloads/callbacks.
- [x] 1.3 Verify resume compares the normalized current plan against the sealed team identity before claim; policy drift, duplicate/conflict, and ambiguous in-flight cases fail closed without member activation.

## 2. Fresh-process restore and transport parity

- [x] 2.1 Add fresh-process parallel and vote restore tests proving only pending members execute and settled effects/votes are not repeated.
- [x] 2.2 Add repeated-pause successor tests proving policy/provenance and original team/run/session/causation identity survive generation changes.
- [x] 2.3 Add Run/Stream parity tests for vote winner, participant boundary, tie-break, cancellation, timeout, retry, and terminal arbitration.

## 3. Capability and contract evidence


- [x] 3.1 Update team capability evidence references and validation so explicit File owner/deployment plus policy-binding, duplicate-claim, ambiguous-in-flight, and parity evidence are required; memory/unconfigured owners remain unsupported.
- [x] 3.2 Add or update the dedicated team continuation shell/PowerShell gate and fixtures for policy drift, duplicate claim, crash window, repeated pause, and no-side-effect rejection.
- [x] 3.3 Update `README.md`, roadmap, integration, runtime diagnostics/configuration, contract index, team README, and `examples/host-execution-plan/README.md` with the supported profile and explicit non-goals.

## 4. Verification and delivery

- [x] 4.1 Run focused team/planbridge tests and `git diff --check`.
- [x] 4.2 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `openspec validate --all`, roadmap/documentation/example-impact validators, and all applicable contract gates.
- [x] 4.3 Record verification evidence, archive with `scripts/openspec-archive-seq.ps1`, explicitly stage only this change's files, commit, merge into latest `master`, push, verify merged state, and delete the merged branch/worktree.

Verification evidence for 4.2 (2026-11-08): `go test ./...` and `go test -race ./...` passed; `golangci-lint run --config .golangci.yml` reported 0 issues; `pwsh -File scripts/check-quality-gate.ps1` passed all 94 steps (including team continuation, replay, documentation, semantic stability, performance, and vulnerability checks); `openspec validate --all` passed 154/154; roadmap status, example-impact, and documentation-impact validators passed; and `pwsh -File scripts/check-team-continuation-contract.ps1` passed.

## Example Impact Assessment

修改示例

理由：示例支持矩阵需要记录 team policy digest/provenance 与显式 File owner 的恢复条件；运行入口和 expected markers otherwise remain unchanged.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/teams/README.md`、`examples/host-execution-plan/README.md`
- 维护 owner：Runtime contracts owner、team owner maintainer、examples/documentation maintainer
- 验证证据：`go test ./orchestration/teams ./orchestration/planbridge ./core/types`、team continuation contract gate、`openspec validate --all`、`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | owner-bound restore tests |
| components | 修改文档 | `orchestration/teams/README.md` | team owner maintainer | team continuation tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | capability deployment checks |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | strict validators |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | gate evidence |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples/documentation maintainer | team policy-binding and owner-bound rejection matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | pre-activation rejection |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency |
