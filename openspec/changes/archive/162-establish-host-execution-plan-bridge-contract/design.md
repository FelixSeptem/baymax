## Context

动机见 [proposal](proposal.md)。本 change 在原生事件 adapter（归档 160）与宿主恢复准入候选之间建立计划接缝；`orchestration/planbridge` 的公开 opt-in API 已按本文实现。以下审计基于 Baymax，Danny 的观察来自集成方报告。

| 既有入口 | 已有责任 | 接入限制 |
| --- | --- | --- |
| composer.Run/Stream | readiness guard、runner 委托 | 不解释通用计划图；调用 composer 不会自动运行 workflow/teams |
| workflow.ValidateDefinition / Run / Stream | DAG、依赖、step timeout/retry、checkpoint | 首版只桥接 flat workflow；不重新解释子图或建立新的 checkpoint store |
| teams.Run/Stream / normalizePlan | task validation/defaults、parallel/vote、aggregation | normalizePlan 未公开；需要先证明纯验证接缝缺失，再最小暴露原 validator |
| scheduler.Guardrails / composer.SpawnChild | depth/active children、原生 timeout resolver 与 parent clamp | 编译事实不能替代实际 activation 时的 parent budget |
| composer.DispatchChild | enqueue、queue claim、local execution、terminal commit | claim 面向整个队列；不承诺任意共享队列的 task-specific dispatch |
| host/eventadapter | 真实 source callbacks 与独立 host references | 不控制执行，不提供通用 durable cursor/resume |

## Goals / Non-Goals

Goals：将宿主描述和原生 binding 的不一致变成运行前可测试的错误；编译整份计划后才委托 source owner；以原生结果与真实事件证明 Run/Stream 对等。边界必须足够明确，使后续恢复准入引用 plan version/digest，而不推断旧业务权限仍有效。

Non-Goals：迁移 TaskProfile、权限、业务 revision/effect key 或 commit gate；通用 ExecutionGraph/new executor；bridge 自己做 retry、budget ledger、parent terminal reduction、checkpoint persistence、mailbox worker 或后台 fanout；新增 runtime config、Host wire command、QueryRuns 字段；迁移 cron/DST/outbox 或 Provider 协议。

## Decisions

### 1. 描述、编译快照、原生 binding 分离

新增 opt-in `orchestration/planbridge`。公开入口固定为 `Describe(Descriptor, Binding, Limits)`、`Compile(Descriptor, Binding, Limits)`、`Compiled.Run/Stream(ctx, Execution)`、`Identity()` 与 `Canonical()`。Describe 生成 reference-only SourceFacts，宿主可独立保存并由 Compile 对照另一次 binding；Describe 不证明既有宿主计划正确。Binding 必须提供 live Admit 与 SemanticDigest。编译只有验证、规范化和快照，不调用模型/tool/handler、enqueue、publish、claim、commit 或写诊断。

Descriptor 使用 `host_execution_plan.v1`，不包含任意业务正文或 Go callback。公开字段分为：

| 字段组 | 合同 |
| --- | --- |
| identity | schema_version、plan_id、plan_version；version 为正整数，replan 使用新版本 |
| axes | shape: direct/subagent/team/workflow；delivery: inline/mailbox；trigger: manual/scheduled/resumed/callback |
| host evidence | authority_digest、tool_capability_digest 必填；input_ref/input_digest、handler_ref/handler_digest 按 binding 要求填入 |
| source correlation | run/parent/workflow/team/step/task references 按实际 owner 路径声明；与 binding 一致，不自动生成 parent |
| structure | workflow step/dependency refs，team task/strategy/aggregation refs，或 child request facts；只能存在与 shape 对应的一种结构 |
| budgets | 所需的 native timeout/profile/retry/depth/active-child/parent-remaining facts；字段适用性由 owner 路径决定 |
| continuation | 可选 checkpoint reference 与显式 capability requirements；未声明支持的能力必须拒绝 |

Binding 单独提供 runner request、teams.Plan、workflow.Definition/adapter、composer child request，以及宿主已授权的 input/callback。其结构事实必须与描述逐项一致：ID、dependency、task order、strategy、source timeout/retry/guardrail facts、parent refs 不能仅依靠宿主声称的 digest 绕过检查。不把业务 input body、callback closure 或 credentials 塞进 descriptor、错误或 replay。

