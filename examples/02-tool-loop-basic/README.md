# 02-tool-loop-basic

## Pattern

- Tool Call
- Sequential

## 学习目标

- 理解本地工具注册与调用反馈闭环。
- 观察模型发起 tool call 后的二次推理收敛路径。
- 理解 ReAct 模式下 Run/Stream 共享 loop 语义（step-boundary dispatch + feedback）。

## ReAct 配置（最小）

```yaml
runtime:
  react:
    enabled: true
    max_iterations: 12
    tool_call_limit: 64
    stream_tool_dispatch_enabled: true
    on_budget_exhausted: fail_fast
```

说明：
- 本示例主程序演示 Run 路径的工具闭环。
- Stream 等价语义由集成契约 `integration/react_loop_parity_contract_test.go::TestReactLoopRunStreamParityIntegrationContract` 覆盖。

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
