# MCP Runtime Reliability Profiles

更新时间：2026-03-11

## 目标

为 `mcp/http` 与 `mcp/stdio` 提供一致的可靠性策略入口，减少散装参数导致的行为漂移。

## Profiles

- `dev`
  - 快速反馈优先，重试最少
  - 适合本地联调与故障快速暴露
- `default`
  - 兼顾稳定性和吞吐的通用配置
- `high-throughput`
  - 提高并发与队列容量，背压倾向拒绝
  - 适合高吞吐、可容忍部分请求快速失败场景
- `high-reliability`
  - 更长超时与更高重试次数
  - 适合链路波动较大、恢复成功率优先场景

## 默认值（按 profile 解析后）

- `call_timeout`
- `retry`
- `backoff`
- `queue_size`
- `backpressure`
- `read_pool_size`
- `write_pool_size`

注：profile 默认值可被显式配置覆盖。
配置来源优先级：`env > file > default`。

## 统一语义

`mcp/http` 与 `mcp/stdio` 对齐以下行为：
- 重试停止条件（不可重试错误立即 fail-fast）
- backoff 计算
- 错误分类与事件字段
- 最近 N 次 MCP 调用诊断摘要字段
- profile 参数读取来源统一为运行时配置快照（`runtime/config.Manager`）
- 共享执行骨架统一为 `mcp/internal/*`（internal-only），transport 仅保留协议差异逻辑

## 分层边界

- `mcp/internal/reliability`：重试/超时/backoff/fail-fast 执行骨架
- `mcp/internal/observability`：事件发射与诊断映射桥接
- `mcp/http`、`mcp/stdio`：连接管理、池化/心跳、协议专属请求处理

说明：`mcp/internal/*` 仅供 `mcp` 子包复用，不对外暴露。

## 调优建议

- 首次上线使用 `default`。
- 高并发业务优先尝试 `high-throughput`，并观察 queue reject 比例。
- 网络抖动较大环境优先尝试 `high-reliability`，并关注延迟变化。
- 开发环境建议 `dev`，便于快速暴露重试掩盖的问题。

## 故障语义

- 心跳失败：触发重连流程，记录重连诊断字段。
- 瞬时错误：按 retry 策略重试。
- 不可重试错误：立即终止重试并返回失败。

## 诊断摘要字段（Recent N）

- `time`
- `transport`
- `profile`
- `call_id`
- `tool`
- `latency_ms`
- `retry_count`
- `reconnect_count`
- `error_class`
