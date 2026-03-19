# Development Roadmap

更新时间：2026-03-19

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-03-19：
- 已归档并稳定：A4-A21（含 A19 性能门禁、A20 全链路示例、A21 外部适配模板与迁移映射）。
- 进行中：
  - `introduce-external-adapter-conformance-harness-and-gate-a22`
  - `introduce-adapter-scaffold-generator-and-conformance-bootstrap-a23`

## 1.0.0 基线定义（本仓库后续参考口径）

`1.0.0` 以“收口基线”为目标，不再按 A 编号无限扩展范围。当前基线由以下能力组成：

1. 运行时主干稳定：
- Runner Run/Stream 统一语义与并发背压基线。
- Multi-provider（OpenAI/Anthropic/Gemini）统一 contract。
- Context Assembler CA1-CA4、Security S1-S4 已归档能力。

2. 多代理主链路稳定：
- A11-A18（同步/异步/延后、恢复边界、协作原语、统一诊断查询）语义收口。
- Shared contract gate 与 Run/Stream 等价约束保持阻断。

3. 质量与可回归稳定：
- A19 性能回归门禁（基线 + 相对阈值）。
- A20 全链路示例 smoke 阻断门禁。

4. 外部接入稳定：
- A21 模板与迁移映射（已归档）。
- A22 conformance harness（进行中，需归档）。
- A23 scaffold + conformance bootstrap（进行中，需归档）。

## 1.0.0 里程碑与退出条件

### M1：外部适配链路收口（当前阶段）

完成条件：
- A22 归档：`MCP > Model > Tool` 最小一致性矩阵与 gate 阻断路径稳定。
- A23 归档：脚手架生成、默认目录、`--force` 覆盖策略、bootstrap 对齐与 drift gate 稳定。
- `README` / `runtime-config` / `contract-index` / roadmap 口径对齐。

### M2：1.0.0-RC 冻结

完成条件：
- 不再接收新增功能型提案（A24+）。
- 全量质量门禁连续通过（本地与 CI 口径一致）。
- 无 P0 未关闭缺陷（语义错误、数据损坏、崩溃、严重性能回退、安全阻断问题）。

### M3：1.0.0 发布

完成条件：
- RC 阶段无新增破坏性语义调整。
- 变更仅限 P0/P1 缺陷修复与文档澄清。
- `CHANGELOG` 与版本策略文档完成发布同步。

## 新增提案收敛规则（避免无限追加）

从本文件生效起，到 `1.0.0` 发布前遵循：

1. 默认不新增 A24+ 功能提案。
2. 仅允许以下“阻断型提案”进入：
- P0 安全问题修复。
- P0 正确性问题（contract 违背、Run/Stream 语义不一致）。
- P0 稳定性问题（崩溃、死锁、数据损坏、不可恢复回退）。
- 已有门禁项的严重回归修复（性能/契约/质量门禁无法通过）。
3. 非阻断型需求统一进入 `post-1.0 backlog`，不进入 1.0.0 范围。
4. 任何例外必须在提案中写明：`Why now`、风险、回滚方案、对 1.0.0 时间线影响。

## Post-1.0 Backlog（仅登记，不纳入 1.0.0）

以下方向明确延后：
- 平台化控制面（多租户、RBAC、审计与运营面板）。
- 跨租户全局调度与控制平面。
- 市场化/托管化 adapter registry 能力。

说明：`post-1.0 backlog` 只做记录，不作为当前迭代实施输入。

## 执行与验收规则

- 单变更优先；并行变更需显式依赖边界。
- 严格顺序：`proposal/design/spec/tasks -> code -> tests -> docs`。
- 合并前最少验证：
  - `go test ./...`
  - `go test -race ./...`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `pwsh -File scripts/check-quality-gate.ps1`
