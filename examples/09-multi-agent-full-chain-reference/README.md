# 09 Multi-Agent Full-Chain Reference

Pattern: `Multi-Agent` + `Full Chain`

该示例在单个可运行入口中串联：
- `teams`
- `workflow`
- `a2a`（默认 in-memory）
- `scheduler`（含 delayed-dispatch）
- `recovery`（file backend，本地临时目录）

## Run

默认同时执行 Run + Stream 双路径，并输出 async/delayed/recovery 检查点：

```bash
go run ./examples/09-multi-agent-full-chain-reference
```

可选模式：

```bash
go run ./examples/09-multi-agent-full-chain-reference -mode run
go run ./examples/09-multi-agent-full-chain-reference -mode stream
go run ./examples/09-multi-agent-full-chain-reference -mode both
```

## Required Markers

成功运行后应至少包含以下标记：
- `CHECKPOINT async_report_succeeded=true`
- `CHECKPOINT delayed_dispatch_claimed=true`
- `CHECKPOINT recovery_replayed=true`
- `CHECKPOINT correlation ...`
- `CHECKPOINT run_stream_aligned=true`（`-mode both`）
- `A20_RUN_TERMINAL ...`
- `A20_STREAM_TERMINAL ...`
- `A20_TERMINAL_SUMMARY=...`
- `A20_SUCCESS`

## Checkpoint Meanings

- `async_report_succeeded`：
  表示 A2A `SubmitAsync` 路径已成功收到异步回报。
- `delayed_dispatch_claimed`：
  表示 scheduler 的 `not_before` 任务在边界后可被 claim 并完成。
- `recovery_replayed`：
  表示 recovery-enable 路径完成一次 replay（`replayed_terminal_commits > 0`）。
- `correlation`：
  包含 run/task 级关联标识（workflow/teams run_id、async task_id、delayed task_id、recovery run_id），用于日志排障与 cross-path 对照。
- `run_stream_aligned`：
  表示 Run/Stream 在同一场景意图下核心聚合结果语义一致。

## Extension Guidance

- 默认模式为 in-memory A2A，不依赖外部网络服务。
- 若需要网络桥接，请参考 `examples/08-multi-agent-network-bridge` 的扩展方式。
- 该示例目标是最小可运行与排障定位，不覆盖生产级部署拓扑。
