## 1. Source Audit and Contract Freeze

- [x] 1.1 Record child/fanout ownership across Composer and Scheduler, including
  task/attempt/lease, local/A2A dispatch, terminal commit, recovery snapshot,
  and existing unsupported planbridge cells; add source-boundary tests.
- [x] 1.2 Freeze bounded child continuation schema, canonical digest inputs,
  parent/run/session/causation/authority/plan/write-scope/effect identities,
  pause reasons, generation, expiry, and same-host single-writer limits; add
  exact-boundary and oversize tests.

## 2. Child Continuation Material

- [x] 2.1 Add detached child continuation reference and persisted graph/task
  state DTOs for child/fanout identity, dependencies, pending/completed child
  actions, attempts, leases, budgets/deadline, effect references, terminal
  arbitration, and predecessor/successor lineage.
- [x] 2.2 Add source-owned child continuation store with MemoryStore and
  FileStore implementations, canonical normalization/digest, detached maps and
  slices, reopen validation, and typed malformed/corrupt/foreign/expired/
  unavailable/conflict errors.
- [x] 2.3 Add store crash-window tests for save, durable commit, load, claim,
  running transition, successor pause, and terminal result persistence.

## 3. Atomic Child Restore

- [x] 3.1 Add Composer child continuation admission and full normalized resume
  identity validation, including source generation, task/attempt/lease,
  authority, plan, scope, effect namespace, occurrence, causation, revision,
  budget, predecessor, and capability subset.
- [x] 3.2 Add atomic duplicate/conflict claim behavior and a restore entry that
  resumes the existing child graph without SpawnChild, replacement task,
  enqueue, parent Run, or history replay.
- [x] 3.3 Reopen a FileStore from a new Composer and skip completed child
  effects/terminal attempts; verify local and A2A source identities and native
  terminal projection remain unchanged.
- [x] 3.4 Define cancellation, timeout, retry, lease loss, claim/start, and
  terminal-commit precedence; expose in-progress/recovery-required without
  automatic takeover.

## 4. Child Pause Producers

- [x] 4.1 Add child `input_required` safe-point persistence before the pending
  local/A2A action and verify pending action, prior outcomes, source evidence,
  Run/Stream parity, and no dispatch before resume.
- [x] 4.2 Add child `approval_required` safe-point persistence before effect
  dispatch, including deny/timeout terminal evidence and duplicate approval
  protection.
- [x] 4.3 Add child `budget_exhausted` persistence with consumed resources and
  original deadline; verify current budget revalidation, no replenishment/reset,
  and deterministic resume.

## 5. Capability and Plan Bridge

- [x] 5.1 Add independent child/fanout checkpoint, restore, pause, cancellation,
  timeout, retry, retention, expiry, evidence, and storage-limit manifest
  cells; verify the version/build/API/evidence fixture.
- [x] 5.2 Keep team parallel/vote, mailbox host-plan continuation, and resumed
  planbridge cells explicitly `unsupported_capability`; verify rejection before
  member activation, publish, enqueue, Run, Stream, or direct fallback.
- [x] 5.3 Add an evidenced resumed child binding to planbridge that delegates to
  the source restore entry once and returns native projection; reject reused run
  IDs/checkpoint refs without binding or incompatible contract versions.

## 6. Replay and Integration Gates

- [x] 6.1 Add versioned reference-only child continuation fixtures for graph and
  identity drift, duplicate/conflict, foreign/stale/expired/corrupt,
  lease-loss, crash windows, cancel/timeout/retry races, terminal evidence,
  unsupported forms, and Run/Stream parity.
- [x] 6.2 Add focused child/local/A2A, new-Composer FileStore restore, duplicate
  claim, pause-reason, and no-side-effect integration tests.
- [x] 6.3 Add shell and PowerShell child continuation contract gates, wire them
  into quality/replay paths, and verify intentional fixture drift is rejected.

## 7. Documentation and Example

- [x] 7.1 Update README, roadmap, integration guide, runtime module boundaries,
  runtime diagnostics, and contract index with child-first scope, source
  ownership, storage limits, unsupported team/mailbox/resumed matrix, and
  rollback semantics.
- [x] 7.2 Update composer and planbridge package READMEs plus the durable
  continuation example markers/rollback notes; do not add a second execution
  semantic or model-dependent example.

## 8. Verification and Release

- [x] 8.1 Run focused composer/scheduler/planbridge/replay tests including
  positive, negative, boundary, restart, race, lease-loss, and no-side-effect
  cases; record evidence in `verification.md`.
- [x] 8.2 Run strict OpenSpec validation, diff, documentation/example/roadmap
  gates, package contract gates, and verify replay is not treated as restore.
- [x] 8.3 Run `go test ./...`, `go test -race ./...`,
  `golangci-lint run --config .golangci.yml`, quality gate, and docs gate;
  record environment-limited failures without weakening assertions.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/composer/README.md`、`orchestration/planbridge/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：composer/scheduler、planbridge、diagnostics/replay、documentation maintainers
- 验证证据：`openspec validate restore-composite-continuations-and-graph-state --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| 领域 | 结论 | 维护路径 | owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | composer/scheduler | documentation impact gate |
| components | 修改文档 | `orchestration/composer/README.md` | composer/scheduler | package contract tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | composer/scheduler | documentation impact gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | composer/planbridge | strict OpenSpec validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay | replay gate |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | documentation maintainers | example impact gate |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | planbridge | contract gate |
| best practices | 修改文档 | `docs/guides/integration.md` | documentation maintainers | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
