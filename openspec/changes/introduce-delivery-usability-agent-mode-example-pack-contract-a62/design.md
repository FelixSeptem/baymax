## Context

Baymax 主干 contract（A58-A68 与 A67-CTX）已形成可回归闭环，但示例层仍存在三类问题：
- 覆盖碎片化：示例分布在 `examples/01-09` 与局部目录，模式检索成本高，覆盖边界不清晰。
- 深度不一致：部分示例仅覆盖 happy path，缺少 `production-ish` 治理路径与门禁映射。
- 迁移链路缺失：缺少 `example -> production` 的统一检查清单，PoC 到生产的落地路径不稳定。
- 历史占位回流风险：旧示例曾存在 `TODO/TBD/FIXME/待补` 占位文本，易造成“文档承诺存在、示例实现缺失”的认知偏差。
- context 生产语义依赖风险：a62 的 `context-governed` 示例依赖 context compression 的生产稳定性；若基础治理未收敛，示例验收易发生漂移。

A62 目标是在不新增业务语义的前提下，把“交付易用性”变成可审计 contract：统一模式矩阵、统一双档示例、统一迁移手册、统一门禁阻断。

## Goals / Non-Goals

**Goals:**
- 建立 `examples/agent-modes` 统一入口与 `MATRIX.md` 模式矩阵。
- 每个模式落地 `minimal + production-ish` 双档示例并映射必跑 gate。
- 补齐主干流程与自定义 adapter 关键链路示例，避免主干覆盖缺口。
- 新增 `PLAYBOOK.md` 与 `prod delta` 清单，形成 `example -> production` 迁移基线。
- 清理 `examples/` 历史 TODO 类占位并建立阻断，防止后续回流。
- 新增 A62 专项门禁并接入 `check-quality-gate.*`，保证 shell/PowerShell parity。

**Non-Goals:**
- 不在示例层新增或改写 runtime contract 语义。
- 不引入平台化控制面、托管执行平面或跨租户运维能力。
- 不在 A62 内承担 A63 命名收敛或 A64 性能优化主责。

## Decisions

### Decision 1: 采用“模式矩阵 + 双档示例”作为唯一组织方式

- 方案：所有 A62 示例统一归档到 `examples/agent-modes`，并在 `MATRIX.md` 维护 `pattern -> minimal -> production-ish -> contracts -> gates -> replay` 映射。
- 原因：以矩阵替代散点示例，降低检索和评审成本。
- 备选：延续历史目录分布，仅补索引文档。
- 取舍：统一矩阵维护成本更高，但可获得一致的可回归治理。

### Decision 2: 示例只做 contract 复用，不定义第二语义层

- 方案：示例字段、事件、诊断口径必须引用 A56-A68 与 A67-CTX（若启用）既有 contract 输出。
- 原因：避免 examples 侧出现“可运行但不可回归”的平行解释。
- 备选：示例中定义自解释字段并在文档说明差异。
- 取舍：严格复用降低灵活性，但保证长期一致性与可审计性。

### Decision 3: 主干流程与自定义 adapter 覆盖设为强制项

- 方案：把 mailbox/task-board/scheduler/readiness 主干链路和 `mcp/model/tool/memory` adapter 链路列为必覆盖范围。
- 原因：这两类路径最接近真实交付，且最容易出现“文档有、示例无”的空洞。
- 备选：先覆盖高频模式，主干与 adapter 后续补齐。
- 取舍：一次性范围更大，但可减少后续重复提案。

### Decision 4: 迁移能力采用“PLAYBOOK + prod delta checklist”双层治理

- 方案：新增 `PLAYBOOK.md` 作为全局迁移手册；每个 `production-ish` README 增加 `prod delta` 章节。
- 原因：全局手册解决统一流程，局部清单解决模式差异。
- 备选：仅保留单一全局迁移文档。
- 取舍：双层维护成本更高，但迁移可执行性更强。

### Decision 5: A62 质量门禁采用“三步阻断”

