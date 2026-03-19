## Context

按本提案前提，A18（统一 run/team/workflow/task 查询契约）已完成并进入稳定阶段。当前剩余 P1 缺口主要是“多代理主链路性能基线纳入 CI 阻断”。现状中，仓库已有 `docs/performance-policy.md` 与 CA4 专项 benchmark gate，但对多代理主链路（A11-A17）缺少统一基线比较与失败语义，导致性能退化只能依赖人工发现。

约束：
- 保持 `library-first`，不引入平台化性能服务。
- 保持现有质量门禁结构，新增 gate 走 `scripts/check-quality-gate.*` 主路径。
- 保持跨平台一致性（Shell + PowerShell）。

## Goals / Non-Goals

**Goals:**
- 建立多代理主链路 benchmark 矩阵与统一基线文件。
- 提供可重复、可阻断的回归脚本，支持本地与 CI 一致执行。
- 固化默认参数与阈值，避免评审口径漂移。
- 将性能覆盖映射到主干索引，保证可追溯。

**Non-Goals:**
- 不引入外部时序数据库或性能平台。
- 不做自动化机器归一化/硬件校正。
- 不替代现有 contract gate；本提案仅补性能维度 gate。

## Decisions

### 1) 基准覆盖矩阵采用“主链路四路径”最小闭环
- 方案：新增并维护四类 benchmark：同步调用、异步回报、延后调度、恢复重放。
- 原因：对应 A11-A17 主链路，覆盖收益最高且可维护。
- 备选：一次性覆盖全部子模块 benchmark。拒绝原因：噪声高、维护成本大。

### 2) 回归判定采用相对百分比 + 三指标
- 方案：统一比较 `ns/op`、`p95-ns/op`、`allocs/op`。
- 默认阈值：
  - `ns/op` 退化 <= `8%`
  - `p95-ns/op` 退化 <= `12%`
  - `allocs/op` 退化 <= `10%`
- 原因：在稳定性与敏感度之间平衡，且可与现有 CA4 gate 口径兼容。
- 备选：仅比较 `ns/op`。拒绝原因：无法识别尾延迟与内存分配回归。

### 3) 默认执行参数固定为 `benchtime=200ms`、`count=5`
- 方案：脚本默认参数固定，可被环境变量覆盖。
- 原因：降低样本波动并保持 CI 成本可控。
- 备选：`count=1` 或 `benchtime=50ms`。拒绝原因：结果噪声较大，误报风险高。

### 4) 基线与参数异常采用 fail-fast
- 方案：缺失基线、阈值非法、benchmark 解析失败均立即报错并阻断 gate。
- 原因：性能门禁属于质量约束，不允许静默跳过。
- 备选：warn-only。拒绝原因：无法形成可靠保护。

### 5) gate 接入走现有 quality-gate 主路径
- 方案：在 `scripts/check-quality-gate.sh/.ps1` 接入 `check-multi-agent-performance-regression.*`。
- 原因：保持单入口，降低贡献者认知成本。
- 备选：新增独立 CI job。拒绝原因：会增加维护面与配置复杂度。

## Risks / Trade-offs

- [Risk] benchmark 在共享 runner 上波动引发误报  
  → Mitigation: 固定默认参数、支持 baseline 显式更新流程、文档要求在稳定环境重采样。

- [Risk] 新 gate 增加 CI 时长  
  → Mitigation: 仅覆盖四条主链路 benchmark，控制 benchtime 与 count 上限。

- [Risk] 阈值过严阻碍迭代  
  → Mitigation: 阈值可配置且需通过 PR 明确更新基线与理由。

## Migration Plan

1. 在 `integration/benchmark_test.go` 增加多代理主链路 benchmark 场景。
2. 新增 `scripts/check-multi-agent-performance-regression.sh/.ps1` 与基线 env 文件。
3. 将新脚本接入 `scripts/check-quality-gate.sh/.ps1`。
4. 更新 `docs/performance-policy.md`、`docs/mainline-contract-test-index.md` 与 roadmap。
5. 在 CI 路径验证 gate 阻断语义与本地一致性。

回滚策略：
- 若新 gate 出现不可接受误报，可先在同一 PR 回滚 quality-gate 接入点；
- 保留 benchmark 与基线文件，不影响已有 contract gate 主链路。

## Open Questions

- 当前无阻塞性开放问题；阈值与默认参数按本提案推荐值冻结。
