## Why

当前运行时在功能上已具备可用 v1 基线，但并发执行与异步通信能力仍缺乏统一策略、可观测闭环与安全基线。为支撑高并发场景和后续教程体系扩容，需要先完成 Go 原生 goroutine 并发与异步机制的系统化升级。

## What Changes

- 建立运行时统一并发控制能力：并发度、队列、背压、取消传播和超时策略。
- 引入异步通信管线：工具/MCP 调用通道化，支持异步事件回传与关联追踪。
- 扩展可观测性：新增队列深度、排队时延、fanout、drop/retry、取消原因等指标字段。
- 建立并发安全基线门禁：`go test -race ./...`、goroutine 泄漏检查与取消风暴场景测试。
- 建立性能回归门禁，采用“相对提升百分比”作为优化与回归评估标准。
- 参考 PocketFlow tutorials 结构分批扩容 examples，并在示例目录保留 TODO 便于后续持续优化。

## Capabilities

### New Capabilities
- `runtime-concurrency-control`: 运行时并发调度、背压策略和取消收敛控制。
- `async-communication-pipeline`: 工具与 MCP 调用的异步通信机制和关联事件语义。
- `tutorial-examples-expansion`: 分阶段扩容教程示例并保留 TODO 演进位。

### Modified Capabilities
- `go-quality-gate`: 增加并发安全强制门禁与性能回归百分比评估要求。

## Impact

- Affected code:
  - `core/runner/*`, `tool/local/*`, `mcp/http/*`, `mcp/stdio/*`
  - `integration/*`（并发/异步压测与回归）
  - `observability/event/*`, `observability/trace/*`
  - `examples/*`（分阶段新增）
  - CI 与质量配置（`go test -race`, benchmark, lint）
- APIs/behavior:
  - 新增并发与异步配置项及默认值文档
  - 新增并发可观测字段，完善故障与取消语义
- Dependencies/systems:
  - 不新增核心 LLM/MCP SDK 类型依赖
  - 重点依赖 Go 原生 goroutine/channel/context 与现有观测体系
