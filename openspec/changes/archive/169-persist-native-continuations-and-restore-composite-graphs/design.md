## Context

See `proposal.md` for motivation. The current dynamic-action checkpoint map is owned by one Engine process, while workflow persistence and mailbox history are separate source contracts. The design must add a durable source boundary without introducing a second scheduler, queue, mailbox owner, event journal, or business-state store.

## Goals / Non-Goals

**Goals:**

- Make direct native checkpoint state reopenable by a new Engine through an injected durable store.
- Provide atomic, idempotent claim/resume semantics with bounded identity, digest, expiry, causal and pause facts.
- Preserve host revalidation ordering, Run/Stream parity, terminal evidence, and fail-closed error behavior.
- Make the form × pause-reason manifest authoritative, with direct evidence required and composite cells unsupported until proven.

**Non-Goals:**

- No new parent Run, retry loop, queue, scheduler, mailbox owner, event-history store, or business commit owner.
- No claim that workflow checkpoints, mailbox retained history, callback events, metadata, snapshots, or reused IDs restore runner continuation.
- No planbridge resumed execution, composite graph restore, or hidden best-effort fallback in this phase.

## Decisions

### Durable store and checkpoint envelope

The repository already exposes a metadata-oriented `RunCheckpointStore` with `Save`, `Load`, and `AdmitResume`. Preserve that interface and its implementers; add a versioned durable extension (`DurableRunCheckpointStore`) with actual continuation material and atomic `ClaimResume`, and inject it into a new Engine restore path. A `PersistedRunCheckpoint` is immutable after save; updates are source-owned state transitions with a schema/version and digest over canonical bounded fields. The implementation supplies a reopenable file-backed store for integration tests and deployments; memory storage is test/process-local and cannot satisfy cross-process capability. The envelope stores normalized request references and bounded tool-result references, never arbitrary callbacks, credentials, or business payloads.

### Claim state machine

The source state moves through `constructed -> persisted -> revalidated -> claimed -> running -> paused-successor|terminal`. Each transition is conditional on the prior digest/generation. Host verification precedes claim; duplicate equivalent claims read the retained claim/result; conflicting, foreign, expired, corrupt, consumed, or unavailable claims fail closed. Store and terminal write failures remain visible and never trigger a new Run. A crash after claim/start without an atomic terminal/successor record is projected as in-progress or recovery-required, not automatically retried; exactly-once execution across arbitrary external side effects is not claimed.

The durable extension exposes `ReadResumeClaim(ctx, ref)` as a detached, read-only source lookup. The runner performs current host verification before projecting any retained result, compares the full original request identity (including the proposed budget), and reuses the retained authorized budget for exact duplicates. A completed duplicate does not authorize a second budget increase. Budget proposals and source-authorized budgets are persisted separately, since an owner may normalize a proposal before admitting it. Legacy compatibility decisions resolve their original checkpoint identity rather than the latest successor for that run.

### Host admission ordering

For enabled host revalidation, load and binding checks occur before verifier invocation; verifier code runs outside source locks; source generation, pause reason, expiry, cancellation, and digest are rechecked before `ClaimResume`. Host callbacks authorize current authority only. They do not load source state, reconstruct requests, or declare terminal cancellation.

### Pause reasons and capability matrix

Direct source owners must produce source evidence for `input_required`, `approval_required`, and `budget_exhausted`; all three are completion requirements. Approval pauses a pending high-risk tool before dispatch. Budget resume rechecks current source budget and cannot refill it. The manifest is cell-based. Child, team, workflow, mailbox, and planbridge resumed cells remain unsupported and admission rejects before activation. Later composite work must persist graph/member/step/dependency/budget/native-ID state and prove no completed work is rerun.

### Run/Stream and replay semantics

Run and Stream use the same source claim and terminal rules; only transport projection differs. Loading a checkpoint or replaying a fixture is read-only. Provider/tool execution begins only after a valid claim and source continuation start. Native event history remains an independent source journal and is not a restore mechanism.

### Errors, races, and limits

