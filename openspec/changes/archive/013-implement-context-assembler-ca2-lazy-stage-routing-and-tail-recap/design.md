## Context

当前仓库已完成 Context Assembler CA1：pre-model hook、prefix hash、一致性 fail-fast、append-only journal 与最小诊断字段。CA2 目标是在不改变现有 runner/tool/stream 对外语义的前提下，引入按需加载与阶段路由能力，使 context 组装可在“低成本路径优先”下演进到可接入 RAG/DB 的扩展架构。

约束来自已确认项：
- 提案名固定：`implement-context-assembler-ca2-lazy-stage-routing-and-tail-recap`
- stage 失败策略必须可配置
- 本期只实现本地文件 provider，RAG/DB 仅接口暴露
- tail recap 先落最小字段（status/decisions/todo/risks）
- 路由先采用规则判定，并预留 agentic 判定扩展 TODO
- 允许新增诊断枚举字段
- 文档与实现必须同步
- examples 本期不实现，仅文档 TODO

## Goals / Non-Goals

**Goals:**
- 在 `context/assembler` 内实现 CA2 双阶段装配：Stage1 + Stage2。
- 实现规则化路由：Stage1 满足即跳过 Stage2，不满足再触发 Stage2。
- 为 Stage2 定义 provider 接口并落地本地文件 provider。
- 输出 tail recap 稳定块（status/decisions/todo/risks）并追加到末尾。
- 在 runtime config 中提供 CA2 可配置项（开关/超时/策略/阈值）。
- 在 diagnostics 中增加 stage/recap 相关枚举与字段。
- 保持现有质量门禁与 Run/Stream 语义兼容。

**Non-Goals:**
- 不实现真实 RAG 检索链路或向量数据库存储。
- 不实现 CA3 的 memory pressure（squash/prune/spill-swap）。
- 不引入 HITL 状态机改造。
- 不对外暴露 tool-call argument fragments（保持 complete-only）。

## Decisions

### Decision 1: CA2 仍以内嵌 pre-model hook 方式交付
- Choice: 继续复用 `core/runner -> context/assembler` 的前置调用点，不引入新 runtime 或独立 orchestrator。
- Rationale: 变更域最小，兼容性风险最低，且便于沿用 CA1 观测与 fail-fast 语义。
- Alternative: 独立 context service。Rejected：架构跨度过大，不符合本期收敛目标。

### Decision 2: 双阶段模型固定为 Stage1 -> Stage2
- Choice: Stage1 执行基础上下文组装；仅当规则引擎判定“不满足”时触发 Stage2 provider。
- Rationale: 对齐 P3/P4（按需加载、渐进降级），减少不必要 I/O 与调用成本。
- Alternative: 每次都执行 Stage2。Rejected：违背 lazy allocation 目标。

### Decision 3: 路由采用规则引擎并预留 agentic hook
- Choice: 引入 deterministic 规则判定（关键字段缺失、上下文阈值、触发词），并在接口层预留 `DecisionProvider` 扩展点，标注 TODO。
- Rationale: 可测试、可解释、可快速交付；同时保留后续 agentic 决策演进路径。
- Alternative: 直接用模型判定路由。Rejected：增加不确定性与测试成本，不适合当前阶段。

### Decision 4: Stage2 provider 接口先标准化，默认 file 实现
- Choice: 定义 provider 接口（`Fetch(ctx, req) -> chunks/meta`），实现 `file` provider；`rag`/`db` provider 返回 not-ready。
- Rationale: 满足“本期可用 + 后续可扩展”的双目标。
- Alternative: 仅硬编码文件读取逻辑。Rejected：后续接入成本高。

### Decision 5: tail recap 使用最小固定 schema
- Choice: recap 块固定字段 `status/decisions/todo/risks`，顺序稳定，追加在 assembled context 尾部。
- Rationale: 先保证稳定可观测，再逐步扩展字段。
- Alternative: 自由结构 recap。Rejected：难以做契约测试与前后版本兼容。

### Decision 6: stage 失败策略可配置
- Choice: 提供 stage 级策略配置（fail-fast / best-effort），默认策略在配置层定义并可覆盖。
- Rationale: 兼顾严格模式与可用性模式，适配不同运行环境。
- Alternative: 全局单策略。Rejected：粒度不足。

### Decision 7: diagnostics 扩展枚举并沿用 single-writer/idempotency
- Choice: 新增 stage/recap 字段（如 `assemble_stage_status`、`stage2_skip_reason`、`recap_status`），仍通过 `run.finished` 路径进入 recorder/store。
- Rationale: 保持现有诊断架构一致，降低数据管道变更风险。
- Alternative: 新增独立事件表。Rejected：引入额外一致性成本。

## Risks / Trade-offs

- [Risk] 路由规则过于保守可能导致 Stage2 触发不足。 -> Mitigation: 暴露阈值配置 + 诊断字段记录 skip reason。
- [Risk] best-effort 模式掩盖潜在数据质量问题。 -> Mitigation: 明确 `recap_status` 与 `assemble_stage_status`，在诊断中可见降级轨迹。
- [Risk] provider 接口先行可能带来抽象过度。 -> Mitigation: 本期只保留最小方法集，避免提前设计复杂生命周期。
- [Risk] recap 末尾追加可能影响 prompt token 成本。 -> Mitigation: 约束 recap 字段长度并支持配置关闭。

## Migration Plan

1. 扩展 `context_assembler` 配置 schema（CA2 段、stage 策略、provider 选择、路由阈值）。
2. 引入 Stage2 provider 接口与 file provider；rag/db 返回占位错误。
3. 实现 Stage1/Stage2 路由与 tail recap 组装逻辑。
4. 将 CA2 结果映射到 diagnostics 字段（沿用 single-writer）。
5. 增加单元/集成回归与 race/lint 门禁。
6. 同步 README/docs/roadmap/phased-plan/v1-acceptance，补 examples TODO。

Rollback strategy:
- 将 `context_assembler.ca2.enabled=false` 回退到 CA1 行为。

## Open Questions

- agentic routing hook 在 CA2 仅预留接口，具体启用策略在后续提案确定。
- Stage2 file provider 的数据目录约定是否需要支持多租户命名空间，可在实现期按现有项目结构做最小化约定。
