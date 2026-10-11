## Context

动机见 proposal.md。现有事实源与接缝如下：

| 现状 | 设计约束 |
| --- | --- |
| `types.EventHandler.OnEvent` 无 error 返回；Run/Stream 都接收 handler | 新接入显式暴露投影/交付失败，不能依赖返回 RunResult 推断观察成功 |
| `MapEventToProtocol` 已提供 source、Run/Step、Time、Kind、Payload | 复用该 mapper，新增 wrapper 保存 native type/version/iteration 和 trace/span；不另造协议生命周期 |
| mapper 优先将 TraceID 用作 EventID，fallback hash 不包含全部 payload；payload 浅拷贝 | mapped ID 只是 mapping reference，不能承诺唯一或直接去重；接入输出必须拥有不可被源后续修改的有界快照 |
| durable binding 当前只接收 Realtime，sequence/dedup/cursor 属于该 source | direct callback 保持 `sequence=0`，不能编造 cursor/history 或执行恢复 |
| Host 已有 serialized delivery、deadline、source-declared drop 规则 | 本 adapter 是无需 Host wire protocol 的嵌入入口；不替换 Connection 的交付实现 |
| dynamic action pause 发出 `run.finished(state=input_required)` | 事件名不是 terminal 判据；resolution、pause 与 terminal 维持归档 158/159 语义 |

## Goals / Non-Goals

目标是让宿主把真实执行事件接到自己的记录入口，并能判断这个入口提供了什么语义、哪些事件没有成功交付。结构保持 library-first：`source event -> opt-in adapter -> host-owned execution sink -> application record/projection`；RuntimeRecorder 作为原始事件的独立消费者接入。

本 change 不创建业务记录数据库、AG-UI/SSE transport、订阅服务、durable runner log、全局 event sequence、去重缓存、恢复控制或授权模型。不实施后续 plan、resume revalidation、context 或 outbox 候选。

## Decisions

### 1. 采用薄 adapter，而非扩展 Host transport 或新建 event runtime

三个路径比较：仅写接线示例成本最低，但没有可复用失败反馈、bounded reference 和能力声明；扩展 Host JSONL 能覆盖连接管理，但强制本地应用引入命令/传输层；新增薄 adapter 可以直接实现 EventHandler 并复用协议类型，保持 application sink 由宿主注入。本设计选择第三种。

公共 DTO/纯验证与投影位于 `core/types/native_execution_event.go`，版本为 `native_execution_event.v1`；事件回调与 sink glue 位于 `host/eventadapter`。包仅依赖 core/types 与标准库，不导入业务应用、Provider SDK 或 runtime diagnostics store。第一阶段不改 `host.Connection` 和 `MapEventToProtocol` 的既有行为。

NativeExecutionEvent 包含：版本、既有 EventEnvelope、native event type/schema version、iteration、trace/span references、source capability profile 和独立 HostReferences。source 未提供的 causation/parent/session 不凭空生成；已提供的 source references 在 payload 或既有 envelope 中保留。构造参数中的 Session/expected Run 只能按已授权 source binding 校验/补缺，冲突即拒绝。

### 2. source capability 必须反映事实

Direct callback profile 冻结在 adapter 构造时，不接受宿主把不支持的功能自行标为 supported。它声明 `callback_order`，identity 为 `mapping_reference`，durable history/cursor、dedup 和 gap verification 为 unsupported；未知 parent/causation 显式为 absent。支持 runner、workflow、teams、scheduler、composer、A2A 和 timeline 的原生 Event 映射，但每个 source 接线须声明真实 owner，不能通过 event 名称猜 source。

顺序只保留同一串行 source callback stream 的调用顺序。并发 child/source 分别接独立 observer，宿主通过现有 parent-child references 关联；adapter 不把时间戳、trace/span 或到达顺序转换为全局因果顺序。一个 observer 同时收到重入或并发回调时 fail-fast 为 observation failure，不用无界排队或锁等待掩盖冲突。

Realtime 不经 raw callback 伪装身份；纯投影入口接收现有 source-normalized EventStreamBindingProjection，复用现有 mapper、序列/去重/gap/retention 校验并保留 source cursor outcome。该入口不提供 subscription worker、history 存储或跨批去重缓存，能力范围仍为 Realtime。unsupported durable 请求在执行/订阅开始前明确失败，不降级到 latest。

### 3. 保留每个 direct callback，不根据 mapped ID 去重

两个 native 事件共享 TraceID 时，mapper 会产生相同 EventID；fallback hash 也不等价于 source-owned identity。wrapper 保留现有 ID 及 trace facts并声明身份来源限制，同样 ID 的两个回调仍交付两次；不把内容 hash 或 adapter 递增计数冒充 source ID。重复效果的判断留给 source，而应用自己的持久化 receipt key 属于应用存储语义，不能回写到 runtime。

