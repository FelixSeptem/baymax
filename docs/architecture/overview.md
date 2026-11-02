# 架构总览

Baymax 是一个 library-first、contract-first 的 Go Agent runtime。它把可嵌入的执行 primitive 交给宿主组合，而不是替宿主拥有凭据、网络、业务数据或全局控制面。理解系统时，先看[Execution Model](execution-model.md)建立跨组件概念模型，再看[设计原则](design-principles.md)、[运行生命周期](runtime-lifecycle.md)和[数据与事件流](data-event-flow.md)。

```text
Host / Integration (authority, effects, persistence)
        |
        v
core/runner (Run/Stream state machine)
   |              |                  |
   v              v                  v
context      model/<provider>    orchestration/*
   |              |                  |
   +---------- tool/local ---- mcp/http|stdio
                  |
                  v
       observability/event.RuntimeRecorder
                  |
             diagnostics / replay / gates
```

## 边界原则

- `runtime/*` 不依赖 `mcp/http` 或 `mcp/stdio`。
- 非 `mcp/*` 包不依赖 `mcp/internal/*`。
- `context/*` 不直接引入 Provider 官方 SDK；协议细节位于 `model/<provider>`。
- 诊断写入只有 `observability/event.RuntimeRecorder` 一个入口。
- 配置优先级固定为 `env > file > default`；非法配置和非法热更新必须 fail-fast + 原子回滚。
- QueryRuns/诊断字段采用 additive + nullable + default。

权威依赖矩阵见 [运行时模块边界](../runtime-module-boundaries.md)。

## 关键事实流

宿主提交 Run 或 Stream 请求后，runner 先完成 admission 和配置/预算快照，再由 context 形成请求投影，`model/<provider>` 完成协议适配。模型响应可能触发本地工具或 MCP 调用，再回到 runner 继续循环。每个 source owner 产生自己的 lifecycle facts；`RuntimeRecorder` 只负责结构化诊断单写，不成为新的执行 owner。终态由 Run 与 Stream 共享同一决策语义。

## 读者应先确认的边界

| 问题 | 事实源 | 不能从哪里推断 |
| --- | --- | --- |
| 谁能执行或恢复？ | capability manifest、source owner、integration guide | package 是否存在、DTO 是否能编译 |
| 哪些事实可回放？ | source-native history contract、retention/cursor | callback EventID、host observation sequence |
| 哪些事实是终态？ | source terminal evidence | observer status、RunResult 拼接或业务 ack |
| 谁提交业务 effect？ | 宿主 authorization/effect/commit owner | Baymax 的 ack、digest、diagnostic 或 replay |
| 失败是否会自动降级？ | typed error 和 admission contract | retry、重新 Run parent 或 latest replay |

## 非目标

Baymax 不提供远程控制面、全局路由器、自动 Provider 探测或隐藏的跨租户状态。外部服务和宿主策略由集成方拥有。

验证：`go test ./core/runner ./context ./model/... ./tool/... ./orchestration/...`；边界检查：`scripts/check-runtime-boundaries.sh`。契约到测试的完整映射见 [主线契约测试索引](../mainline-contract-test-index.md)。