需要可复制的输入使用受支持的 bounded JSON-like snapshot；拒绝 cyclic、pointer-rich、unsupported values，不用任意 MarshalJSON 方法绕过预算。编译持有 detached maps/slices/native definitions。函数、engine/registry/store 等服务引用无法深拷贝，其受信边界与禁止并发修改责任须在 package README 明示；不宣称哈希验证了 callback 实现。

选择这层薄桥接，是因为已有 owner 能执行相应结构。仅补文档不能验证 binding 与 plan 漂移；新建通用 ExecutionGraph 则会引入新的拓扑/调度/终态责任，当前没有独立宿主 fixture 支持这一扩张。

### 2. 有界 schema 与 canonical digest

首版默认上限：descriptor canonical bytes 64 KiB、合计 structural items 128、dependencies 1024、单 reference UTF-8 bytes 1024、JSON-like snapshot nesting depth 8。constructor 可显式收紧至正值；零选择默认，负值或放宽默认上限拒绝。binding snapshot 也使用独立 64 KiB 内容预算与相同深度限制，不能用“descriptor 只有 refs”绕过复制成本。不能在预算检查前无界 marshal/clone。超限错误返回 field path 与 limit，不回显正文。

先验证类型/计数/深度/引用，再复用 source normalization，然后 canonicalize。canonical fields 包含 schema、identity、axes、host digests/refs、实际原生结构、source-normalized 可执行默认值和编译时 budget facts；按字段固定顺序编码、map keys 排序、整数 duration 单位固定为纳秒。保留 task/step 和 dependency list 的原顺序，不以排序改变 source tie-break 或执行意义。absent/zero 的等价性只采用 source normalization 已证明的规则。digest 为 `sha256:<lowercase hex>`，不包含 digest 自身。

Golden vectors 固定 canonical bytes 和 digest；map insertion order 不影响 digest，语义顺序、版本、权限/tool digest、结构或预算变化必须影响 digest。input/handler digest 是宿主声明的一致性证据，不认证代码或授权内容。compiler 不维护全局 revision store；阻止旧版本业务提交仍由宿主负责。

### 3. 支持矩阵先于执行

| 路径 | 首版编译/执行支持 | 约束或拒绝证据 |
| --- | --- | --- |
| direct + inline | Run/Stream | 真实 runner 或 composer runner 委托；原生 input_required 返回并保持 pause |
| team parallel/vote + inline | Run/Stream | 原生 aggregation；terminal-only task binding，显式 pause/continuation requirements 拒绝 |
| flat workflow + inline | Run/Stream | 原生 validator/dispatch adapter；首版不桥接 subgraph、动态 action continuation |
| local subagent + inline | 限定的原生 child dispatch | 仅隔离、独占 scheduler binding；terminal-only local handler；不从 shared queue 抢任务 |
| subagent fanout | 经既有 teams/workflow 表达 | bridge 不新建 fanout loop；所有 callbacks 先完成 binding 验证 |
| 任意 mailbox delivery | 不支持 | 在 enqueue/publish 前返回 unsupported capability；不把 accepted 当 completed |
| resumed trigger / native checkpoint resume | 不支持执行 | checkpoint reference 可作为 opaque fact 校验；不能仅复用 run_id 打开新 Run |
| 跨 owner durable continuation | 不支持 | 不把 snapshot、event replay 或引用相同 checkpoint 当恢复证明 |
| manual/scheduled/callback trigger + 已支持 inline shape | 支持来源标签 | 不创建 cron/定时器/callback worker，也不提供额外授权 |

local subagent 的 Run/Stream 若都复用同步 DispatchChild，其 Stream 只转发已有 source timeline，不能宣称 LocalChildRunner 本身提供 token streaming。实施 fixture 必须证明完整 scheduler isolation、claim/commit、实际 callbacks 和对等 source outcome；缺少可验证隔离 binding 或缺少 source callback 时必须拒绝相应请求。不要通过清空用户队列、反复 claim、bridge 锁/重试循环来制造隔离。

teams/workflow callback 的 pause 语义并未由本提案扩展。宿主须显式提供 terminal-only binding；需要暂停的 callback 不能接入此路径。若封装 runner 结果，应在 source adapter 边界检测 input_required 并明确返回 unsupported continuation，保留 checkpoint/error facts，不把它包装成成功 task。源 owner 已执行的副作用不会因该运行时错误回滚。只有 direct 路径原样承接已支持的 runner pause；resume 继续使用既有显式 API，并等待后续宿主重验证提案。

