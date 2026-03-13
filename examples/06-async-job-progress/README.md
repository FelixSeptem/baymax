# 06 Async Job Progress

Pattern: `Map-Reduce (approx)` + `Parallel`

该示例演示异步任务拆分、进度回传与聚合收敛，并输出结构化事件。

## Run

```bash
go run ./examples/06-async-job-progress
```

## What To Observe

- `job.started` / `job.progress` / `job.completed` 结构化事件
- 并行任务完成顺序与提交顺序不一致
- 最终聚合指标（平均延迟）

## Out Of Scope

- 不覆盖分布式 job queue
- 不覆盖持久化恢复
