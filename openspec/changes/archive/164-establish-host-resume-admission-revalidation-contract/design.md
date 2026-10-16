## Context

动机见 proposal.md。审计基线是 `core/runner/dynamic_action_resume.go`、`core/types/dynamic_action_resume.go`、`host/host.go`、`host/action_resume.go` 与归档 162 的 planbridge Identity/Admission。runner map 持有真实 request/outcomes；metadata 的 RunCheckpointStore 接口不等于已接线的 durable continuation。Host 通用 Authorizer 看 command，不能代替源 checkpoint 与暂停时业务绑定的一致性校验。

## Goals / Non-Goals

**Goals:** 在一个原生 dynamic action resume 边界完成 source 验证、当前宿主重验证和单次 decision commit；固定完整 duplicate identity、拒绝可重试而不消费 checkpoint、Run/Stream parity 与恢复支持矩阵。Why now 是 plan/event identity 基线已交付，可用真实 runner 证明业务授权接缝。

**Non-Goals:** 通用执行图、业务授权引擎、effect store、统一恢复状态机、跨进程 runner restore、新 Host command、QueryRuns schema、全局队列、自动 retry、team/workflow/mailbox continuation 和 planbridge resumed 支持。业务确认记录的创建/提交不在 admission callback 内执行。

## Decisions

### 1. guard 放在 source runner 的提交边界

考虑了三种方式：仅使用通用 Host Authorizer（成本低，但直接 API 可绕开且不掌握真实 checkpoint）；在 runner 原生 admission 注入 opt-in guard（推荐，所有接入共享源检查）；建设统一 durable resume executor（范围过大，复制既有 owner）。

在 core/types 定义 `HostResumeRequest`、`HostResumeBinding`、`HostResumeFacts`、`HostResumeResult` 与 `HostResumeVerifier`；runner constructor 通过 `WithHostResumeVerifier(verifier)` 安装后覆盖该 Engine 的所有 dynamic action resume。宿主 callback 负责从 application execution record 读取可信绑定，并读取当前业务权限/revision。不增加默认跨模块依赖或独立 adapter store。`HostResumeVerifierFunc` 提供函数适配器，`ValidateHostResumeResult` 提供相同的纯确定性校验。

没有安装 guard 时保持已有合法 source 路径；安装后 nil/缺失 verifier、绑定缺失或回调错误 fail-closed。Host authorization/readiness 仍先运行，不被 guard 取代。已启用的 guard 不得按请求字段关闭。

### 2. reference-only 绑定与确定性校验

版本为 `host_resume_admission.v1`。可信暂停绑定至少含 source Run/session/checkpoint identity/version/digest/mode/action reference，以及宿主 plan ID/version/digest、business revision、authority digest、tool capability digest、write-scope digest、confirmation reference、effect key reference。无 planbridge 的直接宿主提供自己版本化的执行描述，不能使用空 identity；planbridge 宿主复用 Compiled.Identity，不改变其 resumed 不支持矩阵。

runner 给 verifier detached checkpoint 与 normalized decision，不信任 command payload 中声明的“当前权限”。宿主绑定必须由可信 callback 读取，绑定与 source checkpoint 一一匹配；重验证结果必须明确 accepted/rejected，并回显同一绑定 identity 与当前验证的 revision/digests。只有权限通过且绑定所期望的 revision/plan/scope/capability 仍一致才 accepted；replan 或扩大写权限不能通过更新 resume 请求续跑旧 checkpoint。digest 是一致性证据，callback 必须实际授权。

引用采用现有 dynamic action 的 256 UTF-8 bytes 上限，digest 为 128 bytes，版本/非空/类型/非法字符 fail-fast；独立新 DTO 的 serialized envelope 上限 32 KiB，plan version 必须大于零。示例只使用 synthetic references，实际 effect key 由宿主映射成 opaque reference。验证顺序固定为字段/版本/大小、source correlation/mode、已接收 decision 等价性、source state/expiry、宿主 binding、当前权限/revision、source commit 再检查；不转发 raw callback errors。

