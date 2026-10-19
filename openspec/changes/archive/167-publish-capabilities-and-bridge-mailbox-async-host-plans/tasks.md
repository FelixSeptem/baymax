## 1. Proposal baseline and public contract wiring

- [x] 1.1 Update `docs/development-roadmap.md` current-status entry for this active Spec 1 change and verify `openspec list --json` agrees with `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`.
- [x] 1.2 Complete the public API, owner-boundary, typed-error, compatibility and rollback documentation named by the Documentation Impact Assessment before marking code tasks complete; verify every listed maintained path is present in the branch diff and `pwsh -File scripts/check-openspec-documentation-impact.ps1` passes.
- [x] 1.3 Define versioned capability-manifest negative vectors for missing dimensions and unsupported pause/history/restore dimensions; verify parsing and normalization tests do not invoke a source.

## 2. Capability manifest

- [x] 2.1 Add bounded immutable public manifest DTOs/read interface with contract version, build/module revision, public API version, execution-form dimensions, retention policy, evidence reference and unsupported reason; verify unit tests reject missing/oversized/invalid declarations and caller mutation cannot change a returned snapshot.
- [x] 2.2 Populate the initial capability matrix conservatively from existing source owners, including explicit unsupported history/replay/checkpoint/restore/pause dimensions; verify planner queries never infer support from a package, binding type or DTO presence.
- [x] 2.3 Add admission compatibility checks for manifest/API version and current source availability; verify unavailable/incompatible/undeclared `mailbox_async` fails with typed unavailable or `unsupported_capability` before publish, Run, Stream, retry or restore.

## 3. Mailbox public binding and outcome contract

- [x] 3.1 Add bounded source-owned mailbox host-plan binding DTOs for mailbox contract version, message identity, parent run, causation, authority digest, plan digest, write scope, effect namespace and resource limits; verify invalid or conflicting fields fail before any mailbox record is created.
- [x] 3.2 Add detached mailbox admission/outcome DTOs that retain message ID, delivery/lease ID, attempt/retry causation, ack/nack/retry/DLQ result, original run/parent run, host correlation and terminal outcome without body, credential, callback-error or business-commit fields; verify positive, redaction/bounds and mutation-isolation cases.
- [x] 3.3 Extend the existing mailbox owner with an idempotent host-plan admission operation using the full logical identity; verify exact duplicate convergence preserves original authority/occurrence/effect namespace and conflicting reuse fails closed without overwriting message, delivery, lease, attempt or outcome.
- [x] 3.4 Cover source-native lease loss, timeout, retry and DLQ projections; verify their attempt/terminal evidence retains original causal identity and no path re-runs a parent Run.

## 4. Planbridge compile and admission integration

- [x] 4.1 Extend `host_execution_plan.v1` binding snapshot and limits for `mailbox_async`; verify mailbox configuration/version, message identity, parent/causation, authority/plan digest, write scope, effect namespace and resource-limit fields participate in bounded preflight.
- [x] 4.2 Make `Compile` admit mailbox delivery only with a compatible manifest declaration and mailbox binding; preserve existing inline and unsupported-resume behavior, and verify rejected mailbox requests publish/enqueue/claim nothing and never fall back to direct execution.
- [x] 4.3 Make activation revalidate current host admission and mailbox owner availability before one native-owner admission; verify planbridge does not own queue, worker, scheduler, lease, retry, DLQ or terminal state.

## 5. Execution outcome and parity conformance

- [x] 5.1 Project the mailbox owner’s admission and lifecycle/terminal snapshots through planbridge without synthesizing a parent Run or business commit; verify ack/nack/retry/DLQ remains distinct from terminal and host effect facts.
- [x] 5.2 Add real-owner integration tests with fake host authorization/effect collaborators for successful mailbox admission, exact duplicate, conflicting duplicate, unavailable/version mismatch, lease loss, timeout, retry and DLQ; verify each rejected case has no source side effect.
- [x] 5.3 Add Run/Stream parity tests for one logical mailbox continuation; verify normalized terminal outcome, original authority/occurrence/effect namespace and source correlation agree without a second logical admission or fabricated callback/event.

## 6. Replay, gates, examples and maintained documentation

- [x] 6.1 Add versioned reference-only replay fixtures and drift classifications for manifest evidence/version drift, binding mismatch, duplicate conflict, no-side-effect rejection, lease-loss/retry/DLQ causation and Run/Stream parity; verify replay is read-only and invokes no mailbox, handler, model, tool, host callback or restore path.
- [x] 6.2 Add equivalent shell and PowerShell mailbox-plan-bridge contract gates and quality-path wiring; verify each gate runs the contract/replay/integration evidence and detects removal of a required negative boundary.
- [x] 6.3 Update `examples/host-execution-plan/README.md` before changing its example code with semantic anchor, mailbox runtime path, manifest support matrix, expected markers and rollback notes; then add an offline fake-owner example and verify its documented supported/rejected markers without a provider or business database.
- [x] 6.4 Synchronize README, integration, package, module-boundary, diagnostics, best-practices and contract-test-index documentation with the delivered APIs and evidence; verify no document claims Spec 2 history/replay or Spec 3 durable restore support.

## 7. Final verification and OpenSpec completion evidence

- [x] 7.1 Run targeted planbridge/mailbox/integration/diagnostics-replay/contribution tests, then `go test ./...`, `go test -race ./...` and `golangci-lint run --config .golangci.yml`; record actual commands, exit codes, risks, rollback evidence and any environmental limitation in `verification.md`.
- [x] 7.2 Run `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, the dedicated shell/PowerShell contract gates and `openspec validate publish-capabilities-and-bridge-mailbox-async-host-plans --strict`; verify all proposal/design/tasks commitments, example impact declaration and public-documentation evidence remain consistent before archive.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`
- 维护 owner：orchestration、mailbox、host-integration 与 documentation maintainers
- 验证证据：OpenSpec strict validation、documentation impact gate、roadmap consistency 与 mailbox plan bridge contract gate

The roadmap is the current synchronized planning document. Tasks 1.2 and 6.3–6.4 are mandatory completion gates for all final public semantics and maintained documentation; no implementation or archive claim is valid while their paths are absent from the branch diff.

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/development-roadmap.md` | orchestration maintainers | 复核 owner 分层及禁止第二 queue/worker/scheduler/state machine |
| components | 修改文档 | `orchestration/planbridge/README.md`、`orchestration/mailbox/README.md` | orchestration/mailbox maintainers | manifest、binding/outcome、owner delegation 与 support matrix 对照 source tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | mailbox availability/version 的 constructor/admission 语义；无新增配置时明确保持 `env > file > default` |
| contract/API | 修改文档 | `docs/guides/integration.md`、`docs/mainline-contract-test-index.md` | contract and host-integration maintainers | spec requirements、public DTO、Run/Stream integration、fixture/replay/gate 对照 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | source-owned bounded correlation/outcome；RuntimeRecorder 单写且无新 QueryRuns contract |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples maintainers | README 文档先行、offline marker、no-fallback 与 rollback 路径验证 |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host-integration maintainers | compile/admit/dispatch、typed rejection、duplicate convergence 与 host boundary |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | digest/ack/DLQ 不等于 business commit 或 restore 的反例核对 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | `openspec list --json` 与 roadmap status consistency |