测试必须展示不同 native type、同一 TraceID 的事件都保留，以及相同 native 事件两次回调均保留。Realtime 的已归一化重复 overlap 按原 source contract 处理。修复共享 mapper ID 策略属于另一项已确认 gap 的增量变更，本 change 不扩大 wire contract。

### 4. host references 与 payload 有界且隔离

HostReferences 只允许 task ID、schedule occurrence reference、plan version/digest、projection digest、knowledge generation reference 和 citation reference 列表；无任意业务正文或权限赋值。引用不覆盖 source Run/Session/Step/Event、parent/causation、terminal outcome 或 action decision。禁止未知 key、错误类型、非法 UTF-8、负/无限 limits 和超限字段；拒绝时返回稳定 reason，source 与输入参数不变。

默认限制：scalar reference 每项 1024 UTF-8 bytes，citation refs 最多 16 项，全部 host references JSON 最多 8 KiB，整条 projected event JSON 最多 64 KiB。允许构造时显式设置有限正值，不引入 env/file/hot-update 参数。source payload 超限或不可安全序列化时报告投影失败，不能静默裁掉 terminal/identity 字段。

输出是拥有独立内存的 JSON-compatible 快照，含嵌套 map/list 与 references；源在回调返回后或宿主在 sink 返回后修改自己的对象，不得污染其他消费者。source 在同步捕获期间不得并发修改同一 payload；该前置条件写入 handler contract。adapter 不将事件正文写入 diagnostics，现场 sink 的业务数据访问由宿主授权，离线 fixture 仅包含 bounded fake facts。

有界预检最多 128 层，拒绝循环/不可序列化值；禁止调用自定义 JSON/Text marshaler，标准 `time.Time` 除外。数字保持 `json.Number` 精度。Realtime 空批次的 subscription/outcome 也受单条 event byte 限制和 UTF-8 校验；batch 数量仍由既有 source binding 的 MaxBatchSize 校验拥有。

### 5. 有界同步交付与独立观察失败

Sink 提供 context-aware 的 error-returning `Consume(ctx, event)` 合同。adapter 同步调用，无内部 async queue、每事件 goroutine 或自动 retry；默认 delivery timeout 为 1 秒，可通过构造参数设为有限正值。sink 必须响应 deadline、不能重入同一 observer；不合作的任意 Go 回调无法由库强制中断，文档必须明确这一限制，禁止用 detached goroutine 假装有界。

delivery deadline 基于 observer context，不能仅用已取消的业务 Run context 导致 canceled terminal 无法交付；observer 生命周期独立于 Run。mapper/validation/sink error、timeout、并发重入、observer close 后写入均产生稳定 observer reason。adapter 保留固定大小的 Status snapshot：状态、首个失败 reason 和 event reference、交付计数；不保存事件正文、无界错误列表或重试队列。

首个失败后该 observer 停止向 sink 交付，后续回调仅计入有界状态，不声称 complete delivery。sink 返回错误/超时可能发生在持久化之后，此时 delivery 为 unconfirmed，不能宣称未写入或重试。调用方在 Run/Stream 返回及 resume 返回后读取 Status；Observer Close 幂等，仅停止观察，不调用 Engine.Cancel/Resume/Retry。Realtime source-declared drop 继续由现有 Host/source 路径处理，本 callback adapter 不新增 drop 模式。

RuntimeRecorder 接线使用显式 EventHandler fan-out，adapter 的 observer failure 不反馈回自身形成事件循环。Status 是集成错误反馈，不是诊断写入口；如宿主需要记录，使用现有标准 host delivery fact，经 RuntimeRecorder 的单写入口，不新增 QueryRuns schema。

### 6. terminal 与业务完成严格依赖 source facts

adapter 不把最终 RunResult 变成第二条 parent/terminal event，不用事件名、error nil 或 sink ack 推断业务完成。`input_required` 是暂停事实，`run.dynamic_action.resolved` 表示 decision 被接收；只有 source 的 completed/failed/canceled outcome 才是对应 Run 的 terminal。confirm 不产生 canceled，deny/timeout 透传 source canceled，同 idempotency 的 native resume 不再发 resolution，adapter不补发。

缺失 terminal 是 observation incomplete，不合成 completed；已有 terminal conflict 的投影复用现有 owner/arbiter，不在 adapter 留一个 terminal FSM。source terminal event 在无法交付时通过 Status 显示 failure，即使 source Run 已正常完成。提交业务结果仍取决于宿主 commit gate。

## Risks / Trade-offs

