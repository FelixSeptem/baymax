# 01-chat-minimal

## Pattern

- Sequential

## 学习目标

- 运行最小单轮 Agent 调用链路。
- 理解 `runner.Run` 的基础输入输出结构。

## 运行

```bash
go run ./examples/01-chat-minimal
```

## 预期输出

```text
hello from 01-chat-minimal
```

## 边界（本示例不覆盖）

- 工具调用与多轮 loop
- MCP 调用
- 并发/背压/重试策略
