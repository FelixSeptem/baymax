## Context

The roadmap candidate and `proposal.md` establish the product boundary. Existing mailbox and scheduler packages already own their respective delivery, lease, retry and DLQ lifecycles; composer/snapshot recovery and the host execution/resume contracts establish adjacent correlation and current-admission patterns. No existing public integration binds a host-owned application outbox and effect idempotency record to both target types across the two crash windows without conflating source lifecycle with business commit.

The implementation must remain library-first: the host owns persistence, schedule/occurrence policy, authorization and business effects; `context/*` cannot take Provider SDK dependencies; diagnostics use `RuntimeRecorder` only; and Run/Stream parity cannot introduce a parallel terminal/recovery owner.

## Goals / Non-Goals

**Goals:**

- Deliver an explicit constructor-only bridge that admits a bounded, reference-only host dispatch descriptor and delegates it to exactly one existing target: mailbox or scheduler.
- Make complete logical dispatch identity, current host binding, target admission and host-effect idempotency observable to contract/replay tests without storing host payloads.
- Make the two crash windows converge at-least-once while retaining target-native lifecycle and host-owned business reconciliation.
- Supply a small offline example and parity gate that prove the target/support matrix and no-side-effect-before-rejection behavior.

**Non-Goals:**

- Implementing an application outbox store, business/effect database, cron/DST/misfire/backfill engine, webhook workflow, exactly-once protocol, distributed transaction, remote event store or hosted control plane.
- Replacing mailbox/scheduler retry, lease, DLQ, terminal or query semantics; converting planbridge's separately documented mailbox rejection into support; or creating a generic ExecutionGraph.
- Treating a bridge descriptor/digest, delivery acceptance, source terminal event, replay, or metadata snapshot as authorization, business commit, durable runner continuation, or permission to repeat an effect.

## Decisions

### 1. One bridge contract with a tagged native target

The public descriptor will contain a closed target discriminator (`mailbox` or `scheduler`) plus one target-specific reference binding. Common host facts (outbox identity, occurrence/revision/authority/effect references, correlation and idempotency) stay in a shared bounded envelope. A target adapter validates only its own binding and invokes the existing owner.

This avoids two competing application-outbox contracts while preserving target-specific semantics. A generic delivery engine was rejected because it would hide mailbox acknowledgement versus scheduler attempt/lease semantics and become a second lifecycle owner. Two independent APIs were rejected because their crash recovery and idempotency rules would drift.

### 2. Admission is host-current and source-rechecked

The bridge validates syntax, bounds and canonical identity before calling a trusted host admission callback. The callback receives reference-only facts and returns a bounded decision based on current authorization, revision/outbox eligibility and effect-key eligibility. It runs outside source locks. Immediately before target action or reconciliation, the relevant source owner rechecks its authoritative current state; callback output is never cached as permanent authority.

This follows the host-resume pattern without reusing a runner checkpoint contract for application delivery. It protects callback races while preventing lock-held host code. Missing, invalid, canceled, panicking or stale decisions fail closed.

### 3. Full logical identity drives duplicate convergence

Duplicate equality includes normalized bridge profile/version, target, application outbox identity, occurrence identity if present, business revision/effect reference, target binding, source correlation and idempotency key. Exact duplicates return or observe existing logical dispatch state; mismatches reject. The bridge stores only the bounded state needed to delegate or query the native owner; it does not introduce a second durable outbox/ledger.

Target-native idempotency remains authoritative: mailbox uses its envelope idempotency semantics; scheduler uses task/attempt semantics. The bridge's identity prevents a caller from treating a reused outbox key with changed target/revision/effect binding as a safe duplicate.

### 4. Explicit facts prevent commit conflation

Implementation refinement: `orchestration/outboxbridge/README.md` fixes the public
profile, limits, error strings and support matrix. Native envelope/task
`DispatchBinding` is an additive optional persisted digest. On duplicate publish
or enqueue the native store compares it atomically before mutation; two empty
bindings retain existing semantics, and a bound/unbound collision rejects. The
native ID derives from the idempotency key, not the mutable descriptor digest.
Cross-target key binding is checked by the trusted host against its durable
outbox, since mailbox and scheduler intentionally do not share an identity store.
Admission is a point-in-time observation; revocation versus business effect must
be protected in the host effect CAS transaction. No claim of a cross-store atomic
authorization transaction is made. `Inspect` never dispatches or reconciles a
native terminal automatically. Scheduler acceptance exposes convergent identity
without claiming a duplicate flag unavailable from the native enqueue API.

Bridge results will model at least `dispatch accepted`, `target lifecycle state`, `host effect evidence`, and `host outbox confirmation` as separate optional/reference-only facts. No result conversion can infer business commit from a mailbox ack, scheduler terminal, Run/Stream terminal or descriptor digest. Host effect evidence is supplied by a trusted idempotency/CAS collaborator; the bridge never executes or writes the effect.

This is essential for both crash windows. It lets recovery decide what is known without inventing a transaction: a target may be delivered before outbox confirmation, and an effect may be committed before source acknowledgement/reconciliation.

### 5. Retry scope is deliberately non-composable

