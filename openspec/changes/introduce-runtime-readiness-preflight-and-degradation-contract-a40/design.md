## Context

Baymax 当前已具备多代理主链路与契约门禁，但运行准备度判断仍分散在各模块初始化逻辑中。调用方缺少统一入口来回答三个问题：

- 当前是否可安全启动（`ready`）？
- 是否可启动但存在退化（`degraded`）？
- 是否必须阻断（`blocked`）？

在 `0.x` 阶段继续推进“可回归收敛”时，readiness 预检是连接“能力存在”与“可用可判定”的关键一环。

## Goals / Non-Goals

**Goals:**
- 提供库级 runtime readiness preflight API，统一返回状态分级与结构化 findings。
- 引入 `runtime.readiness.*` 配置域并纳入启动/热更新 fail-fast + 原子回滚语义。
- 固化 `strict` 策略：`strict=true` 时将 `degraded` 视为 `blocked`。
- 将 readiness 判定结果纳入 diagnostics additive 字段与质量门禁回归链路。

**Non-Goals:**
- 不引入平台化控制面、远程运维探针系统、SLO 编排面板。
- 不做全量网络依赖探活（默认 `remote_probe_enabled=false`）。
- 不改变既有 Run/Stream 状态机与业务终态裁决逻辑。

## Decisions

### 1) 结果模型采用三态分级：`ready|degraded|blocked`
- 决策：使用三态替代布尔结果，提升可解释性与自动化决策能力。
- 原因：仅有 pass/fail 无法表达“可运行但退化”的中间态。
- 备选：仅 `ok|fail`。拒绝原因：无法支撑 strict 策略和降级可观测。

### 2) Findings 采用结构化规范字段
- 决策：finding 统一字段 `code/domain/severity/message/metadata`。
- 原因：便于跨模块聚合、去重、门禁断言和文档映射。
- 备选：自由文本日志。拒绝原因：不可测试、不可稳定回归。

### 3) strict 策略仅改变判定，不改变底层降级路径
- 决策：`strict=false` 允许 `degraded` 继续运行；`strict=true` 将 `degraded` 结果提升为 `blocked`。
- 原因：保证策略清晰，避免把策略开关与底层 fallback 实现耦合。
- 备选：strict 模式禁用所有 fallback。拒绝原因：侵入性高，回归风险大。

### 4) 默认关闭远程探活，保持离线可执行
- 决策：`runtime.readiness.remote_probe_enabled=false` 默认不做网络探测，仅检查本地可判定状态。
- 原因：符合 library-first 和离线开发场景；避免把环境噪声引入主流程。
- 备选：默认开启远端探测。拒绝原因：不稳定、易误判、增加外部依赖。

### 5) readiness 入口放在 runtime 主域，composer 仅透传
- 决策：readiness 核心判定收敛在 runtime/config + diagnostics 领域，composer 提供可选透传摘要。
- 原因：保持模块边界，避免编排层承担配置/健康判定职责。
- 备选：仅在 composer 暴露 readiness。拒绝原因：限制复用，且不利于非 composer 集成。

## Risks / Trade-offs

- [Risk] 预检规则过严导致误阻断  
  -> Mitigation: 默认 `strict=false`，并将阻断升级交给调用方显式启用。

- [Risk] findings 码表持续膨胀、语义漂移  
  -> Mitigation: 维护 canonical code 集合与 gate drift 检测。

- [Risk] readiness 与现有 fallback 逻辑不一致  
  -> Mitigation: 将 fallback 状态纳入最小必检集，并加入 parity/replay 测试。

- [Risk] 文档与代码状态不同步触发 gate 失败  
  -> Mitigation: readiness 字段纳入 docs consistency 与 mainline contract index 同步要求。

## Migration Plan

1. 定义 readiness 结果模型与 finding 结构、状态分级规则。
2. 扩展 `runtime/config`：新增 `runtime.readiness.enabled|strict|remote_probe_enabled`，补齐校验与热更新回滚。
3. 在 runtime 初始化/管理层实现 readiness 聚合检查入口。
4. 扩展 diagnostics additive 字段，记录 readiness 状态、finding 总量与主要原因码。
5. 在 composer 增加 readiness 透传入口（若启用 composer 集成路径）。
6. 补齐 unit/integration 契约测试：strict 矩阵、fallback 可见性、replay idempotency、Run/Stream 等价。
7. 接入 `check-quality-gate.*` 并更新 README/roadmap/runtime-config-diagnostics/mainline index。

回滚策略：
- 通过 `runtime.readiness.enabled=false` 关闭 readiness 输出；
- 保持 `strict=false` 可快速解除阻断策略；
- 新字段均为 additive，不影响现有调用路径。

## Open Questions

无阻塞项，按推荐值推进：
- `enabled=true`
- `strict=false`
- `remote_probe_enabled=false`
