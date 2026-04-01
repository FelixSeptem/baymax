## Context

A54 已完成 memory SPI 与 builtin filesystem 基线，但当前仍缺少 A59 目标中的统一治理面：
- memory scope（`session|project|global`）尚未冻结解析优先级与预算裁剪规则；
- 写入策略尚未形成与 backend 选择解耦的 contract；
- 检索质量、生命周期与恢复一致性缺少统一 replay/gate 口径。

与此同时，A58 正在收敛跨策略层判定链，A59 需要在不引入平台控制面的前提下，把 memory 相关 contract 一次性补齐，避免后续再拆平行提案。

约束条件：
- 保持 `library-first + contract-first`；
- 保持 `RuntimeRecorder` 单写入口；
- 配置保持 `env > file > default`，非法配置 fail-fast，热更新失败原子回滚；
- 不破坏 A54 既有 `runtime.memory.mode=external_spi|builtin_filesystem` 语义。

## Goals / Non-Goals

**Goals:**
- 冻结 memory scope 解析、注入预算、写入模式、检索治理、生命周期治理的 contract。
- 在 builtin filesystem 路径补齐 v2 级别的一致性能力：索引更新策略、恢复校验、drift detect。
- 冻结 A59 可观测字段、replay fixtures 与 drift taxonomy，并接入独立 gate。
- 保持 external SPI 与 builtin filesystem 双路径语义等价（在等效输入下）。

**Non-Goals:**
- 不引入托管 memory 控制面、远程管理服务或平台化运维面板。
- 不替换 A54 SPI 主接口（`Query/Upsert/Delete`）或既有 fallback taxonomy。
- 不改写 A58 策略裁决合同，也不提前吸收 A60 成本/时延 admission 规则。

## Decisions

### Decision 1: 保留 backend selector，新增 write mode 维度

- 方案：保持既有 `runtime.memory.mode=external_spi|builtin_filesystem` 用于 backend 选择；新增 `runtime.memory.write_mode=automatic|agentic` 管理写入策略。
- 备选：复用 `runtime.memory.mode` 承载 backend 与写入策略。
- 取舍：复用会造成枚举语义冲突并破坏兼容；拆分字段可在保持旧行为稳定的前提下扩展策略面。

### Decision 2: Scope 解析采用 deterministic fallback 链

- 方案：scope 解析顺序固定为 `session -> project -> global`，未命中时按 deterministic fallback 返回空结果或降级来源标记。
- 备选：由调用侧自行决定 scope 顺序。
- 取舍：调用侧自定义会导致 Run/Stream 与多入口漂移；固定顺序更易回放与门禁验证。

### Decision 3: 检索治理采用可分层开关，不引入强绑定外部依赖

- 方案：检索链路固定为 `retrieve -> rerank(optional) -> temporal_decay(optional)`；支持 keyword/vector hybrid 组合但不强制绑定特定向量数据库。
- 备选：直接引入特定外部检索服务作为主路径。
- 取舍：A59 目标是 contract 收敛而非平台绑定，保持 provider-neutral 更符合 library-first。

### Decision 4: Filesystem v2 一致性采用 snapshot + WAL + index checksum

- 方案：沿用 A54 snapshot/WAL 主链，新增 index checksum 与 drift detect；出现 drift 时按策略执行增量重建或全量重建并记录 canonical reason。
- 备选：仅在查询失败时惰性重建。
- 取舍：惰性重建可实现但不可预测；显式 drift detect 更利于诊断与回归测试。

### Decision 5: 可观测与回放只做 additive 扩展

- 方案：新增 QueryRuns 字段（`memory_scope_selected`、`memory_budget_used`、`memory_hits`、`memory_rerank_stats`、`memory_lifecycle_action`）并保持 nullable/default 兼容；新增 `memory_scope.v1`、`memory_search.v1`、`memory_lifecycle.v1` fixtures。
- 备选：复用现有 memory 统计字段，不新增 fixture。
- 取舍：复用会降低可解释性并无法稳定捕获 A59 新漂移类型。

## Risks / Trade-offs

- [Risk] 字段拆分（`mode` vs `write_mode`）引入配置迁移认知成本  
  → Mitigation: 在 `docs/runtime-config-diagnostics.md` 与 `memory/README.md` 提供对照表和迁移示例。

- [Risk] 检索链路新增 rerank/decay 后带来额外 CPU 与 latency 开销  
  → Mitigation: 全量能力可开关；默认启用 conservative 配置，并通过回归门禁验证阈值。

- [Risk] drift detect 触发重建导致冷启动时间抖动  
  → Mitigation: 支持增量优先、全量兜底；记录 `recovery_consistency_drift` 便于快速定位。

- [Risk] external SPI 与 builtin 双路径行为不一致  
  → Mitigation: 增加双路径等价集成测试与 mixed replay fixtures，作为 required-check 候选阻断。

## Migration Plan

1. 扩展配置模型：引入 `runtime.memory.scope.*`、`runtime.memory.write_mode.*`、`runtime.memory.injection_budget.*`、`runtime.memory.lifecycle.*`、`runtime.memory.search.*`。
2. 在 `memory/facade` 注入 scope 解析与 budget 裁剪逻辑，保持 SPI 接口不变。
3. 扩展 filesystem engine 到 v2：索引更新策略、checksum/drift detect、恢复重建策略。
4. 扩展 diagnostics/recorder additive 字段并保证单写入口与幂等。
5. 增加 replay fixtures 与 drift 分类，并落地 `check-memory-scope-and-search-contract.*`。
6. 将新 gate 接入 `check-quality-gate.*`，暴露独立 required-check 候选。
7. 同步文档（roadmap、runtime config/diagnostics、mainline index、memory README、README）。

回滚策略：
- 任何配置非法或热更新失败都回滚到上一个有效快照；
- 如果 A59 新字段导致兼容问题，可禁用 `write_mode/search/lifecycle` 新能力，退回 A54 基线行为。

## Open Questions

- None. A59 以“一次性收敛 memory contract”为目标，不预留同域平行拆案。
