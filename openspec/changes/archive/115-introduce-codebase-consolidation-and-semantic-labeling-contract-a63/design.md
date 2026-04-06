## Context

A67-CTX 与 A68 实施后，Baymax 主干能力进入“语义稳定、命名不稳定”的阶段：核心 contract 已较完整，但活动目录仍残留编号化命名（`ca1/ca2/ca3/ca4` 与 `Axx`）以及临时/过时文档条目。该状态会放大跨模块沟通成本，也会让 gate、README、测试命名与实际实现逐步失配。A63 的设计目标是做一次“语义不变的收敛整理”：统一命名、收敛文档、保留可回滚兼容跳板，并通过门禁阻断回流。

约束前提：
- 不改变 Run/Stream、readiness/admission、reason taxonomy、diagnostics/replay 语义。
- 运行态写入继续走 `RuntimeRecorder` 单写入口。
- 架构边界规则（module dependency、config precedence、fail-fast + 原子回滚）保持不变。
- `Axx` 仅允许保留在 `openspec/**`；`openspec/**` 之外（含内容、文件路径、文件名）必须消除 `Axx`。
- 必须定义“`openspec/**`（允许）vs 非 `openspec/**`（禁止）”矩阵并作为 gate 规则的唯一来源，避免扫描规则在不同脚本中分叉。

## Goals / Non-Goals

**Goals:**
- 在活动代码、测试、脚本、文档中完成 Context Assembler 命名统一，不再使用 `ca1|ca2|ca3|ca4` 作为活动口径。
- 在 `openspec/**` 之外移除 `Axx` 编号表述并替换为语义化描述（含内容、文件路径、文件名）；编号映射仅在 `openspec/**` 内管理。
- 清理临时/过时文档与离线生成物，仅保留“当前现状 + 可追溯索引”。
- 补齐运行时 Harness 架构单一总览入口，统一 `state surfaces -> guides/sensors -> tool mediation -> entropy control` 与主干 contract/gate 映射路径。
- 对公开配置键、诊断字段、脚本入口与测试夹具提供兼容 alias 与回滚路径。
- 新增命名回流阻断 gate（shell/PowerShell 语义等价），持续防止旧命名回流。
- 建立大文件拆分治理：限制非 `openspec/**` 的 `*.go` 文件行数上界，并通过可审计例外清单做渐进收敛。

**Non-Goals:**
- 不引入新运行时能力或新对外 API。
- 不改写既有 contract 字段语义或 replay fixture 语义。
- 不在 A63 内处理性能专项优化（归 A64）。
- 不改变 `openspec/**` 内历史编号事实与归档可追溯性。

## Decisions

### Decision 1: 采用“语义词表 + 单一映射源”治理命名
- 方案：维护唯一映射表（semantic name <-> legacy number/name），实现、文档、脚本帮助统一引用该映射。
- 备选：各模块 README/注释分别维护局部映射。
- 取舍：单一映射源可降低漂移，便于 docs consistency 自动校验。

### Decision 2: Context Assembler 命名采用“语义主名 + 兼容别名”两阶段收敛
- 方案：第一阶段保留旧名 alias（输入兼容、输出语义主名）；第二阶段在活动目录清除旧名，仅映射层保留历史引用。
- 备选：一次性硬切命名。
- 取舍：两阶段可降低批量重命名导致的回归风险，并支持回滚。

### Decision 3: `Axx` 文本清理由“目录分层”执行
- 方案：
  - `openspec/**` 允许保留 `Axx`（内容、路径、文件名）用于历史追溯；
  - `openspec/**` 之外禁止 `Axx`（内容、路径、文件名）；
  - 对外说明统一使用语义名称，并在 `openspec/**` 索引提供映射跳转。
- 备选：全仓库彻底移除 `Axx`。
- 取舍：按 `openspec` 单目录分层可保留历史可追溯性，同时避免“多个特例目录”导致规则漂移。

### Decision 4: 门禁引入“命名回流扫描 + openspec 单一例外规则”
- 方案：在 `check-quality-gate.*` 和 docs consistency 相关流程中加入命名扫描；`Axx` 仅允许 `openspec/**`，不再保留其他 allowlist 特例。
- 备选：仅靠 reviewer 人工审查。
- 取舍：自动阻断更稳定，可持续防回流；单一例外（仅 `openspec/**`）可显著降低误配概率。

### Decision 5: 文档遵循“当前现状优先”并固定入口路径
- 方案：
  - `README.md` 与核心模块 README 只描述当前可用能力；
  - roadmap 描述 active/candidate/archive 状态，不保留临时叙事；
  - 语义映射、归档规则、门禁路径在固定文档入口集中维护。
- 备选：保留多份临时说明并在 PR 中补充解释。
- 取舍：集中入口可减少重复、降低读者认知切换。

### Decision 6: 对公开契约字段采用“语义主名 + 兼容旧名”迁移窗口
- 方案：对 `runtime/config` 键、`runtime/diagnostics` 字段、`run.finished` payload、replay fixture 键实行双读策略，默认输出语义主名，旧名兼容读取并提供去除时间窗。
- 备选：直接重命名公开字段，不提供兼容窗口。
- 取舍：双读迁移可避免外部消费方一次性断裂，同时满足命名收敛目标。

### Decision 7: gate/test/env 标识从编号转语义，并保留集中映射
- 方案：将脚本函数名、测试名、环境变量前缀与 fixture label 中 `Axx`/`CAx` 标识改为语义化标识；`Axx` 仅在 `openspec/**` 映射中保留。
- 备选：保持编号标识，仅更新文档说明。
- 取舍：仅改文档无法消除代码面编号耦合，后续仍会持续回流。