Expose bounded typed errors for malformed, foreign, expired, corrupt, unavailable, unsupported, conflicting, in-progress, and terminal checkpoints. Cancellation/timeout/retry/resume precedence comes from source control state; accepted cancellation is not terminal evidence. Limits cover serialized envelope, references, digests, result references, expiry and retained duplicate results.

### Public API profile and dependency binding

新增 `native_continuation.v1` profile；具体 Go 字段名可在实现时细化，语义与版本不得静默改变：

- `CheckpointRef`：schema/version、checkpoint ID/digest、source/owner namespace、run/session、created/expiry、pause reason、source position、predecessor/attempt。原 parent、host correlation、authority/plan/write-scope、occurrence/effect namespace 作为绑定原样保留。
- `PersistedRunCheckpoint`：public ref、generation、continuation material reference/digest、依赖 profile/revision、mode、iteration、completed/pending work、累计预算和 deadline。
- `DurableRunCheckpointStore`：`Save(ctx, persisted)`、`Load(ctx, ref)`、`ClaimResume(ctx, request)`，并提供条件式 source control transition 与 `CommitResumeResult`。三方法本身不足以持久化 cancellation、successor 与 terminal；这些操作属于同一 store/source owner 的扩展，不另设 host registry。
- `Engine.ResumeFromCheckpoint(ctx, HostResumeRequest, handler, stream)`：显式恢复 API，不调用 public Run 入口重新分配逻辑执行。既有 `ResumeDynamicAction` 和 host `run.action.resume` 在 durable binding 下委托同一 source guard；旧 mode/decision 与结果读取授权保留。

旧 `RunCheckpoint`/`RunCheckpointStore` validation 和 signatures 保留。新 profile 使用独立 DTO 支持三类 reason，不能把旧 `PendingAction.Kind` 改值就声称产生 approval/budget pause。

公共 checkpoint envelope 保持 reference-only；实际恢复所需的 normalized request/model context、completed tool outcomes、pending call arguments、loop position 和 accounting 是 source-owned private continuation material，必须随 source checkpoint 一起可持久化。FileStore 保存有界 immutable material record，再原子发布 checkpoint reference；只有 material 可重开读取时才能提升 manifest。不得只落 metadata 然后依赖旧 Engine map、host callback 或重新调用 model/tool 补 state。禁止复制 PendingAction 业务 body、业务账本或 credential。公开文档须说明 private source material 含继续执行所需的运行数据，不能承诺整个持久化文件只有 references。

新 Engine 通过显式注入的相同 model/tool/context dependency profile 继续既有 loop。执行代码/客户端不序列化；source material 中 revision/digest 与当前绑定不符返回 typed unavailable/foreign。可选 material resolver 必须读取 source 保留的 immutable material 并验证 digest，不可生成替代计划、重新执行工具或由 host verifier 构造 continuation。handler 为当前调用提供，只订阅后续事实，不重放旧 callback。

### Bounds and retention

公开 envelope 上限为 64 KiB，identity/reference 为 256 UTF-8 bytes、digest 为 128 bytes，reference 列表最多 256 项；source material 独立限 1 MiB、1024 个 completed/pending entries；先计数/限长再 clone/encode。零覆盖选择默认，正值只收紧，负值或放宽失败。边界与超一均须 fixture/test。

checkpoint 默认保留 24 小时，expires 必须晚于 created 且不得晚于原绝对 deadline；无绝对 deadline 时仍受 checkpoint TTL 限制。重复 admission/result 从完成时至少保留 24 小时并公开 retained-until。过期返回 expired/not-found，不猜测 duplicate success。活跃 claim 和 terminal tombstone 不因普通 checkpoint GC 重新变为可 claim。FileStore 不自动删除 material；显式清理以 retained references 为准且不得恢复旧 checkpoint 的执行资格。源记录数量由 constructor 的正容量限制，满时 unavailable，不驱逐活跃记录。配置为 constructor opt-in，不新增 env/file/hot-update 键。

### FileStore atomicity and process ownership

