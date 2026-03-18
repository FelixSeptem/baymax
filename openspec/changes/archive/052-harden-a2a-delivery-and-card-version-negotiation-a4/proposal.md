## Why

`a2a-minimal-interoperability` 已经建立了最小互联闭环，但当前交付通道与 Agent Card 版本语义仍偏“单路径假设”，在跨进程长链路和异构 peer 版本并存场景下存在稳定性与兼容性风险。现在补齐 delivery 模式治理与版本协商契约，可以把 A2A 从“可用”推进到“可运营”。

## What Changes

- 为 A2A 结果交付增加标准化 delivery 模式与降级语义：`callback|sse`，并支持协商失败后的确定性 fallback。
- 为 Agent Card 增加 schema 版本协商规则（major/minor），并输出归一化协商结果与失败原因。
- 扩展 A2A timeline reason 与 diagnostics 字段，覆盖订阅、重连、回退、版本不匹配等关键路径。
- 在 `runtime/config` 增加 A2A delivery/negotiation 配置与 fail-fast 校验，保持 `env > file > default` 与热更新回滚语义。
- 收敛 A2A 与 MCP 边界：A2A 负责 peer 协作与交付协商，MCP 保持工具调用语义，不承载 peer 协商状态。

## Capabilities

### New Capabilities
- `a2a-delivery-and-version-negotiation`: 规范 A2A delivery mode、版本协商、fallback 与错误语义。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 A2A delivery/negotiation 配置与诊断字段契约。
- `action-timeline-events`: 增加 A2A delivery/negotiation 事件 reason 与关联字段契约。
- `runtime-module-boundaries`: 增加 A2A delivery/negotiation 的模块边界与写入口约束。

## Impact

- 影响代码：
  - `a2a/*`（delivery 协商、version 协商、retry/reconnect 策略）
  - `runtime/config`（`a2a.delivery.*`、`a2a.card.version_*` 配置与校验）
  - `observability/event`、`runtime/diagnostics`（新增 additive 字段）
  - `core/types`（协商结果与归一化错误码 DTO）
- 影响测试：
  - A2A delivery 协商与 fallback 契约测试
  - A2A 版本协商兼容矩阵测试（major/minor）
  - Run/Stream 等价与 replay 幂等测试
  - A2A+MCP 组合边界回归测试
- 影响文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/runtime-module-boundaries.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
