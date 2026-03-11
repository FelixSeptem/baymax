## Context

当前 Baymax 已完成并发/异步能力与基础示例扩容，但 MCP 层仍存在 `http` 与 `stdio` 代码路径重复、策略配置分散、行为语义轻微漂移的问题。R1 路线图中“统一 MCP 配置对象与默认值文档化”仍是剩余项，且 v1 限制明确指出 MCP heartbeat/reconnect 仍为 best effort。

## Goals / Non-Goals

**Goals:**
- 为 MCP 层建立统一可靠性 profile 体系，明确各 profile 的目标与默认值。
- 收敛 `http/stdio` 的重试、backoff、重连和事件归一化逻辑到共享组件。
- 建立 MCP 故障注入测试矩阵，验证 profile 行为与恢复能力。
- 增加运行诊断摘要，支持“最近 N 次调用”快速定位。
- 同步修正文档状态漂移，保证 README/docs 与实际归档状态一致。

**Non-Goals:**
- 不引入新的 MCP 传输协议类型。
- 不扩展到分布式调度或多租户治理。
- 不在本提案内交付 R3 高阶示例本体。

## Decisions

### Decision 1: 采用 profile-first 配置模型
- Choice: 通过命名 profile 固化常用部署策略，并允许局部覆写。
- Rationale: 降低调参复杂度，减少“参数组合不可控”问题。
- Alternatives:
  - 仅暴露散装参数：灵活但难以形成稳定运行基线。

### Decision 2: 引入 MCP shared runtime 组件
- Choice: 将重试/重连/backoff/事件归一化下沉到共享层，`http/stdio` 仅保留传输差异。
- Rationale: 去重并确保行为一致性。
- Alternatives:
  - 两端各自维护：重复代码和行为漂移风险持续存在。

### Decision 3: 使用故障注入验证 profile 行为
- Choice: 在 integration 中加入 heartbeat timeout、reconnect storm、queue backpressure 等场景。
- Rationale: 可靠性能力必须通过可复现故障测试而非仅代码审查。
- Alternatives:
  - 依赖线上反馈：回归发现滞后且风险高。

### Decision 4: 诊断输出最小化但可行动
- Choice: 先输出最近 N 次调用摘要（耗时、重试、错误分类、重连次数、profile）。
- Rationale: 提供直接排障价值，避免一次性引入复杂控制面。
- Alternatives:
  - 全量控制台/仪表盘：实现成本高，不适合作为该提案范围。

## Risks / Trade-offs

- [共享层抽象不当导致传输特性被过度约束] → 仅收敛通用策略，保留 transport-specific hook。
- [profile 默认值不合理引发线上回归] → 配置发布前运行故障注入与 benchmark 对比。
- [诊断字段过多引起噪声] → 首版仅保留排障高价值字段并文档化解释。

## Migration Plan

1. 定义 profile 结构与默认值，文档先行。
2. 抽取共享 runtime 组件并迁移 `mcp/http`。
3. 迁移 `mcp/stdio` 并对齐事件与错误分类。
4. 增加故障注入测试与回归基准。
5. 添加诊断摘要接口/命令并更新 docs。

## Open Questions

- profile 是否支持运行时热切换，或仅在启动时固定。
- “最近 N 次调用摘要”落地形式（API、CLI、日志导出）优先级排序。