- 方案：`examples-smoke`（可运行）+ `pattern-coverage`（不漏模式）+ `migration-playbook-consistency`（文档-示例-门禁一致）。
- 原因：只跑 smoke 容易漏覆盖；只做覆盖校验又无法证明可运行。
- 备选：仅运行 smoke 或仅运行矩阵静态检查。
- 取舍：三步阻断耗时更高，但能显著降低回归漏检。

### Decision 6: 历史示例 TODO 占位实行“清零 + 禁回流”治理

- 方案：A62 交付前，`examples/` 历史示例中的 `TODO/TBD/FIXME/待补` 占位必须清零；未完成事项迁移到 `MATRIX.md`/`PLAYBOOK.md`/`tasks.md` 可追踪条目。
- 原因：示例是交付入口，占位符会造成“可参考但不可落地”的错误预期。
- 备选：允许保留 TODO 并在 README 解释。
- 取舍：清零治理提高维护门槛，但能显著提升示例可信度与可执行性。

### Decision 7: context-governed 子项采用“A69 前置收口、A62 后置验收”

- 方案：a62 非 context 模式按既有计划并行推进；`a62-T15 context-governed-reference-first` 与其验收必须引用 A69 gate/replay 输出，并在 A69 收敛后完成最终标记。
- 原因：示例层不应倒逼 runtime 语义定义，必须基于稳定 context compression 生产合同。
- 备选：让 a62 先自定义 context-governed 验收口径，后续再回收。
- 取舍：局部任务节奏变慢，但能显著降低示例与 runtime contract 双向漂移风险。

### Decision 8: 后续提案必须执行 Example Impact Assessment

- 方案：后续涉及行为/配置/契约变化的提案，必须在 proposal/design/tasks 中显式给出 example 影响评估结果：`新增示例`、`修改示例` 或 `无需示例变更（附理由）`。
- 原因：避免 runtime contract 演进后示例滞后，导致交付入口与主干语义脱节。
- 备选：仅在评审评论中口头确认是否需要示例改动。
- 取舍：提案编写负担略增，但可显著降低“代码已变、示例未跟进”的长期维护风险。

## Risks / Trade-offs

- [Risk] 示例数量增长导致维护成本抬升。  
  -> Mitigation: 采用 `minimal/prod-ish` 分层与矩阵索引，新增示例必须绑定 gate 映射。

- [Risk] `production-ish` 示例漂移为“伪生产”文档。  
  -> Mitigation: 强制 `prod delta` 清单与 `PLAYBOOK` 一致性门禁阻断。

- [Risk] 示例与主干 contract 字段漂移。  
  -> Mitigation: 示例统一注入 diagnostics/tracing，并纳入 replay/drift 回归。

- [Risk] 门禁接线后 CI 耗时增加。  
  -> Mitigation: 支持按模式子集执行 smoke，保留 required-check 最小阻断路径。

- [Risk] 旧示例迁移期间出现双入口认知混乱。  
  -> Mitigation: 通过 README 映射表明确“旧路径 -> 新模式”迁移关系，分阶段退场。

- [Risk] TODO 清理不彻底，后续改动再次引入占位。  
  -> Mitigation: 新增 `legacy-todo-cleanup` 阻断门禁并在 PR 校验中默认启用。

- [Risk] A69 收敛延迟导致 context-governed 示例验收后移。  
  -> Mitigation: 明确“非 context 并行、context 后置”执行策略，并在任务清单中将 A69 依赖显式化。

## Migration Plan

1. 建立 `examples/agent-modes` 目录骨架与 `MATRIX.md`。  
2. 按 P0 -> P1 -> P2 分批迁移/新增示例，先保证可运行与主干覆盖。  
3. 接入三类 A62 专项门禁并并入 `check-quality-gate.*`。  
4. 发布 `PLAYBOOK.md` 与 `prod delta` 清单，完成旧示例映射说明。  
5. 全量执行 quality/docs consistency 后，将 A62 变更标记为可实施。  

回滚策略：保留旧示例入口映射，门禁新增项可独立关闭并回退到 smoke-only 路径，但需在变更记录中说明原因与恢复计划。