confirm/deny/timeout 都验证调用方的 decision 权限；confirm 同时要求当前执行写范围，deny/timeout 可按宿主取消权限接受，不要求拥有执行写权限。结果 reason 使用有限的 `host_resume.*` 分类（invalid_binding、binding_mismatch、authority_denied、revision_drift、plan_drift、scope_drift、confirmation_invalid、effect_conflict、revalidation_failed），源 stale/expired/terminal/conflict/unavailable 继续保留既有 source 分类。effect_conflict 表示宿主判断该 effect reference 不可用于该业务版本，Baymax 不维护 effect 幂等账本。

### 3. source 校验 → 锁外验证 → 原子再校验 → decision commit

runner 在既有 checkpoint 锁内读取 detached snapshot/generation，先拒绝无效请求；callback 在锁外同步运行，支持 cooperative context cancellation。回调不得修改 source 或提交业务 effect；panic、无效结果与 context 取消转换为有界拒绝。不用 goroutine 强制中断不合作回调，不在锁内调用外部代码。

返回后在源锁内对同一 checkpoint generation、state、mode、完整 decision identity、expiry、active control cancellation 再检查；context 已取消时不得 commit。该 commit 是 runtime admission 的唯一 linearization point。若取消已先提交，resume 拒绝；若 resume 已先提交，随后取消由既有 control/terminal owner 处理。只在提交后发一次 resolved；confirm 继续原生 loop，deny/timeout canceled，不建立 adapter terminal。

验证拒绝不消耗幂等键和 pending checkpoint，也不改变 active Run、iteration、tool results 或 terminal。允许通过宿主规则恢复有效授权后，对同一 checkpoint 再次合法申请；revision 真正变更时应由宿主建立新计划/新 Run，不能伪造旧版本。授权检查与业务 executor commit 不是跨系统原子事务；宿主仍必须在 effect 边界 CAS revision/authority/effect key。

### 4. 精确重复、并发与第二次暂停

完整 normalized equivalence 包含 Run/session、checkpoint ID/version/digest、token、decision kind/reference、mode 和 idempotency key。已经接收的相同 key 但不同 identity 不可返回成功 duplicate；无论 guard 是否启用都 fail-closed，这是对既有 contract 的修正。

同一 checkpoint 的最终结果已保留时，exact duplicate 返回该结果，不再次执行 verifier/resolution/terminal/effect；结果读取/调用方权限仍由 Host authorization 或直接调用方验证。首次 resume 正在执行时 duplicate 返回稳定 in-progress classification，不能返回空 result 伪装完成，也不能再次继续；宿主可查询或稍后重复读取。其他 key/decision 为 conflict。两个回调并发验证可各自读取事实，但锁内只允许一个 source commit；输家重新分类，不二次发事件。

若 continuation 产生新的 PendingAction，新 checkpoint 是不同 generation；checkpoint ID 为不透明的 `dynamic-action:<generation>`，调用方不得解析或自行重建。旧 decision/key 不能覆盖或恢复新 checkpoint。首阶段只承诺仍由 source 保留的 checkpoint admission/result 的 duplicate（包含 continuation error），不新建无界历史缓存，旧 checkpoint 被替换后返回 stale/unknown。测试通过 barrier 控制竞态，不依赖 sleep。

### 5. support matrix 与观测

| Path | Scope |
| --- | --- |
| 原 Engine 内 direct Run / Stream dynamic action | 支持启用 guard 的同 mode 恢复 |
| Host run.action.resume | 经通用授权后委托同一 source guard；既有 response/event barrier |
| 新 Engine + 原 checkpoint metadata | 明确拒绝 unavailable/unknown，不执行模型/工具 |
| metadata SPI / snapshot / event replay | 仅引用/审计，不能恢复 request/outcomes/control |
| planbridge resumed、team/workflow/mailbox continuation | 保留既有 unsupported；不扩大执行范围 |
| retry / follow-up new run | 使用既有入口，不能消费 dynamic checkpoint |

