# 宿主恢复准入与重验证

状态：功能分支已实施，交付门禁进行中。`host_resume_admission.v1` 的公开合同位于 `core/types/host_resume_admission.go`，source 接线位于 `core/runner/dynamic_action_resume.go`。本文说明实际 API、默认值、边界和迁移要求；完整验证与交付状态以 OpenSpec tasks/verification 为准。

## Source ownership

`Engine.ResumeDynamicAction` 已拥有 native checkpoint admission 与 Run/Stream continuation，Host 的 `run.action.resume` 经 readiness/通用 Authorizer 后委托该入口。已接收 confirm/deny/timeout 各产生一次 `run.dynamic_action.resolved`；deny/timeout 产生 canceled terminal，confirm 继续执行。通用 Host authorization 不定义暂停时的业务 plan/revision/write-scope 与当前事实的匹配合同。

runner 在原 Engine 中保留 request/outcomes/continuation；`RunCheckpointStore` 定义 metadata SPI，但尚未接线为 durable runner continuation。metadata、snapshot、replay 或相同 run ID 均不能恢复新 Engine 的内部执行。每次 PendingAction 使用独立 source generation 与 opaque checkpoint ID；不要解析该 ID。恢复保留原 iteration 与 pending tool results，下一模型步骤从 checkpoint iteration + 1 开始。

## 接线与可信绑定

宿主使用 `runner.WithHostResumeVerifier(types.HostResumeVerifier)` 显式安装 source runner constructor guard。默认不安装；安装后覆盖该 Engine 的 direct API 与 Host command，不能按请求关闭。`WithHostResumeVerifier(nil)` 或 typed-nil function adapter 启用 fail-closed guard，返回 `HostResumeRevalidationFailed`。

```go
engine := runner.New(model, runner.WithHostResumeVerifier(
    types.HostResumeVerifierFunc(func(ctx context.Context, req types.HostResumeRequest) (types.HostResumeResult, error) {
        // 从可信暂停记录加载 Binding，并实际检查当前授权，生成 Current。
        return trustedHost.RevalidateHostResume(ctx, req)
    }),
))
```

`HostResumeRequest{Version, Checkpoint, Decision}` 是 detached source snapshot；`Detached()` 复制 action pointer 与 result-reference slice。verifier 返回 `HostResumeResult{Version, Status, Binding, Current, ReasonCode}`；Status 只接受 accepted/rejected，accepted 必须省略 ReasonCode，rejected 必须使用有限 `HostResumeReason`。runner 调用纯校验函数 `types.ValidateHostResumeResult`，accepted 标记本身不代表授权通过。

`HostResumeBinding` 保存暂停时的可信事实，`HostResumeFacts` 保存当前事实及 `DecisionAuthorized`、`ExecutionAuthorized`、`CancellationAuthorized`、`ConfirmationValid`、`EffectEligible`。字段名及 JSON tags 见 core/types；`HostResumePlan{ID, Version, Digest}` 的 Version 为正整数。callback 必须实际授权后设置布尔事实，不得仅复制历史 digest。Binding 的 checkpoint/action/correlation 必须匹配 source；CreatedAt、ResumeAttempt 等元数据不构成业务授权。

宿主在暂停时保存受信 execution binding，重验证时从 application execution record 读取该绑定与当前事实。source checkpoint snapshot 来自 runner，不能用客户端声明的 digest 替换。

| 绑定内容 | 责任 |
| --- | --- |
| Run/session、checkpoint ID/version/digest、mode、action reference | runner 验证真实 source identity，宿主绑定必须与它一致 |
| plan ID/version/digest | 宿主绑定原始执行计划；使用 planbridge 时复用 `Compiled.Identity()` |
| business revision、authority/tool/write-scope digests | 宿主读取当前事实并实际授权，不以 digest 自身作为权限 |
| confirmation/effect opaque references | 宿主验证确认和 effect eligibility；Baymax 不拥有业务账本 |

不使用 planbridge 的宿主提供自己的非空、版本化执行描述。reference 上限 256 UTF-8 bytes，digest 上限 128 bytes，新 request/result JSON envelope 上限各 32 KiB；版本必须匹配，引用不允许空值、非法 UTF-8、内部空白或控制字符。`HostResumeMaxReferenceBytes`、`HostResumeMaxDigestBytes`、`HostResumeMaxEnvelopeBytes` 固定这些上限。不得传递业务正文、原始 effect key、凭证或敏感确认记录；用 opaque references 关联宿主记录。该绑定不改变 planbridge 的 resumed unsupported 边界。

## Admission 顺序

1. runner 校验字段、完整 source identity、mode、duplicate 与 checkpoint state/expiry，取 detached snapshot/generation。
2. 在 source 锁外同步调用 verifier；宿主加载可信 binding 和当前权限/revision，返回明确的 accepted/rejected 及一致 identity。callback 不提交 effect，也不修改 source 状态。
3. callback 返回后，runner 在锁内再次检查 generation/state/mode/expiry/cancellation/context，只有仍有效的一个 decision 可以提交。
4. 提交后发一次 resolved，confirm 继续原生 loop；deny/timeout 交由原生 canceled terminal 路径。