### 4. 纯全计划预检与 source 再准入

编译流程：bounded shape/reference validation → support matrix → full native binding validation/default normalization → canonical snapshot/digest。任何 step/task 无效，都在首次 source activation 前失败。错误族建议固定为 `unsupported_version`、`invalid_plan`、`limit_exceeded`、`binding_mismatch`、`unsupported_capability`；字段 reference/path bounded，原生错误可 unwrap，不回显 payload。

workflow 新增纯 `Engine.PreflightDefinition`，复用 normalizeDefinition、ValidateDefinition 与 owner defaultStepTimeout；teams 暴露调用原有 normalizePlan 的纯 `NormalizePlan`；scheduler 暴露纯 `NormalizeTask`，并复用 ValidateSpawn 与 timeout resolver。gap fixtures 已先证明原接口不能无副作用地固定完整默认值，新增入口不执行 callback 或写 store；不复制规则。ChildBinding 不接受 scheduler/store/composer，每次 activation 创建私有 memory scheduler 并只 DispatchChild 一次。新的 source API 保持既有运行语义；如 fixture 要求改变既有行为，先补相应 spec delta 和文档，不能藏在 bridge。

Compile 不保留“永久获准”token。激活前校验配置/能力前提是否变化；影响编译语义时 fail-fast 要求重新编译，不能静默改 digest。动态余额、租约、readiness 与宿主权限由 source/宿主在实际 activation boundary 重检。parent budget 使用当前实际余额与 source timeout resolver 的 clamp，预算耗尽拒绝；compile 时完整的预算快照只说明描述成立，不是可花费余额。

bridge 不缓存宿主授权结论、不绕过 action gate/readiness、不收取 token、不重试失败 source call。新的业务权限或 revision 是否有效，宿主需在调用前验证；后续 resume contract 不由此提前实现。

### 5. 委托结果和事件保真

编译快照选择唯一 source owner，Run/Stream 各委托一次对应原生路径。结果使用 owner-tagged native result，加 plan identity/version/digest references；保留原生 error、failed/canceled/timeout、task/attempt、checkpoint 与 aggregation。不同 owner 不共享新的 terminal enum/reducer。暂停、accepted、scheduler terminal 与业务提交不能相互推导。

source EventHandler/timeline 由真实 owner 产生，沿用归档 160 observer 支持矩阵；如多个 source 并行，分别建 observer，以 parent/causation refs 关联。PlanVersion/PlanDigest 放入既有 HostReferences，PlanID 由宿主 execution record 关联，不为此修改 NativeExecutionEvent schema。Run 与 Stream 比较 normalized native outcome/parent refs/预算事实，允许真实 source callback 数量和 stream 分片有差异。既有 source callback 错误/observer 错误分别遵循 source/adapter 合同，bridge 不新增 fail-open/fail-closed 决策。

不从 RunResult 构造 parent/terminal 事件，不从 QueryRuns 推导实时 timeline，不声称 global sequence/durable cursor；RuntimeRecorder 仍唯一诊断写入入口。

### 6. 验证与文档交付

单测覆盖所有错误族、默认值与边界、digest vectors、input mutation、opaque binding 限制、完整预检 no side effect；integration 使用真实 runner/teams/workflow/scheduler 和 fake model/tool，不以 spy-only owner 代替行为证据。至少证明 direct Run/Stream（含原生 pause）、teams parallel/vote、workflow 依赖/失败/retry/timeout、隔离 child parent clamp/claim/commit；共享队列与 unsupported pause/resume/mailbox 为拒绝证据。

新增版本化 reference-only fixture/replay 与 `scripts/check-host-execution-plan-contract.sh/.ps1`，接入 quality gate；Run/Stream normalized drift、缺少 source-owner evidence、side-effect-before-rejection 和 digest drift 均须失败。replay 只校验离线证据，不执行 handler或恢复 Run。主线测试索引仅在 suite 真正接入 gate 后添加。

先维护 `examples/host-execution-plan/README.md`，声明 semantic anchor、runtime path、支持矩阵、expected markers、rollback。实施后交付离线可运行代码，且同步 package README、integration/best-practices、配置/诊断限制和契约索引。三份工件接受同一文档同步承诺；每个新增维护路径都进入三份一致的承诺。没有实际测试、文档和 gate 证据不得勾选任务。