只保留有界 reason/opaque references。可用既有 host admission/source fact events 表达 guard rejection，新增 payload 须 additive/nullable/default 且经过 RuntimeRecorder；不创建新 resolution/terminal 类型或 QueryRuns 字段。replay 是纯离线 normalized facts，不调用 verifier、provider、tool 或 store。

## Risks / Trade-offs

- [授权检查后业务变化] → 运行准入仅是一个时间点；业务 executor/commit gate 必须再次 CAS，测试明确不宣称 exactly-once。
- [callback 阻塞或重入] → 锁外同步调用、ctx 合作、panic 收口，不承诺强制中断；重入造成 generation 变化必须拒绝旧请求。
- [读 duplicate 泄漏旧结果] → 完整 identity 不构成身份认证；宿主必须独立授权结果读取。
- [单次 confirm 后产生第二次 pause] → generation + checkpoint identity 隔离，不将最新 checkpoint 的结果写回旧 continuation。
- [加强 duplicate 等价性影响旧调用方] → 文档提供迁移要求：重复请求必须保持所有 normalized 字段与 mode 不变。

## Migration Plan

先固定现状 gap 与 example 文档，新增 DTO/guard 后交付最小 source 接线，再完成真实 runner/Host parity、replay、gate、公开文档和完整门禁。默认不启用业务 guard；需要该保证的宿主必须显式启用并传入可信绑定读取/当前重验证 callback，不能把历史 digest 当 accepted 结果。

回滚见 proposal：停用新接缝或停止恢复准入，不删除已有 checkpoint/application/effect/outbox；已有写入不能由 runtime rollback 撤销。实施 verification.md 必须区分执行了的 shell/PowerShell 检查与环境不支持项。

## Example Impact Assessment

新增示例

先完成 `examples/host-resume-admission/README.md` 基线，再写离线真实 runner 场景；不修改 agent-modes，不创建新业务数据库。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`CHANGELOG.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/host-resume-admission.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`examples/host-resume-admission/README.md`
- 维护 owner：runner、host integration 与 documentation maintainers
- 验证证据：`verification.md` 记录真实 runner Run/Stream、Host integration、并发重验证、fixture/replay、双平台专用 gate、OpenSpec strict validation、documentation/example/roadmap/docs 与完整 Go/race/lint/quality 门禁的实际命令、退出结果和环境限制

“已同步”指同一功能分支中的维护文档已按实际实现更新：`runner.WithHostResumeVerifier`、`types.HostResumeVerifier`/函数适配器、reference-only DTO、默认关闭、256-byte reference/128-byte digest/32-KiB envelope、有限拒绝分类、完整 duplicate identity 和 same-Engine 恢复矩阵。后续公开语义变化仍须同步维护文档及三份工件；任务勾选与归档以 `verification.md` 的实际通过证据为准。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runner/host maintainers | guard 不拥有执行/业务状态，callback 锁外运行，RuntimeRecorder 单写 |
| components | 新增文档 | `docs/guides/host-resume-admission.md` | runner/host maintainers | checkpoint owner、绑定接缝、并发重检和支持矩阵对照实际 API |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | constructor-only opt-in，不新增 env/file/hot-update 键，不改变配置优先级 |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md`、`docs/guides/host-resume-admission.md` | contract maintainers | delta requirement 与正负边界/integration/replay/gate 对照，规划与已交付分开 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | bounded reason/reference，不记录业务正文，单写入口，无新增 QueryRuns 字段 |
| examples | 新增文档 | `examples/host-resume-admission/README.md` | examples maintainers | 文档先行后已实现真实 runner 示例，覆盖合法恢复/revision drift/duplicate/effect CAS markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md`、`docs/guides/host-resume-admission.md` | host integration maintainers | 既有 action-resume command 接线，不增加 command kind 或 wire vocabulary |
| best practices | 新增文档 | `docs/guides/host-resume-admission.md` | documentation maintainers | digest 不等于授权，恢复不是 retry/replay，effect 提交仍须宿主 CAS |
| roadmap | 修改文档 | `docs/development-roadmap.md`、`README.md` | roadmap maintainers | archive 162 状态修正，active slug 与 openspec list 一致 |