confirm 要求当前 decision 和 execution/write authority，以及匹配的 revision/plan/tool/scope/confirmation/effect；deny/timeout 按宿主取消权限验证，不要求具备执行写权限。错误、panic、无效 result 或 context 在 commit 前取消都拒绝。callback 必须合作响应 context，不能保证强制中断任意不合作 Go 代码。

拒绝不消费 checkpoint/key，不推进 iteration、执行模型/工具/effect 或产生 resolved/terminal。宿主可以在权限再次有效且绑定仍匹配时重新申请；revision 真正变更时应建立新 plan/Run，不能通过改请求中的 digest 续跑旧 checkpoint。并发 cancel/expiry/replacement 必须在 source commit 前重检，不能让旧 snapshot 覆盖新的暂停或终态。

## Duplicate 与结果读取

公开 pause 结果的 PendingAction 与 source checkpoint 隔离；保留结果的 TerminalOutcome、Error、warnings、tool summaries 及结构化 error details 在存储和读取边界复制，调用方修改结果不会改写 source identity 或后续 duplicate 结果。

等价字段为 normalized Run/session、checkpoint ID/version/digest、token、decision kind/reference、mode、idempotency key。`DynamicActionDecision.Normalized()` 去除引用两端空白；decision kind 保持既有严格词汇。即使 guard 默认关闭，key/kind 相同而其他字段不同也不能得到 duplicate success；依赖旧宽松匹配的调用方需迁移为完整等价请求。

source 仍保留该 admission 的最终结果时，exact duplicate 返回结果和原 continuation error，且不再次调用 verifier/继续执行/发事件；首次 continuation 尚未完成时返回 `runner.ErrDynamicActionInProgress`（`dynamic action resume is in progress`），不返回空成功。结果读取权限仍由宿主独立验证，幂等键不是身份认证。只承诺当前 source retention 范围内的重复读取；下一次 PendingAction 替换 checkpoint 后，旧请求返回 stale/unknown，不建立无界历史缓存。尚未接收的 checkpoint 有 24h age 上限；已保留的完成 admission 可在 source retention 内 exact duplicate 读取。

## 恢复支持矩阵

已提交 decision 的 continuation 失败通过 `*types.DynamicActionContinuationError` 返回，`errors.As` 可区分准入后执行失败，`errors.Is`/`Unwrap` 可访问原执行原因；exact duplicate 保留同一执行原因。Host 对该首次恢复及 exact duplicate 仍返回 accepted / `dynamic_action.resume_admitted`，运行失败或取消由 source terminal 事件表达；provider/tool 错误正文不会被当作 admission ReasonCode。`types.NewDynamicActionContinuationError(nil)` 返回 nil，宿主自定义 resumer 可用该构造函数标识已接收后的执行失败。

| 入口 | Source 基线 | 本合同支持 |
| --- | --- | --- |
| 原 Engine 内 native dynamic action Run/Stream | 同原 mode continuation | opt-in 宿主重验证与并发/完整 duplicate 合同 |
| Host run.action.resume | 通用授权后委托 source，response/event barrier | 委托同一 guard，不新增 command kind |
| 新 Engine + checkpoint metadata/same run ID | 无 retained runner continuation | 明确 unavailable/unknown，不能执行模型/工具 |
| snapshot/event replay/metadata SPI | 引用、审计或各 owner 的既有能力 | 不等于恢复 runner 内部状态 |
| planbridge resumed、组合 team/workflow/mailbox dynamic continuation | plan bridge 首版 unsupported | 保持原边界，不开放组合恢复 |
| retry/follow-up/new Run | 各自既有入口 | 与消费暂停 checkpoint 的 resume 分开 |

## 业务提交、诊断与回滚

准入只验证一个时间点；宿主必须在实际 effect/commit gate 再次 CAS authority/revision/effect key。command accepted、resolved 或 runner completed 都不表示业务提交成功；本提案不保证跨系统原子 commit 或 exactly-once。

有界拒绝分类为 `host_resume.invalid_binding`、`binding_mismatch`、`authority_denied`、`revision_drift`、`plan_drift`、`scope_drift`、`confirmation_invalid`、`effect_conflict`、`revalidation_failed`（各项均使用 `host_resume.` 前缀），Go 中可用 `errors.Is(err, types.HostResumeRevisionDrift)` 等判断。callback error/panic 与 commit 前 context cancellation 收口为 revalidation_failed，不返回原始错误或 panic 内容。已有 source ErrDynamicActionUnknown/Stale/Expired/Terminal/Conflict 保持 source ownership；工具 capability 或写 scope 漂移共用 scope_drift。

guard 不产生额外 diagnostics/resolved/terminal；Host 将 bounded rejection reason 通过既有 admission response 返回。accepted confirm/deny/timeout 各一次 resolved，deny/timeout 随后一次 canceled finished；pause 的 input_required finished 不是 canceled terminal。既有事件的诊断记录仍经 RuntimeRecorder 单写；不增加 QueryRuns 字段，不复制正文或 raw callback errors，无 env/file/hot-update 键。

回滚停用新 guard，回到显式 source-native API；要求恢复重授权的宿主应停止接收无法重验证的恢复。保留 checkpoint/application/effect/outbox 事实；回滚无法撤销已发生写入。离线可运行示例见 [host-resume-admission](../../examples/host-resume-admission/README.md)，contract/replay/gate 映射见 [主线索引](../mainline-contract-test-index.md)。