### Decision 8: 引入 `*.go` 单文件行数预算与超限拆分策略
- 方案：
  - 为非 `openspec/**` 的受治理 `*.go` 文件设定单文件行数预算（默认阈值 + 硬阈值）；
  - 对超限文件执行语义拆分（提取子文件/子模块），保持导出行为、控制流逻辑与契约语义不变；
  - 对短期无法拆分的历史文件采用受控例外清单（owner、原因、到期时间），并在 gate 中阻断新增超限或超限扩张。
  - 拆分类变更必须触发强校验：Run/Stream parity、impacted contract suites、diagnostics replay idempotency 全部通过后才允许合入。
- 备选：仅在代码评审中建议拆分，不设自动化门禁。
- 取舍：仅靠人工审查难以持续；预算 + gate + 例外清单可执行性更高，且可渐进推进。

### Decision 9: 运行时 Harness 架构采用“单文档总览 + 现有索引引用”收口
- 方案：新增并维护单一总览文档（建议 `docs/runtime-harness-architecture.md`），集中描述运行时 outer-loop 结构与 contract/gate 映射；其他 README/docs 仅引用 canonical 路径，不重复维护平行叙述。
- 备选：继续在 README、roadmap、module-boundaries 中分散维护架构叙述。
- 取舍：单入口可减少重复和口径漂移，但需要 docs consistency 持续校验链接与内容同步。

## Risks / Trade-offs

- [Risk] 批量重命名引起测试名、脚本名、文档锚点断裂。
  -> Mitigation: 分阶段实施；先 alias 后替换；为关键脚本提供兼容入口与重定向提示。

- [Risk] 过度清理导致历史上下文缺失，影响排障。
  -> Mitigation: 历史编号仅在 `openspec/**` 索引层保留；所有清理动作附迁移映射与归档记录。

- [Risk] 命名扫描规则误报，影响开发效率。
  -> Mitigation: 对 `Axx` 采用“仅 `openspec/**` 允许”规则，其他目录零特例；规则输入集中在单一矩阵文件。

- [Risk] 文档“现状化”过程中遗漏模块入口。
  -> Mitigation: 以 `core-module-readme-richness` 基线做清单校验，并加入 docs consistency 对齐检查。

- [Risk] 运行时 Harness 总览文档与现有索引内容不一致。
  -> Mitigation: 固定 canonical path，并在 docs consistency 中加入双向引用与路径有效性检查。

- [Risk] 公开配置键/诊断字段重命名引发外部解析失败。
  -> Mitigation: 增加语义主名与旧名双读兼容、parser compatibility 测试、mixed fixture 回放断言。

- [Risk] gate 命名扫描对历史索引目录误报或漏报。
  -> Mitigation: `openspec/**` 唯一例外 + 非 `openspec/**` 全量扫描，并在 shell/PowerShell 复用同一规则输入。

- [Risk] 大文件拆分引发包内循环依赖或行为回归。
  -> Mitigation: 采用“纯重构”拆分策略（先提取私有辅助逻辑再收敛调用点），并以强校验门禁（Run/Stream parity + impacted contract + replay）兜底验证。

- [Risk] 行数阈值设置过严导致推进阻塞。
  -> Mitigation: 设置硬阈值与受控例外清单，先阻断新增超限与超限扩张，再逐步收敛历史超限文件。

- [Risk] 将非 Go 文件纳入同一规则导致误拦截。
  -> Mitigation: A63 仅治理 `*.go` 文件；其他文件类型暂不纳入该门禁。

## Migration Plan

1. 盘点与映射：生成 A63 命名盘点清单，建立集中映射表并标记高风险路径（配置、诊断、脚本、测试夹具）。
2. 兼容桥接：为公开配置键、脚本入口、测试夹具增加 alias/迁移提示，保证旧入口可用且可观测。
3. 语义替换：批量替换 `openspec/**` 之外目录中的 `ca1|ca2|ca3|ca4|Axx` 表述为语义化描述；同步更新 README 与 docs。
4. 契约迁移：为配置键、诊断字段、payload 与 replay fixture 增加语义主名与兼容旧名策略，并补齐 parser/mixed-fixture 兼容测试。
5. 门禁接线：新增/强化命名回流扫描并接入 quality gate + docs consistency（shell/PowerShell parity）。
6. 大文件治理：盘点超限 `*.go` 文件，执行语义拆分；引入 `*.go` 行数预算检查脚本并接入质量门禁。
7. 冗余清退：剔除离线 scaffold 冗余副本、过时阶段文档、临时备份工件，并纳入仓库卫生检查。
8. 运行时 Harness 文档收口：新增/更新 canonical 总览文档并同步 root/module README 的入口引用。
9. 验证与收口：执行 contract/replay/quality/docs 强校验门禁；确认行为语义与逻辑均不漂移后去除不再需要的临时桥接。

回滚策略：
- 代码层：保留 alias 与迁移映射可快速切回旧入口；
- 文档层：`openspec/**` 内索引映射与历史归档始终保留可追溯链接；
- 门禁层：若扫描规则误伤，可回退到上一版规则并保留 fail-fast 主流程。

## Open Questions

- None. A63 范围与强制项已由 roadmap 固化，实施中仅允许在该提案内增量补充，不再拆分平行提案。
