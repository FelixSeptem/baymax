## Context

当前仓库已有 CA2 Stage2 provider 与基于文件系统的 memory 思路，但 memory 能力仍分散在不同路径：
- 外部 memory 框架（如 mem0、zep、openviking）缺少统一 SPI 与 profile 契约；
- 现有文件系统 memory 没有作为“内置引擎”被标准化为可切换运行模式；
- 可观测性、readiness、replay、quality gate 对 memory 语义尚未形成一体化闭环。

A52 正在实施 sandbox 治理，本提案 A54 需要在不干扰 A52 的前提下，完成 memory 领域一次性完整 contract 收敛，避免后续围绕同类主题重复拆提案。

## Goals / Non-Goals

**Goals:**
- 定义统一 memory engine SPI（`Query/Upsert/Delete`）与归一化错误分类。
- 定义双模式运行契约：`external_spi|builtin_filesystem`，并支持热更新原子切换与回滚。
- 标准化内置文件系统 memory 引擎契约（append-only WAL + 原子 compaction/index）。
- 提供主流 memory framework profile pack：`mem0`、`zep`、`openviking`，并保留 `generic` 扩展位。
- 打通全局一致性治理：runtime config、readiness、diagnostics、replay、conformance、quality gate、文档/roadmap。
- 保持 Run/Stream 语义等价，并保持既有对外 API 的兼容窗口（`additive + nullable + default + fail-fast`）。

**Non-Goals:**
- 不引入新的平台控制面或跨租户 memory 调度系统。
- 不强制要求所有外部 memory provider 暴露一致底层特性，只要求 canonical 合同输出一致。
- 不在 A54 实现多副本分布式一致性存储；该能力留给后续独立里程碑。
- 不改变 A52 sandbox 合同边界，只在共享治理机制上复用 readiness/replay/gate 基础设施。

## Decisions

### Decision 1: 以 `memory facade + SPI` 作为统一接入抽象

- 方案：
  - 新增统一 memory facade，CA2/运行时只依赖 facade 与 SPI，不直接依赖具体 provider SDK。
  - SPI 固定 canonical 操作：`Query`、`Upsert`、`Delete`，并定义统一 request/response/error 语义。
- 备选：
  - 延续 provider-specific 分支（在 assembler 内直接分支 mem0/zep/openviking）。
- 取舍：
  - facade+SPI 增加一次抽象成本，但可避免主流程渗透 provider 细节，降低长期维护与扩展成本。

### Decision 2: 双模式切换采用原子热更新与失败回滚

- 方案：
  - runtime config 定义 `memory.mode=external_spi|builtin_filesystem`。
  - 启动与热更新统一走 fail-fast 校验；热更新失败时必须保留旧快照并原子回滚。
- 备选：
  - 仅在启动时切换模式，热更新不支持。
- 取舍：
  - 原子切换复杂度更高，但与现有 runtime config 体系一致，减少运维切换风险。

### Decision 3: 内置文件系统 memory 引擎采用 `WAL + compaction/index` 契约

- 方案：
  - 写入路径 append-only WAL，读路径走索引快照，compaction 通过原子替换确保崩溃恢复。
  - 明确并发语义和恢复语义，保证可回归。
- 备选：
  - 直接 JSON 覆写或无 WAL 的简化文件存储。
- 取舍：
  - WAL/compaction 实现复杂度更高，但在可恢复性、一致性和性能上更可控。

### Decision 4: 主流框架通过 profile pack 标准化，而非逐个定制散装接入

- 方案：
  - 定义 `mem0|zep|openviking|generic` canonical profile id。
  - profile 负责 provider 映射、能力声明、错误 taxonomy 对齐与默认参数解析。
- 备选：
  - 每个接入团队按需在本地封装，无统一 profile 契约。
- 取舍：
  - profile pack 需要前置 schema 设计，但可显著降低重复胶水代码和语义漂移风险。

### Decision 5: 可观测与契约测试一次性纳入主线 required-check 候选

- 方案：
  - diagnostics 增加 memory additive 字段；
  - readiness 增加 `memory.*` finding；
  - replay 增加 `memory.v1` fixture 与 drift 分类；
  - conformance/gate 增加 memory 专项 contract checks（shell/PowerShell 等价）。
- 备选：
  - 先实现功能，再后补观测和测试。
- 取舍：
  - 前期工作量更大，但能避免“功能先行、契约滞后”的重复提案循环。

### Decision 6: 适配器兼容治理扩展到 manifest/template/migration 三位一体

- 方案：
  - adapter manifest 新增 memory 领域声明字段与兼容性校验；
  - 模板与迁移映射同时覆盖 external SPI 与 builtin filesystem 切换；
  - 每个模板条目绑定 conformance case id，防止文档与实现漂移。
- 备选：
  - 仅更新代码，不同步模板与迁移映射。
- 取舍：
  - 文档维护成本上升，但接入 DX 与审计可追踪性明显提升。

## Risks / Trade-offs

- [Risk] 外部 provider 能力差异大，可能导致“最小公约数”过窄。  
  -> Mitigation: 采用 required/optional 能力分层，required 保稳定，optional 走降级可观测。

- [Risk] 模式切换复杂，热更新失败可能影响运行稳定性。  
  -> Mitigation: 严格原子切换与回滚，切换前强制 preflight 与配置校验。

- [Risk] 文件系统引擎 compaction 期间存在一致性风险。  
  -> Mitigation: 明确 WAL checkpoint 与原子 rename 规则，补 crash-recovery 合同测试。

- [Risk] 观测字段增长导致诊断卡片膨胀。  
  -> Mitigation: 所有字段遵循 bounded-cardinality 与 additive 规则，复用既有诊断预算治理。

- [Risk] conformance/gate 引入后 CI 时间上升。  
  -> Mitigation: 分层执行（快速 smoke + contract matrix），并支持独立 required-check 配置。

## Migration Plan

1. 新增 memory SPI/facade 与 runtime config 字段，保留默认兼容路径。
2. 实现 builtin filesystem engine（WAL + compaction/index）并补恢复测试。
3. 实现 external SPI profile-pack（mem0/zep/openviking/generic）与能力协商。
4. 将 CA2 Stage2 memory 接口统一迁移至 memory facade，移除主流程 provider-specific 分支。
5. 接入 readiness `memory.*` findings、diagnostics additive 字段、RuntimeRecorder 事件。
6. 新增 replay `memory.v1` fixtures 与 drift 分类断言，保证与既有 fixture 共存。
7. 扩展 adapter manifest、template、migration mapping，并绑定 conformance case id。
8. 新增 memory conformance gate 脚本并接入 `check-quality-gate.*` 与 CI 独立 required-check 候选。
9. 同步 `README`、roadmap 与契约测试索引文档，执行 docs consistency 校验。

## Open Questions

- None for A54 scope. 本提案目标是一次性冻结 memory 接入主合同；后续仅做 profile 增量扩展与实现优化，不再拆同类治理提案。
