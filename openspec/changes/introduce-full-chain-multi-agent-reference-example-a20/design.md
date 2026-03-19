## Context

当前仓库已经具备多代理主干能力与分段示例（`07/08`），但缺少一个“全链路单入口”示例来展示编排域关键模块如何组合运行。对外传播与新接入方通常需要同时理解 team/workflow/a2a/scheduler/recovery 的协作语义，仅依赖分段示例会增加理解成本。

A20 聚焦示例与文档层收敛，不改动运行时核心行为。

## Goals / Non-Goals

**Goals:**
- 提供一个可直接运行的全链路参考示例，覆盖 `team + workflow + a2a + scheduler + recovery`。
- 示例默认使用 in-memory A2A 与本地可运行依赖，保证开箱即用。
- 同时覆盖 Run/Stream 双路径，并展示 async + delayed + recovery 最小组合路径。
- 将示例 smoke 校验纳入 quality gate 阻断路径，避免示例长期漂移。

**Non-Goals:**
- 不新增平台化控制面或远程部署模板。
- 不引入新协议或变更现有运行时契约语义。
- 不替代现有分段示例（`01-08`）。

## Decisions

### 1) A20 仅做示例、文档与 smoke 校验，不改运行时行为
- 方案：所有变更限定在 `examples/*`、文档与门禁脚本。
- 原因：降低与 A19 并行实施冲突风险，确保可独立交付。
- 备选：在示例中顺带调整运行时接口。拒绝原因：扩大范围并增加回归风险。

### 2) 默认走 in-memory A2A，网络桥接作为可选扩展
- 方案：全链路示例默认使用内存 server/client 路径。
- 原因：零外部依赖、CI 可稳定复现。
- 备选：默认 HTTP 网络桥接。拒绝原因：环境依赖和波动更高。

### 3) Run 与 Stream 双路径同一示例内并列展示
- 方案：同一示例提供 `Run` 和 `Stream` 两条最小可观察路径。
- 原因：保持语义对照，减少用户在多个示例之间跳转。
- 备选：仅保留 Run。拒绝原因：无法体现双路径语义一致性。

### 4) 示例强制包含 async + delayed + recovery 最小组合
- 方案：全链路流程中至少包含一次 async 回报、一次 delayed 调度、一次 recovery 恢复路径。
- 原因：覆盖目前多代理最关键协作语义组合。
- 备选：拆为多个独立场景。拒绝原因：全链路感知不足。

### 5) 示例 smoke 作为 quality gate 阻断项
- 方案：在 `check-quality-gate.sh/.ps1` 接入示例 smoke。
- 原因：示例属于外部入口，必须与代码演进同步。
- 备选：文档约定手工运行。拒绝原因：无法持续防漂移。

## Risks / Trade-offs

- [Risk] 全链路示例过于复杂导致可读性下降  
  → Mitigation: 主路径最小化，复杂分支放到注释和 README “扩展路径”。

- [Risk] 示例 smoke 增加 CI 时长  
  → Mitigation: 仅保留最小断言，避免长时 benchmark 或重型集成依赖。

- [Risk] A19 并行实施导致脚本改动冲突  
  → Mitigation: A20 仅在 gate 接入点做最小变更并保持可重入脚本结构。

## Migration Plan

1. 新增 `examples/09-multi-agent-full-chain-reference` 目录与最小可运行入口。
2. 在示例中接入 team/workflow/a2a/scheduler/recovery 组合路径，并输出关键观测点。
3. 增补示例 README，明确运行命令、配置与预期输出。
4. 增加示例 smoke 测试或脚本，并接入 `check-quality-gate.*`。
5. 更新顶层 README、roadmap 与 mainline index 映射。

回滚策略：
- 如示例 smoke 稳定性不达标，可先回滚 gate 接入，保留示例主体。
- 回滚不影响 runtime 核心行为与既有 contract suites。

## Open Questions

- 当前无阻塞性开放问题，按推荐值固定：
  - 仅示例/文档/smoke 范围
  - 默认 in-memory A2A
  - Run/Stream 双路径
  - async + delayed + recovery 组合覆盖
  - smoke 纳入阻断 gate
