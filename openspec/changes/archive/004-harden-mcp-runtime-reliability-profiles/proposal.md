## Why

当前 MCP `http/stdio` 路径仍存在重复的重试、重连、事件归一化与默认值定义，导致行为一致性与调优路径不稳定。现在需要以统一可靠性 profile 收敛 MCP 运行时，完成 R1 剩余项并为 R2 可运维能力提供稳定基线。

## What Changes

- 引入 MCP 运行时可靠性 profile（`dev/default/high-throughput/high-reliability`）并统一默认值文档。
- 将 `mcp/http` 与 `mcp/stdio` 的重试、backoff、重连、错误分类和事件归一化逻辑收敛到共享组件。
- 补齐 MCP 故障注入测试矩阵（transient error、reconnect storm、heartbeat timeout、queue/backpressure）。
- 增加 MCP 诊断输出（最近 N 次调用摘要与关键统计字段），便于线上排障。
- 校正 README 与 docs 的状态字段，避免归档状态与文档状态漂移。

## Capabilities

### New Capabilities
- `mcp-runtime-reliability-profiles`: MCP 可靠性 profile、共享重试/重连策略与统一事件语义。

### Modified Capabilities
- None.

## Impact

- Affected code:
  - `mcp/http/*`, `mcp/stdio/*`
  - 新增共享 runtime 组件（建议 `mcp/runtime/*`）
  - `integration/*`（MCP 故障注入测试与 benchmark）
  - `docs/*`, `README.md`
- Behavioral impact:
  - MCP 不同传输在错误分类、重试停止条件、事件字段上实现一致化
  - profile 化配置替代零散调参
- Operational impact:
  - 配置与诊断路径标准化，降低线上排障成本
