# 03-mcp-mixed-call

## Pattern

- Tool Call
- Routing

## 学习目标

- 展示 local tool 与 MCP proxy 的混合调用入口。
- 理解“同一轮 loop 内按工具名路由不同执行路径”的基础方式。

## 运行

```bash
go run ./examples/03-mcp-mixed-call
```

## 预期输出

```text
mixed local+mcp done
```

## 边界（本示例不覆盖）

- 真实 MCP 传输连接（当前为最小 proxy 风格）
- 重连与故障注入
- 高级路由策略（成本/置信度驱动）
