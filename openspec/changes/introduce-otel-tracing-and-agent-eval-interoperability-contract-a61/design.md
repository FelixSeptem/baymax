## Context

A55 已提供 `runtime.observability.export.*` 与 diagnostics bundle 基线，仓库也已有 `observability/trace` 轻量封装和 `RuntimeRecorder` 单写入口。但在实际接入中仍有两个缺口：

1. OTel 采集接入缺统一 contract：runtime 到 collector 的 span 拓扑、attribute 映射、失败语义和诊断字段未冻结，导致不同后端解释不一致；
2. agent eval 缺统一执行治理：local/distributed 执行、分片汇总、失败重试与断点续跑没有稳定 contract，回归口径难以持续阻断。

同时，A60 正在实施预算 admission，A61 需要与 A58/A59/A60 字段体系协同，避免语义分叉。A61 的设计必须满足：
- `library-first`：不引入托管控制面；
- `contract-first`：配置、观测、回放、门禁一体冻结；
- `Run/Stream` 等价；
- `RuntimeRecorder` 单写入口与 additive 字段兼容。

## Goals / Non-Goals

**Goals:**
- 目标一：实现“顺滑接入 OTel 采集”的 contract，固定 collector 互操作行为；
- 目标二：完整兼顾原目标，冻结 agent eval 互操作与 `local|distributed` 执行治理；
- 冻结 tracing/eval 配置域、诊断字段、replay fixtures、contract gate；
- 保持 Run/Stream 语义等价与字段可回放；
- 通过独立 required-check 候选实现持续阻断。

**Non-Goals:**
- 不引入托管 tracing control plane、托管 eval 调度服务、平台化 UI/RBAC/多租户运维面板；
- 不重定义 A58/A59/A60 已冻结解释字段；
- 不在 A61 内推进 A64 性能专项或 A62 示例收口；
- 不新增平行“评测执行平台”提案（A61 内一次收口）。

## Decisions

### Decision 1: A61 采用“双目标同提案收口”，并以 OTel 采集接入为主落点

- 方案：A61 在一个 contract 内同时冻结 tracing interop 与 eval interop，主落点放在 OTel 采集接线顺滑。
- 备选：先单独做 tracing，再另开 eval proposal。
- 取舍：并行拆分会带来字段和门禁重复建设，且与 roadmap 的“一次性收口”约束冲突。

### Decision 2: tracing 配置采用独立 `runtime.observability.tracing.otel.*`，并与现有 export 基线兼容

- 方案：引入 tracing 专用配置域（endpoint/protocol/sample_ratio/batch/export timeout/resource attrs）。
- 兼容策略：当 tracing endpoint 未显式设置时，可复用既有 `runtime.observability.export.profile=otlp` 的 endpoint 作为默认来源，降低接线成本。
- 备选：完全复用 `runtime.observability.export.*` 不新增 tracing 配置域。
- 取舍：完全复用会混淆“事件导出”和“trace 导出”语义；独立配置域更清晰，同时通过 fallback 保持迁移顺滑。

### Decision 3: 固定 OTel semconv v1 映射，覆盖核心 agent 域

- 方案：冻结 `otel_semconv.v1`，覆盖 run/model/tool/mcp/memory/hitl 的 span 名称、父子关系和 canonical attributes。
- 备选：允许不同后端适配层自由映射。
- 取舍：自由映射会导致跨 backend 漂移，无法稳定 replay/gate；固定 v1 映射更可回归。

### Decision 4: tracing 与 eval 输出统一进入 diagnostics additive 字段

- 方案：通过 `RuntimeRecorder` 单写入口增加 `trace_export_status`、`trace_schema_version` 与 eval 相关字段。
- 备选：分别在 exporter/evaluator 模块各自写入 diagnostics。
- 取舍：多写入口会破坏幂等与回放稳定性；单写入口符合现有架构硬约束。

### Decision 5: distributed eval 只做库内嵌入式执行治理

- 方案：`runtime.eval.execution.mode=local|distributed`，distributed 仅表示库内分片/重试/续跑/聚合策略。
- 备选：引入独立远程任务调度服务。
- 取舍：服务化会偏离 `library-first`，并触发新的运维与安全面，不符合 A61 边界。

### Decision 6: gate 增加边界断言并与 replay fixture 同步冻结

- 方案：新增 `check-agent-eval-and-tracing-interop-contract.*`，至少断言：
  - `control_plane_absent`
  - tracing/eval fixture drift taxonomies
- 备选：只靠单测与集成测试。
- 取舍：缺少独立 gate 难以在 CI 上持续阻断 contract 漂移。

## Risks / Trade-offs

- [Risk] tracing 配置与既有 export 配置出现歧义  
  → Mitigation: 明确 tracing 专用配置优先级，未设置时才 fallback 到 export otlp endpoint，并在文档给出冲突解析表。

- [Risk] semconv 映射过于严格导致接入成本上升  
  → Mitigation: 固定 v1 最小字段集，允许 additive 扩展，不允许重定义同义字段。

- [Risk] distributed eval 聚合在失败重试场景下出现非幂等  
  → Mitigation: 为 shard/retry/resume 引入稳定 job/shard identity 与幂等聚合断言。

- [Risk] 新增观测字段引入高基数  
  → Mitigation: 字段长度与枚举边界沿用 diagnostics cardinality 治理策略，超限走现有截断/降级路径。

- [Risk] A61 与 A60 并行实施出现解释链冲突  
  → Mitigation: 明确“引用 A58/A59/A60 canonical 字段，不重定义同义字段”，并在 gate 加断言。

## Migration Plan

1. 扩展 `runtime/config` tracing/eval 配置结构、默认值与校验；
2. 在 `observability/trace` 实现 `otel_semconv.v1` 映射与 collector 互操作导出链路；
3. 在 `observability/event.RuntimeRecorder` 映射 tracing/eval additive 字段；
4. 扩展 `runtime/diagnostics` 存储与查询兼容；
5. 新增 eval local/distributed 执行治理与聚合逻辑；
6. 扩展 `tool/diagnosticsreplay` fixture 与 drift taxonomy；
7. 新增 `check-agent-eval-and-tracing-interop-contract.sh/.ps1` 并接入 quality gate；
8. 在 CI 暴露 `agent-eval-tracing-interop-gate` required-check 候选；
9. 同步 docs 与主线 contract index，保证配置/字段/门禁一致。

回滚策略：
- tracing/eval 配置热更新失败时原子回滚至上一有效快照；
- 可通过关闭 tracing/eval 新开关退回到 A55/A60 基线行为；
- 新增 diagnostics 字段为 additive，不影响旧消费者读取。

## Open Questions

- None. A61 采用双目标一次收口：OTel 采集接入顺滑 + 可观测/评测互操作完整兼顾。
