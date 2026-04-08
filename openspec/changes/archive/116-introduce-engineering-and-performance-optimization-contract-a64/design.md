## Context

A64 作为 A63 之后的下一提案，目标不是引入新行为，而是在既有 contract 语义稳定前提下清理性能热点与工程噪声。当前热点集中在：
- 高频 payload/map/JSON 构建（runner、recorder、provider stream/event）；
- 查询路径全量扫描/排序/聚合（diagnostics、task board、memory query）；
- file backend 高频全量持久化（scheduler/mailbox/recovery）；
- 调用链重复包装与短生命周期对象分配（mcp invoke、skill loader/tokenization）；
- 配置读路径与 policy resolve 的重复值拷贝；
- 事件管线串行 fanout 与逐事件导出。
- `runtime.eval.*` 与运行态质量信号尚未形成 inferential feedback 的受控 advisory 闭环；
- realtime interrupt/resume cursor 与 isolate-handoff 关键状态在崩溃恢复场景下缺少统一 snapshot 扩展收口；
- snapshot 缺少 retention/quota/cleanup 熵预算治理，长跑与高频恢复场景存在容量漂移风险。

同时，仓库存在临时备份工件回流风险（如 `*.go.<digits>`），会放大评审噪声并干扰门禁稳定性。

## Goals / Non-Goals

**Goals:**
- 在不改变 Run/Stream、reason taxonomy、diagnostics/replay 语义前提下，完成 S1~S10 主干优化收口。
- 统一“优化实施三件套”：基线、阈值、语义稳定证明。
- 为每个 S 子项提供模块级 benchmark 与 impacted contract suites。
- 将工程治理缺口（未跟踪临时工件回流）纳入阻断门禁。
- 补齐 inferential feedback advisory、interaction-state recovery、snapshot entropy budget 三类治理缺口并保持默认语义不变。
- 产出 harnessability scorecard 机器可读报告并纳入质量门禁阻断。
- 补齐 multi-agent 涌现行为治理与 drift 分类阻断，避免并发级联错误在优化过程中隐式放大。
- 补齐 harness ROI 与动态深度治理，确保 planner/evaluator/sensor 开销不超过收益并可按复杂度调节。
- 固化 harness 可测试性分层：objective 领域由 computational sensors 阻断，inferential sensors 仅作为主观补充。
- 所有优化具备开关与回滚路径，且热更新失败保持原子回滚。

**Non-Goals:**
- 不引入平台化控制面、外置必选依赖或第二套事实源。
- 不借“性能优化”重定义既有 contract 字段或业务决策语义。
- 不在 A64 之外拆分平行性能提案。
- 不在本提案内处理 A63 命名治理主责范围（A63 已在实施）。

## Decisions

### Decision 1: S1~S10 分片治理，单提案内闭环

- 方案：所有优化必须映射到既有 S1~S10；新增热点只能作为对应 S 的增量任务吸收。
- 原因：避免并行提案造成门禁散落、基准口径分叉和追踪困难。
- 备选：按模块拆多个性能提案并行推进。
- 取舍：并行提案灵活但会削弱语义稳定证明的一致性。

### Decision 2: 语义稳定优先于吞吐提升

- 方案：先定义 `semantic-stability` 阻断，再允许接入性能优化；任意语义漂移直接阻断。
- 原因：当前主线为 contract-first，A64 不承担行为重构职责。
- 备选：先做性能改造，再补语义回归。
- 取舍：先性能后语义会提高回归与回滚成本。

### Decision 3: 查询与聚合统一采用“锁内快照、锁外计算”

- 方案：查询路径在锁内仅做最小快照与游标状态读取，过滤/排序/聚合在锁外执行。
- 原因：降低长时间锁占用，提升并发读写稳定性。
- 备选：维持全流程在锁内执行。
- 取舍：锁外计算需严格保证快照一致性与结果稳定排序。

### Decision 4: file backend 采用“可开关批写 + flush 边界”

- 方案：在不改变默认强一致语义下，增加 debounce/group-commit 可选路径，并显式定义 flush/崩溃恢复边界。
- 原因：降低 `marshal + tmp + rename` 高频 I/O 放大。
- 备选：保留每次变更立即全量持久化。
- 取舍：批写路径必须以恢复一致性测试覆盖，不允许隐式语义漂移。

### Decision 5: 事件与导出管线采用批量化与隔离策略

