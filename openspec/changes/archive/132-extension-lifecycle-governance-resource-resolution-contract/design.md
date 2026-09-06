## Context

Baymax 的扩展相关能力分布在 Skill loader、adapter manifest/capability、hooks/middleware、sandbox/allowlist、readiness 和 RuntimeRecorder 中。当前这些能力可以分别完成加载、校验或执行，但缺少统一的扩展资源身份、来源优先级、能力准入、生命周期和 reload 隔离模型。`earendil-works/pi` 提供了可参考的分层：资源管理负责发现与重载，扩展加载器负责实例化，运行器负责事件分发与绑定；本设计将其转化为 Go、library-first、contract-first 的 Baymax 投影。

约束：配置遵循 `env > file > default`；非法启动配置和热更新必须 fail-fast + 原子回滚；诊断只能通过 `observability/event.RuntimeRecorder` 单写；新增 QueryRuns/诊断字段保持 additive + nullable + default；Run/Stream 必须语义等价；不建设 hosted control plane 或第二事实源。

## Goals / Non-Goals

**Goals:**

- 为扩展资源建立确定性的 discovery、precedence、dedupe 和冲突结果。
- 为扩展建立 manifest/digest/version compatibility 与 requested-vs-declared capability 合同。
- 在激活前统一经过 policy/readiness admission，并输出稳定 denial/degraded finding。
- 为 Hook/工具扩展建立 bounded timeout、资源上限、阻断、失败隔离和 finalize 语义。
- 为 reload 建立 generation/instance 边界，确保旧实例不再接收新事件，运行中 Run 的事实不被改写。
- 通过 RuntimeRecorder 记录加载、拒绝、激活、阻断、失败、回滚和恢复，并可由 replay fixture 重建决策。

**Non-Goals:**

- 不引入 npm、git package manager、托管扩展市场或远程 registry。
- 不实现进程外沙箱、独立扩展进程、session server、Artifact content service 或第二套 Session/Run 状态机。
- 不把扩展声明的 action availability 当作授权；授权仍由既有 policy/sandbox/admission 所有。
- 不改写既有 Tool/MCP、Realtime、Snapshot、Recovery 的 source-of-truth 语义。

## Decisions

### 1. 采用 manifest-first、source-scoped 的资源模型

每个候选资源先形成内部 `ResourceDescriptor`，包含稳定名称、kind、source scope、origin、path/reference、version、digest、declared capabilities 和 precedence rank。来源至少区分 project-explicit、project-auto、user-explicit、user-auto 和 package-like local bundle；排序和去重必须 deterministic。选择该模型是为了让资源解析与执行解耦，并使拒绝原因可回放。

备选方案：直接按目录扫描顺序加载，简单但无法稳定解释覆盖关系；完全复制 pi 的 npm/git package manager，能力更全但违反 Baymax 的 library-first 和供应链边界。

### 2. capability 协商复用 adapter 现有语义

扩展请求能力集合与 manifest 声明能力集合先经过 `adapter/capability` 的 required/optional、`fail_fast|best_effort` 和 canonical reason taxonomy；descriptor 只描述可用动作，不授予权限。这样可以避免 Skill、Adapter、Extension 之间出现三套 capability 语义。

备选方案：为 Extension 创建独立 capability registry，短期清晰但会造成 taxonomy、replay 和 gate 漂移。

### 3. 激活采用 readiness/admission 两阶段

先做纯函数 manifest/schema/version/digest 校验，再执行 policy/readiness admission。`blocked` 不激活；`degraded` 只在既有策略允许时激活并记录 finding；`ready` 才进入运行器。所有阶段输出结构化 finding 和 stable denial reason。

备选方案：加载后再发现问题，用户体验简单但会让非法扩展部分生效；仅返回 Go error，则无法区分拒绝、降级和运行失败，也不利于 replay。

### 4. 生命周期以 generation + save point 管理

每次资源重载生成新的 activation generation。旧 generation 立即标记 stale，不再接收新事件；已经开始的动作按既有 Run 生命周期完成或取消，不被 reload 隐式重试。扩展产生的持久化 mutation 只能在明确 save point 提交，运行中读取使用 turn snapshot。该设计吸收 pi AgentHarness 的阶段/保存点思想，同时不引入新的 Session 状态机。

### 5. 失败隔离保持“扩展局部化、Run 语义不变”

扩展 Hook panic、超时、资源超限、返回非法结果或 finalize 失败，统一映射为扩展域 failure event；根据 policy 选择 skip/deny/degrade，但不得覆盖业务 Run 的权威终态。RuntimeRecorder 记录 extension id、generation、phase、reason、timeout/resource metadata，并限制高基数字段。

备选方案：扩展失败直接终止 Run，语义容易理解但会把可选扩展故障放大为核心运行故障；吞掉所有错误则无法安全治理。

### 6. 诊断与回放采用 additive projection

新增扩展诊断字段遵循 additive + nullable + default，事件仍由 RuntimeRecorder 单写。replay fixture 输入 manifest、资源来源、配置快照、请求能力和 admission 结果，验证排序、拒绝、激活、reload generation 和失败分类稳定。Run 与 Stream 共用同一 resolver/admission/runner 决策函数。

## Risks / Trade-offs

- [进程内扩展仍可能影响宿主] → 首期只承诺局部失败隔离和资源/超时边界；真正进程外沙箱另列长期方向。
- [来源优先级过于复杂] → 固化有限 source scope、排序表和冲突分类，所有新增来源必须补 replay/gate。
- [manifest 与真实行为漂移] → 激活前做 declared/requested capability 校验，运行期记录实际 phase/failure，conformance gate 阻断 drift。
- [reload 与并发 Run 交叉] → generation + stale 标记；不回收正在执行动作的事实，不允许旧实例发新事件。
- [诊断字段导致 cardinality 增长] → 复用 diagnostics cardinality/truncation governance，路径、digest 和扩展名采用 bounded projection。
- [宿主需要更强交互能力] → 保留 UI request/response 的嵌入式接口预留，但不在本变更引入 RPC 或 hosted control plane。

## Migration Plan

1. 先实现 resolver/manifest/capability/admission 的纯函数合同和 replay fixtures，不改变默认运行路径。
2. 以 feature gate 默认关闭接入扩展激活与 reload；无扩展或旧资源路径保持现有行为。
3. 接入 RuntimeRecorder additive diagnostics、Run/Stream parity 和 quality gate。
4. 在完成 conformance、故障注入和文档/示例基线后，按宿主需求逐步开启。
5. 回滚时关闭 feature gate，保留只读诊断与 fixture；不得删除已写入的兼容字段。

## Open Questions

- 首期是否只覆盖 Skill/Hook，还是同时覆盖外部 Adapter manifest？建议合同先覆盖统一 descriptor，运行时接线按 capability 分阶段。
- 是否需要为 extension resource 引入签名校验？建议首期只固化 digest 和来源记录，签名/信任根另行评审。
- 是否需要提供宿主自定义 precedence？建议首期固定系统排序，避免配置覆盖规则本身成为新的治理面。