## Risks / Trade-offs

实施验证中复现 shell documentation-impact gate 的长 changed-path 管道 SIGPIPE：grep -q 提前退出使 pipefail 误判 README 未同步。作为本 change 必需治理接线修复，改为消费完整输入，并用大型 untracked fixture 回归；不删除宿主缓存，不放宽文档同步要求。

- [描述与可执行 binding 脱节] → full native fact comparison、detached snapshots；opaque callback 的宿主责任明示。
- [复制原生验证/defaults 或 stale config] → 单一 source validator，golden fixtures，activation 漂移拒绝重新编译。
- [隔离前提失效、误领 shared queue task] → 专用独占 binding；缺少证据先拒绝，不借用 bridge 循环补偿。
- [child pause 被聚合成成功] → 明确 terminal-only 能力限制与 adapter 负向测试；direct 保留 pause。
- [支持矩阵较保守] → unsupported 是首版合同；后续仅凭真实 gap 增量扩展，同步 spec/docs/tests。
- [digest 被误用为权限或 durable resume] → 反例与文档，source live admission 和宿主业务 commit 独立。

## Migration Plan

显式添加 planbridge，不改旧 owner 默认入口；宿主先离线 Compile 与 golden fixture，再 opt-in 支持矩阵内的执行。源码、测试、文档与 spec delta 同一功能分支交付。停用 bridge 回到原 owner API；保留 task/attempt/checkpoint/outbox/业务记录，不回滚真实 effects。归档前完整门禁，使用归档脚本；只有归档和验证后再按已授权流程合并，不在提案阶段宣称交付。

## Example Impact Assessment

新增示例

本批已交付宿主中立的离线 fake model/tool example 与 smoke；不改变 agent-modes 示例。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/components/orchestration.md`、`orchestration/README.md`、`orchestration/planbridge/README.md`、`docs/runtime-module-boundaries.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`examples/host-execution-plan/README.md`
- 维护 owner：orchestration 与 host integration maintainers
- 验证证据：OpenSpec strict validation、documentation/example impact gate、roadmap/docs consistency；实施后补齐 plan bridge contract/replay、Run/Stream parity 与离线 example smoke

当前“已同步”指本 change 已在同一功能分支交付并同步的对外 API、limits/defaults、错误分类、支持矩阵、owner 边界、兼容性和回滚说明。新增维护路径已纳入三份工件的一致承诺；缺少实际测试或门禁 exit-code 证据时，仍不得勾选任务、宣称验证完成或归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | orchestration maintainers | owner 边界审计与 docs consistency；禁止第二执行/预算/恢复状态机 |
| components | 修改文档 | `docs/components/orchestration.md`、`orchestration/README.md`；实施新增 `orchestration/planbridge/README.md` | orchestration maintainers | 支持矩阵与实际 owner API 对照、package tests |
| configuration | 修改文档 | 实施同步 `docs/runtime-config-diagnostics.md` | runtime config maintainers | 明确 constructor-only limits、既有配置优先级和无新增 env/file/hot-update 键 |
| contract/API | 修改文档 | 本 change spec；实施同步 `docs/mainline-contract-test-index.md` | contract maintainers | strict validation、逐 requirement 的单测/integration/replay/gate 证据；gate 接入后才更新主线索引 |
| diagnostics | 修改文档 | 实施同步 `docs/runtime-config-diagnostics.md` | observability maintainers | reference-only 错误/结果与 RuntimeRecorder 单写；不新增 QueryRuns 字段 |
| examples | 新增文档 | `examples/host-execution-plan/README.md`、`README.md` | examples maintainers | 文档基线先行，离线 smoke 与 expected markers；不依赖 live provider |
| CLI/integration | 修改文档 | `docs/guides/integration.md`；实施新增 `orchestration/planbridge/README.md` | host integration maintainers | plan/binding 编译、执行前拒绝、源结果透传与回滚示例 |
| best practices | 修改文档 | 实施同步 `docs/guides/best-practices.md` | documentation maintainers | digest 不等于授权、投递不等于完成、replay 不等于 resume 的反例核对 |
| roadmap | 修改文档 | `docs/development-roadmap.md`、`README.md` | roadmap maintainers | openspec list 与 roadmap status consistency；候选移入进行中 |