- 方案：exporter 引入 `max_batch_size + max_flush_latency`，dispatcher 增加可配置 fanout/隔离。
- 原因：避免逐事件导出与慢 handler 放大主链路延迟。
- 备选：保持同步串行 fanout 与逐事件导出。
- 取舍：批量策略需额外验证顺序与失败语义保持不变。

### Decision 6: 配置与策略读路径采用快照引用 + 可失效缓存

- 方案：runtime config 提供只读快照引用，policy resolve 采用 `profile+override` 签名缓存并在 reload 原子失效。
- 原因：减少热路径值拷贝与重复 resolve。
- 备选：持续使用值拷贝 + 每次全量解析。
- 取舍：缓存必须严格绑定版本号，避免陈旧策略。

### Decision 7: 工程治理纳入 A64 强门禁

- 方案：repo hygiene 扩展到已跟踪 + 未跟踪临时工件，A64 benchmark 需模块级独立入口。
- 原因：避免脏工件与超大 benchmark 文件侵蚀工程质量。
- 备选：仅靠人工评审发现噪声文件。
- 取舍：自动阻断更严格，但需要维护例外策略最小化。

### Decision 8: inferential feedback 仅作为 advisory 通道接入

- 方案：将 `runtime.eval.*` 与运行态质量信号接入 S2/S9，可观测输出 advisory，不直接改写 readiness/admission 的既有 deny 决策。
- 原因：A64 目标是性能与工程优化，不承担业务裁决语义重定义。
- 备选：让 inferential feedback 直接参与 admission 决策。
- 取舍：advisory 更稳健，避免在优化提案中引入策略语义漂移。

### Decision 9: interaction-state recovery 复用 unified snapshot 合同扩展

- 方案：realtime cursor 与 isolate-handoff 关键状态通过 state/session snapshot 合同段扩展治理，不引入第二套事实源。
- 原因：与 A66/A67-CTX/A68 既有合同一致，降低恢复路径分叉。
- 备选：新增独立持久化平面存放 interaction-state。
- 取舍：复用既有合同可降低一致性与回放复杂度，但需补齐扩展段的兼容窗口测试。

### Decision 10: snapshot entropy budget 采用“默认不变 + 可开关治理”

- 方案：新增 `retention/quota/cleanup` 治理参数并执行 fail-fast + 热更新原子回滚；默认配置保持现有行为。
- 原因：控制长期容量漂移，同时避免默认行为突变。
- 备选：直接启用强制清理策略作为新默认。
- 取舍：默认不变更保守，治理能力通过显式开关逐步启用。

### Decision 11: harnessability scorecard 纳入 gate 阻断

- 方案：新增 `check-a64-harnessability-scorecard.*`，生成机器可读 scorecard（contract 覆盖、drift 统计、gate 覆盖、docs consistency）并阈值阻断。
- 原因：将“可驾驭性”从经验判断转为可审计指标。
- 备选：仅在 PR 说明中人工汇总，不接入阻断。
- 取舍：阻断更严格但可持续，需维护指标口径稳定与阈值治理流程。

### Decision 12: multi-agent 涌现行为采用“矩阵回归 + 漂移分类”治理

- 方案：对 S3/S7/S10 等并发敏感子项补齐 multi-agent 场景矩阵（并行、交错、重试、重放），并将涌现行为漂移纳入阻断分类。
- 原因：单 agent 用例无法覆盖并发涌现风险，优化改动容易放大级联错误。
- 备选：继续复用现有 shared contract 套件，不新增并发矩阵。
- 取舍：矩阵成本更高，但可显著降低“多 agent 场景仅在线上暴露”的回归风险。

### Decision 13: harness 深度采用“ROI 预算 + 动态分层”治理

- 方案：为 planner/evaluator/sensor/garbage-collection 建立 token/latency/quality 三维 ROI 指标；当开销超阈值时降级到轻量 depth，避免过度工程。
- 原因：harness 本身会引入额外成本，必须可度量、可调节、可回滚。
- 备选：统一固定深度策略，不做复杂度分层。
- 取舍：固定策略实现简单但容易在低复杂任务中过拟合开销；ROI+分层更稳健但需要基线维护。

### Decision 14: harness 可测试性采用“computational-first, inferential-second”分层

- 方案：客观 correctness 路径（contract/replay/schema/taxonomy）必须由 computational sensors 阻断；inferential sensors 仅在主观质量域补充，不得替代基础阻断。
- 原因：避免“用不可靠信号验证不可靠信号”的循环，降低误判与不可复现性。
- 备选：将 inferential 检查与 computational 检查等权混用。
- 取舍：分层策略更可审计、更可复现，但需要明确定义主观/客观边界。

