## Why

集成方的 application outbox、业务 effect commit 与 Baymax mailbox/scheduler 投递目前只有相邻能力，缺少可验证的交接合同。崩溃可能发生在 publish/enqueue 后尚未确认 outbox，或宿主 effect 已提交但 mailbox ack/调度收口尚未完成；若没有稳定 correlation、当前准入和明确的重试 owner，恢复会产生重复 effect、过期 attempt 提交或把 delivery/terminal 误判为业务成功。

现在需要在归档 160/162/164/165 已固定的 source correlation、plan binding、恢复准入和 reference-only host 边界上，交付一个显式 opt-in 的 application outbox dispatch bridge。该 bridge 同时支持 mailbox delivery 和 scheduler task dispatch，但不迁移宿主 outbox、cron/DST、业务提交或 effect 幂等事实。

## What Changes

- 新增 versioned、bounded、reference-only 的 `application_outbox_dispatch.v1` descriptor、trusted host binding 与 constructor-only opt-in bridge；descriptor 关联 application outbox、occurrence、business revision、authority/effect-idempotency references、target binding 和 existing run/message/task/attempt correlation，但不承载业务 body、webhook 内容、credential 或 callback error。
- 支持两个显式且互斥的 target：`mailbox` 委托既有 publish/claim/ack/nack/requeue/delay/DLQ owner；`scheduler` 委托既有 task/claim/lease/retry/terminal/DLQ owner。bridge 在 target admission 前验证 descriptor、current host binding、target-specific correlation 与 source capabilities，且不成为消息存储、lease/retry/DLQ、terminal 或业务提交 owner。
- 固定 delivery accepted、scheduler/execution terminal、host effect committed/outbox confirmed 为彼此独立的事实。bridge 不得从 publish、ack、task terminal、Run/Stream terminal 或 replay 推断业务 commit；恢复保持 at-least-once，宿主以 effect idempotency/CAS 证明业务副作用收敛。
- 覆盖 publish/enqueue 后未确认 outbox、effect commit 后未确认 delivery 两个崩溃窗口，以及 duplicate dispatch、mismatched idempotency、stale attempt、lease rollover、delay、cancel、recovery、DLQ 与 target drift。delivery retry、execution retry、business commit retry 分别保留给 mailbox、scheduler、host，bridge 不得叠加重试。
- 提供 host-neutral fake outbox/effect-store 离线示例、两类 target 的真实 owner integration、Run/Stream 必要路径对等证据、版本化 replay fixture、contribution checks、shell/PowerShell contract gate 和标准 quality-path wiring。
- 同步维护公开 API、bounded limits、错误分类、target support matrix、owner 边界、at-least-once/commit 语义、回滚说明与 contract-test index；不增加 env/file/hot-update 配置、QueryRuns 字段、host wire command、Provider SDK、业务数据库、remote event store 或第二套 execution/recovery state machine。

## Capabilities

### New Capabilities

- `application-outbox-mailbox-recovery`: Defines the bounded opt-in bridge from a host-owned application outbox to the existing mailbox or scheduler owners, including admission, correlation, crash-window recovery, idempotency, target-specific delegation, business-commit separation, replay and conformance requirements.

### Modified Capabilities

- None. Existing mailbox and scheduler capabilities retain their native publish/lease/retry/DLQ semantics; this change adds a host-facing bridge without redefining their requirements.

## Impact

Expected implementation areas are a new orchestration bridge package, narrow adapter seams in existing mailbox/scheduler/composer recovery owners where a source-owned public integration hook is proven necessary, `core/types` only for bounded public DTOs if required, `integration`, `tool/diagnosticsreplay`, `tool/contributioncheck`, a dedicated shell/PowerShell gate and `examples/application-outbox-recovery`. Existing direct mailbox and scheduler calls remain compatible and the bridge is disabled unless explicitly constructed.

The host retains application outbox storage, occurrence identity, schedule revision, cron/IANA timezone/DST, misfire/backfill policy, effect key, business authorization and the effect/outbox commit CAS. Mailbox retains delivery lifecycle; scheduler retains attempt/lease/execution lifecycle; source runner retains Run/Stream and terminal semantics; `RuntimeRecorder` remains the sole diagnostics writer. The bridge does not promise exactly-once, cross-system atomic commit, durable runner continuation or rollback of an already committed business effect.

## Risks and Rollback

Primary risks are duplicate or stale dispatch becoming an unintended effect, a target acknowledgement being read as a business commit, source drift after preflight, retry amplification across three owners, and a lease rollover accepting an old occurrence. The contract therefore requires full identity equality, current host admission at dispatch/recovery boundaries, source-owner rechecks, explicit target/result states and host effect idempotency/CAS. Descriptor hashes prove normalized binding consistency only; they never prove business authority or a completed effect.

Rollback stops new bridge dispatch and returns callers to direct mailbox/scheduler APIs. It preserves unconfirmed outbox records, task/attempt/message facts and already committed effects for host-owned idempotent reconciliation; it never deletes pending facts, cancels unrelated delivery, or claims to undo a business write.

## Example Impact Assessment

新增示例

`examples/application-outbox-recovery` will provide a fake outbox/effect-store, legal mailbox and scheduler dispatches, both crash-window recoveries, duplicate convergence and rollback notes without a business database or live provider. Its README will be established before example code with semantic anchors, runtime paths, expected markers and rollback notes.

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

## Validation Strategy

Proposal stage: `openspec validate establish-application-outbox-mailbox-recovery-contract --strict`; `pwsh -File scripts/check-openspec-documentation-impact.ps1`; `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`; `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`; and `git diff --check`.

Implementation stage: targeted package/unit tests for the bridge, mailbox, scheduler, composer recovery, integration, diagnostics replay and contribution checks; equivalent `scripts/check-application-outbox-mailbox-recovery-contract.sh/.ps1`; then `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`. `verification.md` records only commands actually run, their exit codes, environmental limitations, risks and rollback evidence.
