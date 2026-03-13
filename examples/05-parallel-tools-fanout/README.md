# 05 Parallel Tools Fanout

Pattern: `Parallel`

该示例演示单轮内多工具并发扇出执行，并通过 runner 收敛结果。

## Run

```bash
go run ./examples/05-parallel-tools-fanout
```

## What To Observe

- 三个 `local.*` 工具在同一轮被触发
- 最终输出汇总为单次 run 的 final answer
- stdout 可看到结构化 event（JSON）

## Out Of Scope

- 不覆盖重试策略调优
- 不覆盖跨进程并行执行
