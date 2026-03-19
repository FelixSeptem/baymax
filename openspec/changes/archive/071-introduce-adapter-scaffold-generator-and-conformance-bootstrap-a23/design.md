## Context

A21 已提供外部 adapter 样板与迁移映射，A22 正在提供 conformance harness 与阻断门禁。但对贡献者而言，首接入仍需要手工创建目录、拷贝片段、手动拼装 conformance 用例，导致：
- 启动路径不稳定，不同贡献者产物结构差异大；
- 样板与 conformance 约束之间缺少自动对齐机制；
- 质量门禁无法在“样板层”提前发现漂移。

A23 在不改变 runtime 行为的前提下，补齐“可生成、可验证、可阻断”的 adapter 入门链路。

## Goals / Non-Goals

**Goals:**
- 提供统一 scaffold 生成入口，支持 `mcp`、`model`、`tool` 三类 adapter。
- 生成产物默认离线、确定性，目录与文件结构可重复。
- 默认生成 conformance bootstrap 骨架，直接映射 A22 harness 执行入口。
- 默认 no-overwrite + fail-fast；仅 `--force` 允许覆盖。
- 在 quality-gate 中新增 scaffold drift 阻断检查（shell/PowerShell 一致）。

**Non-Goals:**
- 不变更现有 adapter runtime 协议或执行语义。
- 不引入 adapter marketplace、registry 或平台化托管能力。
- 不在 A23 引入性能阈值验收（仍由既有 benchmark gate 负责）。

## Decisions

### 1) 采用“库 + 薄 CLI”结构
- 方案：新增可复用生成库（如 `adapter/scaffold`），`cmd/adapter-scaffold` 仅做参数解析与调用。
- 原因：保持 library-first 定位，便于后续被测试、工具链和外部自动化复用。
- 备选：仅实现 CLI。拒绝原因：复用性与可测试性不足。

### 2) 生成前执行全量冲突预检，避免部分写入
- 方案：先构建完整文件计划并检查冲突；若任一冲突且未启用 `--force`，直接 fail-fast 退出且不写文件。
- 原因：保证原子性预期，降低半生成状态带来的清理成本。
- 备选：边写边检查。拒绝原因：失败后容易留下不一致产物。

### 3) 默认路径和输出命名固定
- 方案：默认输出 `examples/adapters/<type>-<name>`，`type` 仅允许 `mcp|model|tool`。
- 原因：便于文档引用、CI 巡检和 drift 比对统一定位。
- 备选：随机或自由命名默认路径。拒绝原因：可追踪性差、文档难维护。

### 4) conformance bootstrap 默认开启并与 A22 最小矩阵对齐
- 方案：生成最小 conformance 测试骨架和执行说明，直接对接 A22 harness 的入口约定。
- 原因：把“生成即可测”作为默认路径，降低遗漏契约校验的概率。
- 备选：默认关闭 bootstrap。拒绝原因：增加人工步骤，易偏离契约。

### 5) drift 检查作为质量门禁阻断项
- 方案：新增 `check-adapter-scaffold-drift.sh/.ps1`，在 quality-gate 中以 fail-fast 阻断方式执行。
- 原因：持续防止模板与生成结果长期漂移。
- 备选：仅文档提示人工检查。拒绝原因：不可回归、不可规模化。

## Risks / Trade-offs

- [Risk] 模板数量增长导致维护成本上升  
  → Mitigation: 约束为最小可执行骨架，并通过 drift 检查自动发现偏差。

- [Risk] A22 conformance harness 接口演进导致 bootstrap 失配  
  → Mitigation: 生成器仅依赖 A22 稳定入口，变更需同步更新 contract tests 与脚手架模板。

- [Risk] 过于严格的 no-overwrite 影响增量生成体验  
  → Mitigation: 提供显式 `--force`，并在冲突输出中给出具体文件列表。

## Migration Plan

1. 引入 scaffold 生成库与 `cmd/adapter-scaffold` 命令。
2. 增加三类 adapter 模板与占位符替换机制。
3. 生成 conformance bootstrap 骨架并接入 A22 最小矩阵入口。
4. 增加 scaffold drift 检查脚本与集成测试。
5. 将 drift 检查接入 `check-quality-gate.sh/.ps1` 阻断路径。
6. 更新 README、roadmap 与 mainline 契约索引。

回滚策略：
- 若生成器稳定性不足，可先回滚 quality-gate 中的 drift 接入点；
- 生成器与模板代码可保留在非阻断路径继续迭代，不影响 runtime 主链路。

## Open Questions

- 当前无阻塞问题；按既定推荐值冻结：默认路径、默认 bootstrap 开启、no-overwrite + `--force`、gate 阻断、离线确定性。
