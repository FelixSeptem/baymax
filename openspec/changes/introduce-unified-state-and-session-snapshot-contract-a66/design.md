## Context

现有仓库已经具备多处恢复与快照能力：
- scheduler/store 的任务状态快照；
- composer recovery store 的运行态快照；
- memory filesystem 引擎的 WAL/snapshot 机制；
- diagnostics/replay 的回放一致性门禁。

问题在于这些能力尚未形成统一 state/session snapshot 合同：跨模块导入导出需要调用方手工拼接，兼容窗口和冲突策略分散，难以稳定支撑迁移与长链路恢复。roadmap 已明确 A66 需要“复用现有 checkpoint/snapshot 语义与 A59 memory lifecycle，不重写存储层事实源”，因此本设计以统一合同层为主，不做底层事实源替换。

## Goals / Non-Goals

**Goals:**
- 定义统一 snapshot manifest 与模块分段结构，覆盖 runner/session、scheduler/mailbox、composer recovery、memory。
- 定义导入导出合同：版本、兼容窗口、partial restore、冲突策略、幂等保证。
- 定义运行时配置合同：`runtime.state.snapshot.*`、`runtime.session.state.*`。
- 定义可观测与回放合同：A66 additive 字段、`state_session_snapshot.v1` fixture 与 drift taxonomy。
- 保证 Run/Stream 与 memory/file 后端恢复语义一致，不引入平行解释链。

**Non-Goals:**
- 不替换或重写 scheduler/composer/memory 现有事实源存储实现。
- 不引入托管状态控制面、远程恢复调度服务或平台化迁移中心。
- 不在 A66 内展开性能专项（A64 负责）或示例收口（A62 负责）。

## Decisions

### Decision 1: 采用“统一 manifest + 模块分段 payload”的快照格式

- 方案：顶层 manifest 维护版本、来源、兼容窗口、校验摘要；每个模块以分段 payload 表达并保留原有语义。
- 备选：完全扁平化单结构快照。
- 取舍：分段结构更容易复用现有模块快照语义，也便于局部恢复和向后兼容。

### Decision 2: 导入恢复采用 `strict|compatible` 双模式

- 方案：
  - `strict`：版本或字段不兼容即 fail-fast；
  - `compatible`：在兼容窗口内允许有界降级恢复，并记录恢复动作。
- 备选：统一宽松模式。
- 取舍：双模式可同时满足生产稳态和迁移阶段需求，并避免 silent drift。

### Decision 3: 冲突仲裁复用既有 taxonomy，不新增平行语义

- 方案：恢复冲突码、决策链和拒绝语义复用现有 runtime/diagnostics taxonomy，仅做 additive 字段扩展。
- 备选：A66 定义独立冲突码体系。
- 取舍：独立体系会造成解释分叉并增加回放成本。

### Decision 4: memory 仅做生命周期对齐，不重写事实源

- 方案：统一 snapshot 通过 memory SPI 接缝消费/产出，生命周期与 A59 保持一致。
- 备选：在 A66 改造 memory 底层存储模型。
- 取舍：严格遵守 roadmap 约束，降低跨提案耦合风险。

### Decision 5: 回放与门禁采用 fixture-first + 独立 gate

- 方案：新增 `state_session_snapshot.v1` 与 drift 分类；新增 `check-state-snapshot-contract.*` 并接入质量门禁。
- 备选：仅依赖集成测试覆盖。
- 取舍：独立 gate 对跨模块一致性与兼容窗口漂移拦截更稳定。

### Decision 6: 导入导出操作必须幂等且可恢复

- 方案：导入接口引入操作标识与幂等语义，重复导入不应膨胀统计或破坏终态。
- 备选：允许 best-effort 非幂等导入。
- 取舍：非幂等会使 replay 和故障恢复不可预测。

## Risks / Trade-offs

- [Risk] 统一合同层增加实现复杂度，初期接线成本高。  
  → Mitigation: 保持“合同层收口 + 事实源不改写”，按模块分段逐步接入。

- [Risk] `compatible` 模式滥用可能掩盖 schema 漂移。  
  → Mitigation: 兼容窗口显式配置 + drift 分类 + gate 阻断。

- [Risk] 跨模块恢复冲突在并行实施期可能增多。  
  → Mitigation: 冲突码统一、恢复动作可观测、replay fixture 覆盖典型冲突矩阵。

- [Risk] 增加导入导出路径可能影响运行态稳定性。  
  → Mitigation: 导入导出默认按开关关闭，逐步启用并配合 contract gate。

## Migration Plan

1. 配置层：在 `runtime/config` 新增 `runtime.state.snapshot.*` 与 `runtime.session.state.*`，实现 fail-fast 与热更新回滚。
2. 合同层：定义统一 manifest/schema 与模块分段描述，落地序列化/反序列化与兼容校验。
3. 接缝层：在 composer/scheduler/memory 路径接入导入导出适配层，保持现有事实源不变。
4. 观测层：在 diagnostics/recorder 增加 A66 additive 字段与冲突动作记录。
5. 回放层：新增 `state_session_snapshot.v1` fixture、drift 分类、mixed fixture 兼容测试。
6. 门禁层：新增 `check-state-snapshot-contract.sh/.ps1`，接入 `check-quality-gate.*`。
7. 文档层：同步 runtime config/diagnostics、contract index、roadmap 与 README。

回滚策略：
- 配置回滚：热更新非法配置自动回滚到上一个有效快照；
- 功能回滚：关闭 `runtime.state.snapshot.*` / `runtime.session.state.*` 开关恢复到现有恢复路径；
- 数据兼容：新增字段保持 additive，旧解析器可安全忽略。

## Open Questions

- None. A66 按 roadmap 占位口径一次性收口 state/session snapshot 同域需求，不再拆平行提案。