FileStore 第一版 scope 是本机单写进程、可跨进程顺序重开。通过 OS exclusive file lock/fencing generation 防止两个 Engine/store handles 同时持有写权限；第二 writer 返回 unavailable。只允许经 conformance 验证具备 lock、atomic replacement 与 sync 语义的本地文件系统，不宣称跨机器分布式 CAS。进程退出释放锁，新进程重开验证 schema/checksum/generation。

持久化顺序为 write temporary record、sync material、atomic publish source snapshot、完成平台对应的 durable sync。成功 Save/Claim/Commit 只能在 durable acknowledgement 后返回。若平台无法保证所声明的 crash durability，不可 advertise durable restore。部分写入与 lock/sync/rename 错误均 fail closed，不静默使用 MemoryStore。

### Direct pause production and opt-in behavior

| pause reason | source boundary | resume eligibility | preserved evidence |
| --- | --- | --- | --- |
| input_required | 已完成 tool-result 后、下一 model 前 | 匹配 opaque action decision、current host admission 通过 | 原 action、已完成 outcomes、下一 iteration |
| approval_required | Action Gate require_confirm、pending 高风险 tool dispatch 前 | 当前 approval owner confirm；deny/timeout 走 source cancellation | 未执行 call、同批已完成 call、policy/revision/confirmation reference |
| budget_exhausted | 迭代或 tool-call 预算耗尽、下一次受限调用前 | budget owner 提供带 revision 的新上限且 remaining 足够、已用量不减 | consumed counters/cost、pending work、原 time/deadline |

新增 durable approval opt-in 是既有 blocking resolver 的显式替代 path；默认仍保持 resolver missing/timeout 的 fail-closed 旧语义。budget opt-in 仅在运行中 safe point 产生 pause；既有 admission hard deny、不可恢复 timeout、sandbox/provider fatal failure 不改成可恢复暂停。预算增加由 source budget owner 批准，host verifier 只验证 current eligibility；只读 `budget_projection.v1` 不能写预算或授权补充。尚无可靠 source counter 的维度保持 unsupported，不可 reset estimate。三类 direct reason 均须真实 producer 和跨进程测试才可完成本 baseline。

### Race precedence and crash windows

Source control 和 resume claim 使用同一 generation 条件提交。请求 context canceled 在 claim 前拒绝且不消耗 checkpoint。source terminal/cancel 或绝对 deadline 已先提交则阻止 claim；claim 先提交时后续 cancellation 必须由该 attempt 的 source safe point 确认，不能回滚为 unclaimed。过期与 resume 以 claim 时 source clock 为准；retry 永远不能消费 checkpoint 或解除 claim。等价 duplicate 可只读已保留 result，仍须独立结果读取授权；非等价请求不能绕过 expiry/control 验证。

| crash window | durable source fact | next Engine behavior |
| --- | --- | --- |
| 构造后、Save 前 | 无公开 persisted checkpoint | not-found/unavailable；不确认 durable pause，不重新 Run |
| material 已写、checkpoint 未发布 | 不可引用的 material | 不恢复，不暴露半成品 checkpoint |
| Save 完成、response 丢失 | persisted pause 与 ref | 读取相同 ref、current revalidation 后可 claim |
| host callback 通过、claim 前 | 仍是 persisted pause | 重新读取当前业务事实，不复用旧授权结果 |
| claim 已提交、continuation 未开始 | retained claim/attempt | duplicate 返回 in-progress/recovery-required，不自动 reclaim |
| continuation 已开始、结果未提交 | running attempt | 同上；不猜测 tools/effects，不自动 retry |
| successor/terminal 提交后、response 前 | result 或新 checkpoint + predecessor result | exact duplicate 只读既有 projection，不重发 callback/terminal |
| terminal write 失败 | retained attempt 与 unavailable | 不宣称 durable terminal，不重跑补结果 |

durable acknowledgement 是 pause 对外确认点；successor/result 与旧 claim completion 必须同一 source transaction 提交，防止新 checkpoint 丢失或旧 request 消费新 generation。RuntimeRecorder/callback 投影失败不撤销 source commit，不触发 continuation。

### Error and evidence contract

