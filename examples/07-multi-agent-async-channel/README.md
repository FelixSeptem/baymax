# 07 Multi-Agent Async Channel

Pattern: `Multi-Agent` + `Structure`

该示例演示单进程内 coordinator/worker 通过 channel 进行异步协作。

## Run

```bash
go run ./examples/07-multi-agent-async-channel
```

## What To Observe

- coordinator 分发任务，worker 异步处理并回传
- 结构化事件展示 agent 生命周期（dispatch/start/collect/completed）
- 最终聚合结果由 coordinator 收敛输出

## Out Of Scope

- 不覆盖网络通信
- 不覆盖跨进程状态一致性