Mailbox keeps delivery retry/requeue/lease/DLQ; scheduler keeps execution retry/attempt lease/DLQ; host keeps business commit retry. The bridge exposes their boundary facts but never automatically chains a failed source action into another owner retry. This prevents retry multiplication and preserves existing configuration/default owners.

### 6. Package and test layout follow native ownership

Implementation will use a focused orchestration package (provisionally `orchestration/outboxbridge`) for public DTO validation, canonicalization, host admission and target adapter selection. It will depend only on public mailbox/scheduler/composer/runner contracts. Target-specific behavior remains in narrow adapters and tests rather than expanding large owner files. If an existing owner needs a public hook, the task requires proving it with a failing fixture first and adding the smallest semantic API rather than exposing internal state.

Tests will group pure descriptor/admission cases with the bridge, source-lifecycle regressions with mailbox/scheduler, end-to-end crash windows with integration, and canonical fixtures with diagnostics replay. A dedicated cross-platform gate will call only the relevant suites plus the example smoke and contribution checks.

## Risks / Trade-offs

- [Stable bridge identity can diverge from native target identity] → target adapters derive and validate target-owned identity deterministically; mismatches reject before mutation and replay fixtures pin canonical fields.
- [Host callback returns stale authorization during a race] → callbacks run lock-free, source owners recheck immediately before action, and stale/cancelled/replaced state gets finite rejection rather than a synthetic success.
- [At-least-once recovery repeats an effect] → bridge does not call effects; host effect idempotency/CAS is required as a separate current fact and tests cover effect-committed-before-ack.
- [Three retry systems amplify work] → no automatic bridge retry; test each delivery/execution/business retry ownership boundary and reject code paths that compose them.
- [Mailboxes and schedulers have genuinely different completion vocabulary] → expose a tagged result/support matrix rather than flattening their native states into one terminal enum.
- [Public DTOs become unbounded or leak host data] → enforce fixed size/count limits, reference-only validators, redaction tests and diagnostics/replay fixtures without payloads.
- [Large existing owner files grow] → add small dedicated package/files and observe line budgets; split before increasing a governed file beyond its budget.

## Migration Plan

1. Add the disabled-by-default package, descriptor/profile validator and host callback interfaces with no change to direct mailbox/scheduler callers.
2. Add the mailbox and scheduler adapters behind explicit constructor binding, then pin target-specific no-side-effect rejection, duplicate, lease and DLQ behavior with fixtures.
3. Add host fake outbox/effect-store integration coverage for both crash windows; add replay/contribution checks and equivalent shell/PowerShell gate.
4. Document the target matrix, migration from explicit native calls, effect/outbox reconciliation and rollback before publishing the API/example.
5. Roll back by ceasing bridge calls or removing its constructor binding. Preserve outbox, effect, message, task and attempt facts; reconcile with host idempotency keys. Do not delete pending records or claim to reverse a committed effect.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/runtime-module-boundaries.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/README.md`、`orchestration/outboxbridge/README.md`、`orchestration/mailbox/README.md`、`orchestration/scheduler/README.md`、`examples/application-outbox-recovery/README.md`
- 维护 owner：orchestration、mailbox、scheduler、host integration、observability 与 documentation maintainers
- 验证证据：proposal/design/tasks 的一致声明；实现文档与代码/测试/示例/replay/gate 的分支差异；OpenSpec strict validation；documentation/example impact 与 roadmap/docs consistency gates；`verification.md` 记录真实 unit/integration/replay/example/gate 与完整质量门禁的实际退出结果

本 change 的公开语义包括 opt-in API、descriptor limits、有限失败分类、mailbox/scheduler support matrix、delivery/execution/business-commit 三者的非推导关系、at-least-once 恢复模型以及 rollback。上述路径已在本功能分支同步维护；当前实现已交付 `orchestration/outboxbridge`、mailbox/scheduler binding、integration/replay/example 与双平台 contract gate，真实验证证据集中记录在 `verification.md`。归档前仍需保留这些代码、测试、示例、文档和门禁证据。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`, `orchestration/README.md` | orchestration maintainers | bridge 不拥有 outbox、effect、retry、lease、terminal 或 diagnostics state |
| components | 修改文档 | `orchestration/mailbox/README.md`, `orchestration/scheduler/README.md` | mailbox/scheduler maintainers | 两类 target 保留 source-owned lifecycle 与 support matrix |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | constructor-only opt-in；无 env/file/hot-update 键，保持 `env > file > default` |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | contract/host integration maintainers | descriptor/admission/recovery、replay 和 gate 对照 delta requirements |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | 仅 additive + nullable + default；RuntimeRecorder 单写且无业务 body |
| examples | 新增文档 | `examples/application-outbox-recovery/README.md`, `README.md` | examples maintainers | README 先于 example code，覆盖双 target、两类 crash window、duplicate 与 rollback markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | 已交付 constructor-only 接入、双 target 委托、recovery/rollback 边界与可运行示例入口 |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | delivery/terminal/digest 不等于 business commit；effect CAS 仍由宿主负责 |
| roadmap | 修改文档 | `docs/development-roadmap.md`, `README.md` | roadmap maintainers | active slug 与 `openspec list --json` 一致，archive 后再更新归档状态 |
