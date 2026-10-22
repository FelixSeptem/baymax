## 1. Contract vocabulary and source audit

- [x] 1.1 Inventory scheduler, mailbox, runner/checkpoint, history, composer,
  workflow/team and planbridge stores; record the actual persistence,
  ownership, fencing, cursor, and claim guarantees in the design evidence.
- [x] 1.2 Define versioned bounded deployment fields and typed classifications
  for unsupported scope, unavailable owner, foreign writer, and stale writer.
- [x] 1.3 Add additive source-owned capability DTO fields and validation rules;
  ensure absent fields cannot be interpreted as distributed support.
- [x] 1.4 Add positive/negative contract fixtures for local sequential reopen,
  duplicate writer, stale/foreign owner, unsupported cross-machine, and
  unsupported multi-instance requests.

## 2. Source owner enforcement

- [x] 2.1 Update checkpoint/resume store ownership and claim validation so a
  same-host sequential reopen preserves logical identity but duplicate or
  stale ownership fails closed.
- [x] 2.2 Update native history readers and cursor evidence to declare writer
  and transaction scope; reject unsupported concurrent cursor advancement
  without latest fallback or gap suppression.
- [x] 2.3 Update scheduler/mailbox/composer/workflow/team source projections
  to expose truthful local ownership and persistence boundaries without
  unifying their stores.
- [x] 2.4 Add Run/Stream and source-terminal tests proving store failure or
  fence conflict does not trigger a fresh Run, enqueue, retry, or replay.

## 3. Manifest and planbridge admission

- [x] 3.1 Project deployment profile, writer scope, fencing scope, transaction
  scope, and evidence reference in the host runtime capability manifest.
- [x] 3.2 Make host execution plan compilation/admission reject unsupported
  cross-machine, multi-writer, and multi-instance restore/cursor requests
  before source activation, while preserving existing mailbox owner delegation.
- [x] 3.3 Add planbridge tests for local owner success, stale/foreign owner,
  capability mismatch, and no-side-effect rejection.

## 4. Replay, fixtures, and contract gates

- [x] 4.1 Extend reference-only diagnostics replay to distinguish local
  sequential reopen from distributed/parallel unsupported outcomes without
  invoking owner methods.
- [x] 4.2 Add shell and PowerShell deployment-scope contract gates and wire
  them into the quality gate and mainline contract-test index.
- [x] 4.3 Add bounded overlap/duplicate and conflict fixtures for ownership,
  cursor advancement, checkpoint claim, and composite graph claim.
- [x] 4.4 Verify gate parity, fixture drift classification, and replay
  side-effect freedom on both supported and rejected paths.

## 5. Documentation and release hygiene

- [x] 5.1 Update integration, runtime module boundaries, runtime diagnostics,
  package READMEs, capability manifest documentation, and roadmap status with
  the exact same-host single-writer boundary.
- [x] 5.2 Keep `proposal.md`, `design.md`, and `tasks.md` Public Semantic
  Documentation Commitment and Example Impact Assessment declarations
  identical and valid.
- [x] 5.3 Run OpenSpec validation, documentation impact/example/roadmap gates,
  `git diff --check`, and confirm no active proposal claims distributed support.

## 6. Verification and archive

- [x] 6.1 Run affected package tests with positive, negative, boundary, and
  Run/Stream parity coverage.
- [x] 6.2 Run `go test ./...`, `go test -race ./...`, and
  `golangci-lint run --config .golangci.yml` with a worktree-local cache.
- [x] 6.3 Run the full PowerShell quality gate and record any external advisory
  or environment limitation without weakening contract assertions.
- [x] 6.4 Archive with `scripts/openspec-archive-seq.ps1`, review the archive
  index and verification evidence, then commit, merge, push, and verify
  `master` equals `origin/master`.

### Example Impact Assessment

无需示例变更（附理由）：existing examples remain on the supported local
single-writer profile and do not need new runtime markers. Capability fixtures,
offline replay, and contract gates provide the new evidence instead.

### Public Semantic Documentation Commitment (draft)

对外语义变化：是。 The tasks change public capability, ownership, deployment
scope, and error semantics. Required synchronized documentation paths are
`docs/guides/integration.md`, `docs/runtime-module-boundaries.md`,
`docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`,
the affected package READMEs, and the archived verification record. Owners are
the source packages and planbridge named above; evidence is the dedicated
shell/PowerShell gates, Go tests/race/lint, OpenSpec validation, and docs
consistency checks.

## Documentation Impact Assessment

| Area | Outcome | Maintained path | Owner | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | source owners | deployment scope gate |
| components | 修改文档 | `core/types/README.md` | core/types | Go contract tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics owner | docs consistency gate |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | planbridge owner | OpenSpec validation |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics owner | replay fixture |
| examples | 修改文档 | `docs/guides/integration.md` | integration owner | integration guide |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration owner | contract gate |
| best practices | 修改文档 | `docs/guides/integration.md` | source owners | fail-closed tests |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | roadmap status gate |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `docs/development-roadmap.md`, `core/types/README.md`, `core/runner/README.md`, `orchestration/README.md`, `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md`, `orchestration/scheduler/README.md`, `orchestration/composer/README.md`, `orchestration/workflow/README.md`
- 维护 owner：core/types, core/checkpoint, core/history, orchestration source owners, planbridge
- 验证证据：deployment scope Go tests, replay fixture, shell/PowerShell contract gates, docs consistency, OpenSpec validation
