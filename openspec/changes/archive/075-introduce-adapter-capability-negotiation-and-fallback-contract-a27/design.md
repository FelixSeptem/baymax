## Context

目前 A26 负责 adapter manifest 的“静态兼容”判定（版本范围、声明字段合法性），但在运行时请求实际能力时，仍缺统一协商模型：
- 请求能力集合与 adapter 声明能力集合如何匹配；
- `required` 与 `optional` 缺失如何处理；
- Run 与 Stream 路径如何保持语义等价；
- 失败/降级结果如何结构化诊断。

这会导致 adapter 在不同调用模式下呈现不一致行为，增加集成方排障成本。

## Goals / Non-Goals

**Goals:**
- 定义统一 capability negotiation contract（request vs declared capabilities）。
- 固化 `fail_fast`（默认）与 `best_effort` 策略语义与优先级。
- 固化 reason taxonomy，覆盖 required 缺失拒绝与 optional 降级。
- 保证 Run/Stream 协商结果语义等价。
- 将协商契约纳入 conformance harness 与 quality gate 阻断。

**Non-Goals:**
- 不引入外部 capability registry 或远程能力发现服务。
- 不实现跨请求长生命周期 negotiation cache。
- 不扩展到平台化控制面治理。

## Decisions

### 1) 默认策略采用 `fail_fast`
- 方案：当 requested required capability 无法满足时直接失败；仅 optional 缺失允许降级。
- 原因：保持契约清晰，减少隐式“成功但语义不完整”。
- 备选：默认 `best_effort`。拒绝原因：容易掩盖不兼容问题。

### 2) 允许请求级策略覆盖
- 方案：全局默认策略可被请求级字段覆盖（受白名单约束）。
- 原因：兼顾主流程安全与特定场景灵活性。
- 备选：只允许全局策略。拒绝原因：部分场景需要可控降级。

### 3) reason taxonomy 固化并可追踪
- 方案：至少包含：
  - `adapter.capability.missing_required`
  - `adapter.capability.optional_downgraded`
  - `adapter.capability.strategy_override_applied`
- 原因：便于契约测试与诊断面追踪。
- 备选：仅错误字符串。拒绝原因：不可稳定回归。

### 4) Run/Stream 共用同一 negotiation engine
- 方案：抽取统一协商组件，由 Run 与 Stream 调用同一逻辑。
- 原因：避免双实现漂移。
- 备选：分路径各自实现。拒绝原因：一致性难保证。

### 5) gate 集成延续现有结构
- 方案：新增 `check-adapter-capability-contract.*` 并并入 `check-quality-gate.*`。
- 原因：沿用已有阻断路径，降低维护成本。
- 备选：单独新增 CI job。拒绝原因：门禁分散。

## Risks / Trade-offs

- [Risk] 策略覆盖语义过于灵活导致不可预测  
  → Mitigation: 只允许 `fail_fast|best_effort`，并记录 override reason。

- [Risk] optional 降级路径过多导致测试矩阵膨胀  
  → Mitigation: 首期固定最小矩阵，按 capability 族分层扩展。

- [Risk] A26/A27 并行实施造成契约边界冲突  
  → Mitigation: A26 只管 manifest 静态校验，A27 只管运行时协商与降级。

## Migration Plan

1. 新增 negotiation 核心类型、策略解析与匹配逻辑。
2. 在 runtime adapter 调用入口接入统一 negotiation engine。
3. 增加协商诊断字段与 reason taxonomy 发射。
4. 扩展 A22 conformance 与 A23 脚手架测试骨架。
5. 新增 capability contract gate 并接入 quality gate。
6. 更新 README/roadmap/runtime-config-diagnostics/contract index。

回滚策略：
- 若 gate 噪声超预期，可先回滚 quality-gate 接入点；
- negotiation 逻辑可保留为非阻断路径，不影响主线可运行性。

## Open Questions

- 当前无阻塞问题；按推荐值冻结：
  - 默认 `fail_fast`
  - 允许请求级 `task` 字段策略覆盖
  - required 缺失拒绝、optional 缺失降级
  - 不引入协商缓存