- [仅有 callback_order，不能建立全局 timeline] → source/run 标识和 causation 按事实保留，fixture 展示 child 交错，durable runner replay 留给条件观察项。
- [TraceID/reference 被误用为唯一 ID] → capabilities、示例和同 ID 不同事件反例共同约束，无 direct dedup。
- [sink 阻塞或失败后内容已写入] → context-aware 合同与有限 deadline，状态为 unconfirmed，无自动重试/补发。
- [observer context 与 Run context 混用] → observer 独立，取消测试仍收到 source canceled；显式 stop 只改变观察状态。
- [快照复制和 size check 有开销] → 有限 event bytes、基准测量与超限拒绝，不拷贝业务历史或 retention。
- [host references 越权、payload 泄漏] → 固定 reference allowlist、授权 sink、正文不进入新增 diagnostics，fixture 使用 fake data。

## Migration Plan

1. 实施前把代码审计边界固定为测试；文档先明确能力、limits、sink 合同和 observer/source 责任。
2. 新增 opt-in 类型、纯 mapper 与薄 adapter，旧调用路径继续存在；不替换 Host delivery。
3. 接入方把 Run/Stream 的 nil handler 替换为 adapter，并与 RuntimeRecorder 独立 fan-out；resume 使用同样 source correlation 和明确的新观察 session，不跨进程假设缓存存在。
4. 上线前核对 Status 与真实事件覆盖；业务执行记录的存储、receipt identity、授权与 AG-UI/SSE projection 由宿主实现。
5. 回滚关闭新增 adapter，恢复此前显式 handler 路径；不删除 application record、不改 source Run 状态，不回滚外部副作用。

## Validation

真实 runner + capability adapter fake model/tool 验证 completed、工具失败/继续、model failure、cancel/timeout、dynamic action confirm/deny/timeout/duplicate、parent-child 与 Run/Stream parity。Run/Stream 只归一化允许差异：随机 run/call/trace ID、时间、model delta 数量/分片；不能隐藏生命周期、resolution cardinality、source correlation、错误分类或交付失败。

版本化 `native_execution_event.v1` replay fixture 覆盖合法映射、unsupported cursor、reference/size/JSON violation、identity collision 不丢事件、pause 非 terminal、observer failure、source mismatch、Realtime overlap/gap/expired cursor 与 parity drift。离线 replay 不执行 sink、tool、Provider、网络或业务写入。专用 gate 两平台同语义，接入 quality gate，contract index 列出 positive/negative/replay/ownership 证据；example smoke 使用真实 runner，无 live provider。

Parity 在忽略生成时间前验证 direct mapped ID；只归一化 schema-owned envelope/payload/terminal_outcome/checkpoint 的明确生成引用和时间，不改嵌套业务正文或 HostReferences。只移除归属已观察 model step 的已知 fragment-only delta 表示（text/tool-call/argument fragment）；未知、error、clarification、meta 等 delta 必须保留。此比较证明执行生命周期与观察语义等价，不替代模型内容/工具参数的业务验证。

## Example Impact Assessment

新增示例

新增 `examples/native-execution-events`，文档给出 source path、能力矩阵、expected markers、交付失败检查和 rollback。当前不修改 agent-mode；若改变范围，先完成其 MATRIX/README 文档基线。

## Documentation Impact Assessment

对外语义与维护文档必须在同一变更中收敛。新增或改变公开 API/contract、运行时行为、事件顺序/identity/终态、错误与失败反馈、配置默认值/校验或诊断字段时，先更新本设计、spec delta 与影响评估，再同步对应维护文档、代码和测试；additive 和 opt-in 入口也需要说明。维护文档必须反映最终语义、默认值与限制、兼容性、迁移和回滚，并通过 docs consistency、相关 contract/replay gate 与示例核对。文档未同步或验证未通过，对应任务不得勾选，变更不得视为可交付或可归档。

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/architecture/data-event-flow.md`, `docs/runtime-module-boundaries.md` | runtime/host owners | source-owner 边界评审与 docs consistency |
| components | 修改文档 | `core/types/README.md`, `observability/README.md`, 新增 `host/eventadapter/README.md` | core/types 与 host owners | API/example 对照和 package tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config owner | 明确仅构造参数、无 env/file/hot-update 新键；docs gate |
| contract/API | 修改文档 | 新 capability spec, `docs/mainline-contract-test-index.md` | contract owner | OpenSpec strict validation 与 contract/replay gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `observability/README.md` | observability owner | 失败结果和 recorder 单写接线测试；无 QueryRuns 新字段 |
| examples | 新增文档 | `examples/native-execution-events/README.md`, `README.md` 示例入口 | examples owner | 离线 example smoke 与 expected markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md`, `host/eventadapter/README.md` | host/integration owner | handler/sink/failure/capability 接入测试 |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation owner | 禁止 fabricated terminal、TraceID 去重与无限背压；docs gate |
| roadmap | 修改文档 | `docs/development-roadmap.md`, `README.md` | roadmap owner | openspec list 与 roadmap status consistency |