### Decision 15: 既有性能回归门禁纳入 A64 性能门禁主链

- 方案：将 `check-context-production-hardening-benchmark-regression.*`、`check-diagnostics-query-performance-regression.*`、`check-multi-agent-performance-regression.*` 作为 `check-a64-performance-regression.*` 的强制子门禁，按 S1/S2/S3/S7 热点映射执行。
- 原因：这些门禁已经承载主干历史基线，若不并入 A64 会出现“优化任务在 A64，回归阻断在提案外”的治理裂缝。
- 备选：保持旧门禁独立执行，不纳入 A64 主链。
- 取舍：并入主链可提升审计闭环，但需要维护 A64 与主干门禁索引同步。

### Decision 16: 门禁执行采用“影响面映射 + 耗时预算”双治理

- 方案：新增 changed-files -> impacted suites 映射校验，支持 `fast/full` 两档执行；同时输出 gate step 级耗时指标并按预算阈值阻断退化。
- 原因：A64 子项持续扩张后，若仅靠全量门禁会显著拉长反馈周期，影响迭代效率；但直接裁剪门禁又有漏检风险。
- 备选：始终全量执行或仅人工选择子门禁。
- 取舍：影响面映射可提升效率，但必须保留 mandatory suites 的硬约束，并通过耗时预算治理防止“门禁慢性膨胀”。

## Risks / Trade-offs

- [Risk] 多模块并行优化导致回归定位复杂。
  -> Mitigation: 严格按 S1~S10 分批合入，子项前置 benchmark 与 impacted suites。

- [Risk] 锁外计算可能引入顺序/一致性偏差。
  -> Mitigation: 保持稳定排序规则不变，增加结果等价断言与 replay 回归。

- [Risk] 批写/批量导出在低流量场景出现“延迟滞留”。
  -> Mitigation: 强制 `max_flush_latency` 上限并覆盖低频流量测试。

- [Risk] 缓存策略引入陈旧配置读取。
  -> Mitigation: 绑定 reload 版本号，热更新后强制失效并回归验证。

- [Risk] 优化触碰主干 contract 字段导致隐式漂移。
  -> Mitigation: `check-a64-semantic-stability-contract.*` 作为阻断前置，不提供漂移豁免。

- [Risk] inferential feedback advisory 被误用为隐式决策输入。
  -> Mitigation: 在 contract/gate 中固定“advisory only”约束，并增加 decision-trace 等价断言。

- [Risk] interaction-state 恢复扩展引入恢复不一致或回放漂移。
  -> Mitigation: 增加 crash/restart/replay 组合回归与 mixed-fixture 兼容断言。

- [Risk] snapshot entropy 治理参数误配导致过度清理。
  -> Mitigation: fail-fast 校验 + 热更新回滚 + 默认关闭/默认不变策略。

- [Risk] scorecard 指标口径漂移导致误阻断。
  -> Mitigation: 固定指标定义与 baseline 更新流程，阈值调整需附带证据。

- [Risk] multi-agent 涨并发后出现新型涌现行为，现有单链路回归无法捕获。
  -> Mitigation: 为并发敏感模块引入 multi-agent matrix suites 与 drift taxonomy，作为阻断前置。

- [Risk] harness 深度策略过重，产生 token/latency 开销反噬收益。
  -> Mitigation: 为每类 depth 固化 ROI 阈值与降级策略，低复杂任务默认走轻量路径。

- [Risk] inferential 评估被误当作客观阻断依据，导致不稳定误判。
  -> Mitigation: 在 gate 中固定 computational-first 分层，主观域 inferential 结论需附结构化证据。

- [Risk] A64 门禁与既有性能门禁各自演进，导致阈值或执行入口漂移。
  -> Mitigation: 强制将 context/diagnostics/multi-agent 三类性能门禁作为 A64 性能主门禁子步骤并在 impacted suites 中显式映射。

- [Risk] changed-files 映射误配导致应跑未跑。
  -> Mitigation: `check-a64-impacted-gate-selection.*` 对 mandatory suites 执行完备性做硬校验，缺项直接阻断。

- [Risk] 门禁耗时阈值设置不合理导致误阻断。
  -> Mitigation: 采用 baseline + 审批更新流程，并在报告中输出 step 级耗时证据用于审计。
