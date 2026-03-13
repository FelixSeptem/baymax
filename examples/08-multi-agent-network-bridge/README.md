# 08 Multi-Agent Network Bridge

Pattern: `Multi-Agent` + `Structure` (Network)

该示例演示 agent 间通过 **HTTP + JSON-RPC 2.0** 进行最小请求/响应通信。

## Run

```bash
go run ./examples/08-multi-agent-network-bridge
```

## What To Observe

- JSON-RPC 2.0 请求字段：`jsonrpc/id/method/params`
- JSON-RPC 2.0 响应字段：`jsonrpc/id/result/error`
- 结构化事件展示 client/server 收发路径

## Out Of Scope

- 不覆盖 MCP 完整生命周期方法集
- 不覆盖鉴权、重试、断线重连
