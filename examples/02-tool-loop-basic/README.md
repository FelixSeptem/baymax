# 02-tool-loop-basic

## Pattern

- Tool Call
- Sequential

## 学习目标

- 理解本地工具注册与调用反馈闭环。
- 观察模型发起 tool call 后的二次推理收敛路径。

## 运行

```bash
go run ./examples/02-tool-loop-basic
```

## 预期输出

```text
tool result received
```

## 边界（本示例不覆盖）

- MCP 远程工具
- 高并发 fanout
- 异步任务编排
