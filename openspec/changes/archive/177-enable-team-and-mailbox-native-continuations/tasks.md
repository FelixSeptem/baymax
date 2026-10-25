## 1. Shared Continuation Contract and Owner Boundaries

- [x] 1.1 Add bounded public team/mailbox continuation identity, pause, checkpoint, claim, terminal projection and typed failure DTOs; verify malformed, foreign, expired, stale, corrupt, conflicting and unavailable inputs fail closed without side effects.
- [x] 1.2 Define owner-specific continuation store interfaces and same-host single-writer deployment evidence; verify process-local stores do not advertise cross-process restore and incompatible deployment claims return typed unsupported deployment errors.
- [x] 1.3 Add capability source projections for team and mailbox continuation cells; verify each pause reason and restore dimension is promoted only by matching owner namespace, contract/API version, evidence reference and deployment scope.

## 2. Team Native Continuation

- [x] 2.1 Implement team continuation material for parallel/vote plans, including graph/strategy digest, member state, pending/completed effects, aggregate/vote state, budget/deadline, terminal arbitration and causal lineage; verify bounded detached persistence omits payloads and callbacks.
- [x] 2.2 Add source-owned team pause safe points for `input_required`, `approval_required` and `budget_exhausted`; verify settled members are preserved, pending members are deterministic, consumed budget/deadline are retained, and unsafe in-flight work returns unsupported or recovery-required.
- [x] 2.3 Implement atomic team `Save`/`Load`/`ClaimResume`/`MarkResumeRunning`/`CommitResumeResult` with duplicate convergence and successor lineage; verify fresh-process restore does not re-run completed members or allocate a replacement parent/team run.
- [x] 2.4 Integrate team Run/Stream resume and terminal aggregation; verify parallel/vote lifecycle, tie-break, cancellation, timeout, retry and terminal arbitration remain semantically equivalent across transport surfaces.

## 3. Mailbox Native Continuation

- [x] 3.1 Extend the existing mailbox owner with bounded host-plan continuation material bound to message, delivery, lease, attempt causation, host/run/parent and authority/plan/write-scope/effect identities; verify lifecycle payloads and business commits are excluded.
- [x] 3.2 Add mailbox pause safe points and atomic continuation claim/restore using the current source record; verify valid new-process restore resumes the same logical message without second publish, consume, delivery or queue owner.
- [x] 3.3 Validate lease, heartbeat, retry, ack/nack/requeue, expiration and DLQ races at resume and terminal commit; verify lease loss and unavailable stores return source recovery evidence without synthetic retry or parent Run.
- [x] 3.4 Project mailbox continuation terminal outcomes through the existing owner and preserve message/delivery/lease/attempt causation; verify Run/Stream host-plan projections remain equivalent and distinct from host business commit.

## 4. Planbridge, Manifest and Evidence Gates

- [x] 4.1 Update planbridge admission to validate team/mailbox continuation capability cells before member activation or mailbox lifecycle mutation; verify unsupported cells reject with typed `unsupported_capability`/`unavailable` and no side effects.
- [x] 4.2 Add owner/restart/duplicate/conflict/lease-loss/retry/DLQ/cancellation/timeout fixtures and deterministic replay evidence; verify replay is read-only and never invokes Run, Stream, retry, resume, ack, nack, tool or model.
- [x] 4.3 Add an offline example and update integration, team, mailbox, planbridge, runtime-boundary, diagnostics and contract-index documentation; verify Example Impact Assessment and public semantic documentation commitments point to changed files.

## 5. Verification and Delivery

- [x] 5.1 Run focused team/mailbox/checkpoint/manifest/planbridge tests and race tests; verify all owner and Run/Stream parity suites pass.
- [x] 5.2 Run dedicated PowerShell and shell continuation contract gates; verify both platforms cover unsupported admission, fresh-process restore, duplicate claim, lease loss, retry/DLQ and no-side-effect behavior, recording any environment limitation.
- [x] 5.3 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `openspec validate --all`, and `git diff --check`; record exact outputs and limitations before archive.

## Example Impact Assessment

修改示例

理由：the host execution plan and native execution events examples will document configured team/mailbox continuation, source-owned lease/claim evidence and explicit unsupported fallback without changing business payloads.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `orchestration/teams/README.md`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`, `examples/host-execution-plan/README.md`, `examples/native-execution-events/README.md`
- 维护 owner：Runtime contracts owner、team owner 与 mailbox owner maintainers
- 验证证据：owner unit/integration tests、fresh-process fixtures、Run/Stream parity、dedicated PowerShell/shell contract gates、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、质量门禁与 OpenSpec validation

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contracts owner | owner boundary review |
| components | 修改文档 | `orchestration/teams/README.md`、`orchestration/mailbox/README.md`、`orchestration/planbridge/README.md` | team/mailbox maintainers | focused owner tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | deployment scope review |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | strict validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | gate mapping review |
| examples | 修改文档 | `examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md` | examples maintainer | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | admission rejection review |
| best practices | 修改文档 | `README.md` | documentation maintainer | fail-closed guidance review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency gate |
