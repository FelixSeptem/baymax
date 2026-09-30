## Context

本设计建立在归档 `tool-schema-pressure-selection-audit` 能力之上：该能力已经拥有 canonical schema facts、pressure metrics、synthetic quality 和 replay/parity 语义，但其结论仍是单次 bounded audit，不能区分稳定瓶颈与偶发峰值。新的 readiness contract 只消费这些 provider-neutral facts，并额外接收一个有限样本窗口和宿主声明的阈值。

当前主线的 Tool/MCP/Skill/manifest/allowlist/sandbox owner 继续拥有准入与执行事实；readiness evaluator 只做 reference-only 的派生判断。已知文档状态漂移必须在同一 change 中修正，否则质量基线无法通过。

## Goals / Non-Goals

**Goals:**

- 建立 `tool_schema_projection_readiness.v1` 的 bounded 输入、规范化输出、稳定窗口判定和结论枚举。
- 将 pressure、selection quality、subset opportunity、admission integrity、semantic completeness、Run/Stream parity、replay determinism 分成可独立验证的证据轴。
- 为后续 runtime projection 设计提供可重放的触发证据，而不提前承诺运行时接入。
- 复用现有 schema audit/replay/gate owner，并为新能力提供双平台测试和文档映射。

**Non-Goals:**

- 不实现运行时按需 schema projection、selector、ModelRequest 变更或 provider adapter 变更。
- 不创建工具 registry、marketplace、动态下载、credential store、远程 discovery 或新的 admission/readiness state machine。
- 不改变 Tool/MCP 生命周期、sandbox 执行、allowlist、policy、RuntimeRecorder 或 `examples/agent-modes` 语义。

## Decisions

### 1. 采用窗口化 evidence，而不是单一聚合分数

每个 fixture 携带有限有序样本；规范化器对样本排序、去重和边界检查后计算窗口 digest。稳定触发要求在声明的最小样本数中达到阈值，单个 outlier 只能标记 `transient`。这样可以避免把一次性 schema 峰值误判为宿主级瓶颈。

备选方案是沿用归档 146 的单快照 `projection_candidate`：实现简单，但无法证明稳定性，也会把一次性压力直接升级为运行时设计信号，因此不采用。

### 2. 机会评估只允许 admitted subset

机会评估输入使用现有 audit 的 admitted identity 和 canonical facts；候选集合通过稳定 identity 排序，并逐项检查 allowlist、manifest/capability、sandbox 和必需 schema 字段。任何新增 identity、准入事实缺失或必要语义丢失都直接归类为 `blocked`。

备选方案是让 evaluator 自己重新执行准入或调用 runtime selector：这会产生第二套 admission 语义和控制面，违反 owner 边界，因此不采用。

### 3. 结论采用门槛式三值 verdict

`not_ready` 表示证据不足或仍为 transient；`ready_for_runtime_design` 只表示可以进入下一阶段设计；`blocked` 表示完整性、隐私、语义、parity 或 replay 违反。结论不会返回“已启用 projection”等运行时状态。

备选方案是单一连续 score：分数容易掩盖某一轴的安全或语义失败，也不利于 drift 分类和回滚审计，因此不采用。

### 4. 结果采用 additive、nullable/default 兼容形态

新字段只存在于离线 v1 结果中；缺少可选窗口、opportunity 或 quality 数据时保留明确的 nullable/default 轴值，不把缺失当作零压力或成功。replay 忽略未知 additive 字段并为历史 fixture 使用文档化默认值。

### 5. Gate 与文档状态作为同一交付面

在 `tool/diagnosticsreplay` 增加成功、边界、漂移和 Run/Stream fixtures；在 `tool/contributioncheck` 增加 runtime-wiring、raw-payload、admission-subset 边界检查；新增 shell/PowerShell dedicated gate 并接入 quality/docs path。Roadmap/README 的归档状态和 contract-test index 必须同步，否则禁止勾选任务。

## Risks / Trade-offs

- [阈值对宿主负载过于敏感] → 阈值和样本窗口完全由版本化 fixture 声明，结果保留样本 ordinals、digest 和 bounded metrics，后续可通过新版本而非隐式修改回溯。
- [机会评估误把质量保持视为可上线] → 结论名称明确为 `ready_for_runtime_design`，并在输出与文档中声明不启用 runtime projection。
- [新 contract 与归档 146 重叠] → 只复用其 canonical facts 和质量语义，不修改其 requirement；新 spec 专注稳定窗口、subset integrity 和 readiness verdict。
- [文档漂移掩盖真实功能回归] → 将 Roadmap/README status parity 作为独立 gate 场景，失败时使用稳定分类码并阻止完成。
- [跨平台 gate 行为不一致] → shell 与 PowerShell 使用同一 fixture/输出断言，定向测试覆盖两套脚本路径。

## Migration Plan

1. 先加入新 spec、fixture schema 和纯函数 evaluator/replay，再补 boundary、parity、privacy 和 gate 测试。
2. 将 readiness contract 映射加入 `docs/mainline-contract-test-index.md`，并修正 Roadmap/README 的归档状态与基线。
3. 接入 quality/docs gate，验证既有 `tool-schema-pressure-selection-audit` 结果不变。
4. 回滚时删除 readiness evaluator、fixture、gate 和文档映射；不涉及运行时数据、配置迁移或兼容层。

## Open Questions

无。阈值、窗口最小样本数、reason code 和字段上限属于本提案任务中必须固定并写入 fixture 的契约内容，不延后到实现阶段猜测。

## Example Impact Assessment

无需示例变更（附理由）：该设计只增加离线 readiness 证据，不改变 agent-mode 的代码、配置、运行时路径或用户可见行为。
