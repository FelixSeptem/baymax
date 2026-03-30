# Examples Expansion Plan

更新时间：2026-03-30

> 说明：本文件为样例扩展历史规划；当前样例清单与运行方式以 `README.md` 和 `examples/*` 实际代码为准。

## 参考基线

- PocketFlow Design Pattern: `https://the-pocket.github.io/PocketFlow/design_pattern/`
- 对齐范围：Sequential / Routing / Parallel / Map Reduce / Tool Call / Structure / Multi-Agent

## Pattern 对齐矩阵

| Example | 主要 Pattern | 对齐目标 |
| --- | --- | --- |
| `01-chat-minimal` | Sequential | 单轮最小链路，建立基础调用认知 |
| `02-tool-loop-basic` | Tool Call + Sequential | 展示工具调用闭环与反馈迭代 |
| `03-mcp-mixed-call` | Tool Call + Routing | 展示 local/MCP 混合路由与分发策略 |
| `04-streaming-interrupt` | Structure（中断/恢复语义） | 展示流式链路中的取消与收敛 |
| `05-parallel-tools-fanout` | Parallel | 展示 goroutine fanout 与背压行为 |
| `06-async-job-progress` | Map Reduce（近似） + Parallel | 展示异步任务拆分、聚合与进度回传 |
| `07-multi-agent-async-channel` | Multi-Agent + Structure + HITL Clarification | 展示单进程多代理协作、clarification await/resume 与异步通信 |
| `08-multi-agent-network-bridge` | Multi-Agent + Structure（Network） | 展示基于 HTTP + JSON-RPC 2.0 的网络通信桥接 |

## Phase R2 Batch (Foundational + Core Patterns)

状态：已完成

- `examples/01-chat-minimal`
- `examples/02-tool-loop-basic`
- `examples/03-mcp-mixed-call`
- `examples/04-streaming-interrupt`

R2 重点覆盖 Pattern：
- Sequential
- Tool Call
- Routing（通过 MCP/local 混合调用体现）
- Structure（最小中断/恢复语义）

每个示例目录包含：
- `main.go`: 可运行最小实现
- `README.md`: 运行方式、预期行为与边界说明

示例增强 backlog 统一收敛在：
- `docs/development-roadmap.md` 的 `Examples Backlog（从 examples TODO 收敛）` 小节

## Phase R3 Batch (Advanced + Concurrency Patterns)

状态：已完成

- `examples/05-parallel-tools-fanout`
- `examples/06-async-job-progress`
- `examples/07-multi-agent-async-channel`
- `examples/08-multi-agent-network-bridge`

R3 重点覆盖 Pattern：
- Parallel
- Map Reduce（以异步任务拆分/聚合形式体现）
- Multi-Agent
- Structure（跨组件协作编排）

说明：
- `07` 与 `08` 分别覆盖进程内 channel 通信与网络通信，避免单示例承载两类复杂度。
- `08` 通信协议固定为 HTTP + JSON-RPC 2.0（参考 MCP 协议语义）。
- `07` 输出结构化 `clarification_request` 事件与 timeline reason（`hitl.await_user`/`hitl.resumed`），用于前端直接消费。

## Backlog（Pattern 补齐建议）

为更完整对齐 PocketFlow pattern，建议后续新增：
- `examples/09-routing-strategy-switch`: 显式多路由策略选择（按输入/置信度/成本）
- `examples/10-hierarchical-structure`: 分层编排（planner/worker/validator）
- `examples/11-map-reduce-large-batch`: 大批量任务切分与聚合优化

## Style Guide (PocketFlow-inspired)

- 按难度渐进，确保每个示例都可独立运行。
- 文档强调行为预期（并发、队列、取消、重试）。
- 高阶示例必须包含诊断输出，便于性能与稳定性排障。
- 每个示例需在 README 或注释中标注“对应 Pattern”和“本示例不覆盖的边界”。
