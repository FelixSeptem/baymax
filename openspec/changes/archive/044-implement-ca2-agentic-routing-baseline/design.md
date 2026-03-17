## Context

当前 CA2 已支持 `routing_mode=rules|agentic` 配置校验，但 `agentic` 在装配阶段仍返回 `ErrAgenticRoutingNotReady`，导致配置能力与运行能力不一致。该缺口已进入用户可见面，且会阻断后续基于路由质量的功能增强（如更细粒度的 Stage2 触发策略）。

本次设计覆盖 `context/assembler`、`runtime/config`、`runtime/diagnostics` 与事件映射链路，目标是在不引入灰度系统、不改变现有 Stage2 provider 语义的前提下，交付 callback 驱动的 agentic 路由基线。

## Goals / Non-Goals

**Goals:**
- 在 `routing_mode=agentic` 时支持宿主 callback 决策。
- callback 异常路径按 `best_effort` 回退到 `rules`，不阻断主流程。
- 补齐最小路由诊断字段：`stage2_router_mode`、`stage2_router_decision`、`stage2_router_reason`、`stage2_router_latency_ms`、`stage2_router_error`。
- Run/Stream 在等价输入与配置下保持路由决策语义等价。

**Non-Goals:**
- 不内建 LLM 路由器或外部策略引擎。
- 不新增 rollout / 百分比灰度配置。
- 不调整 Stage2 provider SPI、CA3 压力控制与现有 stage policy 基线语义。

## Decisions

### 1) 路由扩展采用 callback 接口，而非内建策略组件
- 方案：在 `context/assembler` 增加 agentic router callback 接口（通过 Option 注入），输入为路由上下文，输出为 `run_stage2` 决策与 reason。
- 原因：满足 library-first，避免强绑定外部依赖，并允许业务侧按域知识实现决策。
- 备选：
  - 内建 heuristic：实现快，但泛化能力弱，后续仍需 callback 扩展。
  - 内建 LLM router：能力强但成本/稳定性/依赖复杂度显著提高，不符合本次“基线”目标。

### 2) callback 失败语义固定为 `best_effort -> rules fallback`
- 方案：callback 未注册、超时、返回错误、返回非法决策时，统一记录路由错误并立即回退规则路由。
- 原因：已确认不引入阻断语义，优先保障主路径可用性与向后兼容。
- 备选：
  - fail-fast 终止：安全但会扩大可用性风险，与本次需求冲突。

### 3) 诊断字段最小闭环，避免过度扩展
- 方案：仅新增 5 个路由字段，字段语义稳定并保持增量兼容。
- 原因：当前目标是“可观测可排障”，不做完整决策画像系统。
- 备选：
  - 增加更多路由上下文字段：可观测更丰富，但会提高 schema 演进成本与噪声。

### 4) Run/Stream 强制语义等价，允许事件时序差异
- 方案：以路由结果语义（mode/decision/reason/error）为等价判定核心，不强制事件时间戳一致。
- 原因：符合现有契约测试策略，便于稳定落地并避免实现耦合。

## Risks / Trade-offs

- [Risk] callback 实现质量参差导致路由抖动  
  → Mitigation: 失败统一回退 rules；新增契约测试覆盖 callback success/error/timeout/invalid。

- [Risk] callback 超时增加装配延迟  
  → Mitigation: 增加 agentic 路由超时配置并写入 `stage2_router_latency_ms`，便于调优。

- [Risk] 诊断字段扩展影响既有消费者  
  → Mitigation: 全部字段为 additive，保持缺省可空与向后兼容。

## Migration Plan

1. 增量发布 callback 接口与 `routing_mode=agentic` 执行路径（默认仍为 `rules`）。  
2. 增量发布配置字段与校验逻辑，确保热更新失败可回滚。  
3. 发布路由诊断字段与 Run/Stream 契约测试。  
4. 文档同步更新 `runtime-config-diagnostics`、`v1-acceptance` 与 roadmap。  

回滚策略：关闭 `routing_mode=agentic` 或移除 callback 注册，即回退到既有 `rules` 路径。

## Open Questions

- 无阻断级待确认项。
