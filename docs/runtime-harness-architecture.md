# Runtime Harness Architecture

更新时间：2026-04-05

> 本文档是运行时 Harness 的 canonical 总览入口。模块级 README 与主题文档应引用本页，不重复维护平行架构叙述。

## 1. 目标与边界

Runtime Harness 负责把 `Run/Stream` 主循环、工具调度、配置治理与可观测性连接为一个可回归、可回放、可审计的执行平面。

硬约束：

- 不改变 `Run/Stream` 终态语义一致性。
- 运行时配置优先级固定为 `env > file > default`。
- 非法配置与非法热更新必须 fail-fast，并保持原子回滚。
- 运行态诊断写入走 `observability/event.RuntimeRecorder` 单写入口。

## 2. 架构主链

### 2.1 State Surfaces（状态表面）

- `core/runner`：主循环状态机（迭代、终态、取消传播、回压）。
- `runtime/config`：配置快照、热更新、readiness 校验。
- `runtime/diagnostics`：run/timeline/query 读取面与聚合读取 API。
- `context/journal` 与 spill 文件后端：上下文演进与回放可追溯面。
- `observability/event`：结构化事件、RunFinished payload 与 recorder 对接。

### 2.2 Guides / Sensors（引导与传感）

- Policy guides：fail-fast / best-effort、backpressure、retry、timeout、admission。
- Sensors：timeline reason、diagnostics 聚合、taxonomy 分类、基准回归指标。
- 语义守恒：Run/Stream parity、replay idempotency、contract suite 稳定性。

### 2.3 Tool Mediation（工具中介层）

- `tool/local`：本地工具注册、参数校验、调度策略。
- `mcp/http`、`mcp/stdio`：远程工具传输与统一错误语义。
- `orchestration/*`：teams/workflow/composer/scheduler 的跨执行形态编排。
- `model/*`：provider 适配、能力探测与失败归一化。

### 2.4 Entropy Control（熵控制）

- 确定性边界：状态转移顺序、终态优先级、事件因果链。
- 失败收敛：fail-fast、回滚、可降级路径与默认值策略。
- 漂移阻断：命名治理 gate、语义等价强校验、行数预算与债务不扩张门禁。

## 3. Contract 与 Gate 映射

| 关注面 | Canonical 文档 | 主要 Gate / Suite |
| --- | --- | --- |
| 配置键、诊断字段与迁移映射 | `docs/runtime-config-diagnostics.md` | `scripts/check-docs-consistency.*`、`go test ./runtime/config ./runtime/diagnostics` |
| 模块依赖边界 | `docs/runtime-module-boundaries.md` | `scripts/check-runtime-boundaries.*` |
| 主线 contract -> 测试映射 | `docs/mainline-contract-test-index.md` | `go test ./tool/contributioncheck -run TestMainlineContractIndexReferencesExistingTests` |
| 运行时能力路线与状态口径 | `docs/development-roadmap.md` | `scripts/check-docs-consistency.*`（pre-1 governance 校验） |
| 命名治理（语义命名/历史编号） | `openspec/governance/semantic-labeling-*.{yaml,csv}` | `scripts/check-semantic-labeling-governance.*` |
| 大文件拆分治理 | `openspec/governance/go-file-line-budget-*.{env,csv}` | `scripts/check-go-file-line-budget.*`、`scripts/check-go-split-semantic-equivalence.*` |

## 4. 维护约定

- 新增运行时架构叙述时，优先更新本页，再由其他文档链接引用。
- 若 contract/gate 路径发生变化，必须在同一 PR 同步更新本页映射表。
- 任何“语义不变重构”都应附带对应强校验结果（Run/Stream parity、replay、受影响 contract suites）。
