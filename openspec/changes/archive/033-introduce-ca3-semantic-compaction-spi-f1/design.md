## Context

当前 CA3 在压力区间触发 `squash/prune/spill`，具备较强工程稳定性，但 `squash` 主要为内容截断，`prune` 以关键词和访问频次评分为主，缺少“语义保真”能力。用户已确认本期需要提供可执行的 semantic 策略，并允许直接复用现有 LLM client 来完成压缩。

与此同时，系统要求保持：
- 默认行为稳定（默认 `truncate`）；
- `Run/Stream` 语义一致；
- `fail_fast/best_effort` 语义不漂移；
- 文档与实现一致，并对后续 TODO 留痕到 roadmap。

## Goals / Non-Goals

**Goals:**
- 在 CA3 引入 compaction SPI，并支持 `truncate|semantic` 策略选择。
- `semantic` 策略首期可执行，调用当前 LLM client 进行压缩重写。
- 加入“证据最小集保留”规则，避免关键上下文被 prune 删除。
- 增加 compaction 诊断字段并保证 `Run/Stream` 语义一致。
- 严格执行 `best_effort` 回退与 `fail_fast` 终止策略。

**Non-Goals:**
- 不引入 embedding/re-ranker/vector DB 生命周期管理。
- 不改 CA2 路由、external retriever 协议或 action timeline 结构。
- 不新增 CLI 入口，仅通过库能力与配置控制。

## Decisions

### Decision 1: 引入包内 Compactor SPI，不对外暴露
- 方案：在 `context/assembler` 内定义 `Compactor` 接口与策略路由器。
- 理由：满足内部复用与职责隔离，同时控制 API 面稳定性。
- 备选：直接在 `squashMessages` 中硬编码分支。
- 放弃原因：扩展性差，后续接入更多策略会放大维护成本。

### Decision 2: semantic 策略直接复用现有 LLM client
- 方案：通过当前 runner/model 已选中的 client 通道执行语义压缩请求。
- 理由：减少新依赖和重复抽象，迁移成本低，符合当前项目定位。
- 备选：新增独立 summarizer provider。
- 放弃原因：引入额外配置面与生命周期管理，超出本期 scope。

### Decision 3: 默认策略保持 truncate，semantic 通过配置启用
- 方案：新增 `context_assembler.ca3.compaction.mode`，默认 `truncate`。
- 理由：保守上线，避免默认行为回归。
- 备选：默认启用 semantic。
- 放弃原因：成本与不确定性高，难以保证初期稳定性。

### Decision 4: 语义失败处理沿用 fail-fast/best-effort 既有语义
- 方案：
  - `best_effort`: semantic 失败回退 truncate，并记录 `ca3_compaction_fallback=true`。
  - `fail_fast`: semantic 失败立即返回错误终止。
- 理由：与现有 stage 策略语义一致，可预测且易验证。

### Decision 5: 证据保留规则采用“关键词 + 最近窗口”最小实现
- 方案：在 prune 候选筛选阶段引入 evidence guard，优先保护关键内容。
- 理由：先收敛最小可用方案，后续可平滑扩展到更复杂判定。

## Risks / Trade-offs

- [风险] semantic 调用带来额外延迟
  - Mitigation: 仅在 warning/danger/emergency 触发；支持超时与回退；以诊断字段量化影响。
- [风险] semantic 输出不稳定导致压缩质量波动
  - Mitigation: 保留 truncate 默认策略；关键证据保护；文档中明确 TODO 与后续质量治理里程碑。
- [风险] Run/Stream 路径实现分叉导致语义漂移
  - Mitigation: 增加主干契约测试覆盖同输入对齐。

## Migration Plan

1. 增加配置字段并设置默认 `truncate`，保持向后兼容。
2. 引入 Compactor SPI 与策略路由，保留现有 truncate 逻辑实现。
3. 落地 semantic compactor（基于现有 LLM client）并接入超时与错误分类。
4. 接入 evidence retention 规则与诊断字段。
5. 增加 Run/Stream 契约测试与回归测试。
6. 同步 README/docs，并将“semantic 质量增强”TODO 写入 roadmap。

## Open Questions

- semantic prompt 模板是否需要后续抽离为可配置模板（本期先固定，roadmap 留 TODO）。
- 是否在下一期引入 semantic 质量评分（例如保留率/覆盖率）作为 gate（本期不做）。
