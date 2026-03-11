# 04-streaming-interrupt

## Pattern

- Structure（中断与收敛语义）

## 学习目标

- 观察 `runner.Stream` 在超时/取消下的 fail-fast 行为。
- 理解流式 delta 在中断时的结果收敛方式。

## 运行

```bash
go run ./examples/04-streaming-interrupt
```

## 预期输出

输出类似：

```text
final="hello " err=context deadline exceeded
```

注：具体 `final` 片段长度取决于超时时间与调度时序。

## 边界（本示例不覆盖）

- 流式 tool call 增量处理
- UI 渲染与增量展示
- 复杂重试恢复策略
