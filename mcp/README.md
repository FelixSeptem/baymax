# mcp 组件说明

## 功能域

`mcp` 负责远程工具调用能力，提供 HTTP/STDIO 两种传输并统一策略语义：

- 连接与调用：`mcp/http`、`mcp/stdio`
- 配置策略：`mcp/profile`
- 重试语义：`mcp/retry`
- 调用摘要模型：`mcp/diag`

## 架构设计

整体采用“传输实现 + 语义子域 + internal 共享骨架”：

- 传输层：`http` / `stdio` 客户端实现 `types.MCPClient`
- 语义层：profile/retry/diag 只定义可复用策略和模型
- internal 层：
  - `mcp/internal/reliability`：统一重试执行框架与策略解析
  - `mcp/internal/observability`：事件与诊断桥接

两个传输实现均支持消费 `runtime/config.Manager` 的动态策略快照。

## 关键入口

- `http/client.go`
- `stdio/client.go`
- `profile/profile.go`
- `retry/retry.go`
- `diag/diag.go`

## 边界与依赖

- `mcp/internal/*` 仅供 `mcp/*` 内部复用，其他域禁止依赖。
- `runtime/*` 不反向依赖传输实现，保持配置域与传输域解耦。
- 传输层发射标准事件与调用记录，不直接破坏 RuntimeRecorder 单写口径。
