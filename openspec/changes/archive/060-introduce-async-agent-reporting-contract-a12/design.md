## Context

在 A11 之后，同步调用语义已收敛，但异步协作仍主要依赖“提交后轮询/等待”：
- 服务端可异步执行任务；
- 客户端可通过 `WaitResult` 回收终态；
- callback 仍绑定在等待路径内，缺少提交后独立回报通道。

这会导致三个问题：
- 编排层要么阻塞等待，要么重复实现轮询回收；
- 异步回报失败重试与去重逻辑无法集中治理；
- recovery 回放路径下，回报语义缺少统一幂等键。

A12 目标是在不平台化、不引入外部消息系统的前提下，为多代理路径补齐“异步提交 + 独立回报”契约。

## Goals / Non-Goals

**Goals:**
- 提供统一 `SubmitAsync` 与 `ReportSink` 契约，支持提交即返回。
- 回报通道与 `WaitResult` 解耦，形成独立投递生命周期。
- 统一回报语义：至少一次投递、幂等去重、失败重试、可观测。
- 支持最小 sink 实现：in-memory channel 与 callback sink。
- 覆盖 composer/scheduler/a2a 关键路径并补契约测试。

**Non-Goals:**
- 不引入外部 MQ/Kafka/NATS 依赖。
- 不引入平台控制面或多租户分发系统。
- 不替代同步路径；同步调用仍保留为稳定基线。
- 不在本提案内引入业务级定时调度（A13）。

## Decisions

### 1) 新增异步调用契约而非扩展 WaitResult
- 方案：定义 `SubmitAsync` 返回任务句柄；回报通过 `ReportSink` 独立投递。
- 原因：从语义上明确“等待”和“回报”是两条路径，减少接口歧义。
- 备选：继续复用 `WaitResult + callback`。拒绝原因：无法实现提交后真正非阻塞回报。

### 2) 默认保持关闭，显式启用异步回报
- 方案：增加配置开关，默认关闭异步回报通道。
- 原因：保持兼容，避免现有调用方行为突变。
- 备选：默认开启。拒绝原因：会改变现有调用路径和负载模型。

### 3) 回报语义采用“至少一次 + 去重幂等”
- 方案：回报失败可重试；使用幂等键防止重复回报影响聚合。
- 原因：兼顾可靠性与实现复杂度，适配 recovery 回放。
- 备选：精确一次。拒绝原因：实现成本高且需要外部一致性基础设施。

### 4) 回报失败不改变业务终态
- 方案：任务业务终态与回报投递终态分离，投递失败仅记录诊断/告警。
- 原因：保持主业务语义稳定，避免“回报故障反向污染执行结果”。
- 备选：回报失败反向标记任务失败。拒绝原因：会破坏既有终态契约。

### 5) 幂等键采用可重放稳定组合
- 方案：`run_id + task_id + attempt_id + terminal_status + outcome_key(optional)`。
- 原因：覆盖调度接管和 recovery 重放场景，保证去重稳定。
- 备选：仅 `task_id`。拒绝原因：无法区分 attempt 与终态变化。

### 6) 先支持两类内置 sink
- 方案：内置 `channel sink` 与 `callback sink`；接口保留扩展点。
- 原因：满足 lib-first 最小可用能力并便于宿主接入。
- 备选：一次性支持 HTTP/webhook sink。拒绝原因：引入外部 IO 复杂性与安全面扩展。

### 7) 回报重试策略采用指数退避 + 抖动
- 方案：有界重试，指数退避 + jitter。
- 原因：与现有调度治理策略一致，降低瞬时重试风暴。
- 备选：固定间隔重试。拒绝原因：在高并发失败场景容易形成脉冲。

## Risks / Trade-offs

- [Risk] 异步通道引入额外状态机复杂度  
  → Mitigation: 分离“执行终态”和“回报终态”，并以 contract tests 固化边界。

- [Risk] 重试与去重策略实现不一致导致重复统计  
  → Mitigation: 统一幂等键生成逻辑并在 diagnostics/replay 测试中校验。

- [Risk] 默认关闭导致接入方误判能力可用  
  → Mitigation: 文档与配置校验明确开关状态，并在 run summary 暴露启用标记。

- [Risk] recovery 重放与异步回报互相干扰  
  → Mitigation: 回报事件同样走 single-writer 路径，去重优先于聚合。

## Migration Plan

1. 增加异步回报配置域与校验（默认关闭）。  
2. 增加 `SubmitAsync` + `ReportSink` 抽象与内置 sink。  
3. 接入 A2A/composer/scheduler 关键路径并保留同步兼容路径。  
4. 增加 timeline/diagnostics additive 字段与 reason taxonomy。  
5. 补齐 contract tests 与 shared gate。  
6. 更新 README/诊断文档/索引与 roadmap 状态。  

回滚策略：
- 关闭异步回报开关，回退到 A11 同步基线路径；
- additive 字段保留，不破坏旧消费者。

## Open Questions

- 当前无阻塞问题；采用默认关闭与内置双 sink 作为首版范围。
