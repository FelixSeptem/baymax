## Context

当前 A2A 基线已具备 `submit/status/result` 生命周期与最小能力发现，但在以下两类场景仍有空白：
- 交付链路差异：不同 peer 对 callback 或 SSE 的支持能力不同，缺少统一协商与回退语义；
- Card 版本并存：在灰度升级期，peer 之间 schema minor 演进与 major 断裂没有统一处理口径。

仓库已有 shared multi-agent contract（identifier/reason/status 统一口径）与 single-writer 观测约束，本设计在该约束下增量扩展，不引入新写入口或旁路状态存储。

## Goals / Non-Goals

**Goals:**
- 交付 A2A delivery mode 协商与 fallback 语义（`callback|sse`）。
- 交付 Agent Card 版本协商规则与归一化错误映射。
- 交付 A2A delivery/version 的 timeline 与 diagnostics additive 字段。
- 保持 Run/Stream 语义等价和 replay 幂等行为不变。
- 保持 A2A/MCP 边界稳定，不引入职责重叠。

**Non-Goals:**
- 不引入 control plane、中心注册、租户治理与 RBAC。
- 不实现复杂跨地域路由与动态负载均衡。
- 不改写 MCP 传输栈。
- 不在本期引入多种 SSE 子协议方言适配。

## Decisions

### 1) Delivery mode 协商采用“显式优先 + 确定性降级”
- 方案：请求端可显式声明优先模式；若 peer 不支持则按固定顺序降级（`sse -> callback` 或配置指定顺序）。
- 原因：可解释、可测试，便于诊断回放。
- 备选：自动探测后隐式切换。拒绝原因：行为不透明、排障困难。

### 2) Card 版本协商采用 `strict major + compatible minor`
- 方案：major 不一致直接拒绝；major 一致且 peer minor >= min_supported_minor 则视为兼容。
- 原因：兼容规则简单且能覆盖渐进升级窗口。
- 备选：完全语义版本范围表达式。拒绝原因：实现复杂度高、首期收益有限。

### 3) 失败语义统一映射为 A2A 归一化 reason/error layer
- 方案：定义 delivery 与 version 的最小错误码集合（如 `a2a.delivery_unsupported`、`a2a.version_mismatch`），并映射到 runtime taxonomy。
- 原因：保证跨模块可观测一致。
- 备选：保留原始 transport 错误直出。拒绝原因：跨协议比较困难。

### 4) 观测仍走 single-writer
- 方案：A2A delivery/version 事件统一进入 `observability/event.RuntimeRecorder`，诊断聚合依赖既有 idempotency 机制。
- 原因：避免并行事实源，延续现有治理模式。
- 备选：A2A 自建 store。拒绝原因：增加系统复杂度并破坏统一查询口径。

## Risks / Trade-offs

- [Risk] SSE 重连与 callback 重试叠加导致时序复杂  
  → Mitigation: 统一 reason namespace + sequence 语义，并补 replay 幂等测试。

- [Risk] 版本协商策略过严导致短期可用性下降  
  → Mitigation: 提供 `compatibility_policy` 可配置项，并在 diagnostics 输出协商失败细节。

- [Risk] delivery fallback 逻辑引入 Run/Stream 语义差异  
  → Mitigation: 增加等价输入下 Run/Stream 契约测试，校验终态与关键字段一致。

## Migration Plan

1. 在 `runtime/config` 增加 A2A delivery/version 配置字段与校验（默认兼容现有路径）。  
2. 在 A2A client/server 路径实现协商与 fallback 状态机。  
3. 接入 timeline/diagnostics additive 字段与 reason 语义。  
4. 补齐契约测试并更新文档，纳入主干索引与门禁脚本。  

回滚策略：
- 关闭新增 delivery mode（保留 callback 单路径）；
- 关闭版本协商强校验（降级到最小兼容模式）；
- 不影响既有 Run/MCP/skill 核心语义。

## Open Questions

- 首期是否允许 `callback-only` 的 strict 模式（禁止自动 fallback）？
- SSE 重连预算是否与 callback retry 共享配额，还是分别配置？