error 支持 `errors.Is`：unsupported、invalid/malformed、not-found、schema/version、digest/corrupt、foreign/ownership、expired、conflict、in-progress、recovery-required、terminal、source-unavailable。错误只含有界 field/code/correlation，不含 snapshot body 或 raw callback error。已 admitted 后的 continuation failure 与 admission rejection 分开；terminal 来自 source commit，不来自 command accepted。

沿现有 native event / RuntimeRecorder 路径发送 checkpoint persisted、claim accepted、restore started、successor paused、source terminal 或 recovery-required 有界事实，保留 predecessor/attempt/source position 和原 host bindings。callback EventID 不变成 journal identity；runner history 当前仍 unsupported，checkpoint position 仅解释 runner 自身 continuation generation，不声称 retained event cursor 或 child/team 全局顺序。诊断字段 additive + nullable + default，经单写入口投影；history read 与 fixture replay 不写诊断或 source store。

### Composite follow-up acceptance

本次支持矩阵目标为 direct 三类 reason + 跨进程 restore；child/fanout、team parallel/vote、workflow、mailbox 四行全部 unsupported，planbridge resumed 也继续 unsupported。BXM-4 原验收不被删除：后续增量须保存组合/成员/step dependency/预算/未决 action/native IDs，恢复原 graph、跳过已完成 member/step、保留 parent/child/member/step/attempt causality，并验证 failed/canceled/over-budget member 与 aggregate outcome 稳定性。

每个格子顺序为 manifest schema/evidence 定义、admission guard、source restore 与测试证据、最后启用 supported；不得在实现前把 supported 当成现有事实。已有 workflow FileStore 和 mailbox journal 不提升这些格子。

## Risks / Trade-offs

- [Non-serializable continuation] -> Require normalized references and an explicitly configured Engine binding that can rehydrate source dependencies; fail closed when the new Engine cannot reconstruct the source boundary. Do not serialize arbitrary callbacks or provider clients.
- [Multi-writer store races] -> Use source-owned compare-and-swap/locking and generation checks; test separate store handles and process reopen.
- [Sensitive data leakage] -> Keep payloads and credentials outside the checkpoint, bound all fields, and route diagnostics through `RuntimeRecorder`.
- [False capability advertising] -> Generate manifest cells only from source evidence and reject unsupported combinations before activation.
- [Crash between claim and terminal write] -> Persist claim/attempt identity and expose an in-progress/recovery-needed source outcome; never silently retry or create a new parent.

## Migration Plan

1. Ship the store and DTOs behind an explicit durable capability configuration; existing in-memory dynamic-action behavior remains unchanged when not enabled.
2. Enable direct durable cells only after contract fixtures, new-Engine integration, race tests, and Run/Stream parity pass.
3. Keep composite and planbridge resumed cells unsupported; later source-owner changes can add cells through a separate OpenSpec.
4. Roll back by disabling durable capability advertisement and retaining source checkpoints for inspection; do not delete or reinterpret persisted records.

## Open Questions

None. Serialization limits, claim identity, pause matrix, and unsupported boundaries are specified for implementation planning.

## Example Impact Assessment

新增示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/runner/README.md`、`core/types/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：runner/types maintainers、host integration maintainers、documentation and examples maintainers
- 验证证据：`openspec validate persist-native-continuations-and-restore-composite-graphs --strict`、OpenSpec documentation/example/roadmap gates、`scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime architecture maintainers | source store and restore ownership boundaries documented |
| components | 修改文档 | `core/runner/README.md`, `core/types/README.md` | runner/types maintainers | public DTO and Engine responsibility documented |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime diagnostics maintainers | typed checkpoint failure and diagnostics documented |
| contract/API | 修改文档 | `docs/guides/integration.md` | host integration maintainers | direct restore and unsupported scope documented |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime diagnostics maintainers | checkpoint facts use canonical writer |
| examples | 新增文档 | `examples/durable-continuation-restore/README.md` | examples maintainers | offline scenario baseline documented |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | admission and failure behavior documented |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | restore versus retry/replay guidance documented |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active Spec 3 scope documented |
