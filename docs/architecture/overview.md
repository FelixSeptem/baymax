# 架构总览

Baymax 是 library-first、contract-first 的 Go Agent runtime。宿主组合 `core/runner`、Context、Provider、tool/MCP、orchestration 和 observability；运行事实由 runtime 与 `RuntimeRecorder` 形成可查询、可回放的记录。

```text
Host / Integration
        |
        v
core/runner ---- context ---- model/<provider>
      |             |               |
      +--------- orchestration -----+
      |             |
      v             v
 tool/local ---- mcp/http|stdio
      |
      v
 observability/event.RuntimeRecorder -> diagnostics / replay / gates
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

宿主提交 Run 或 Stream 请求后，runner 读取 context snapshot，向 Provider 发起模型请求；模型响应可能触发本地工具或 MCP 调用，再回到 runner 继续循环。每个阶段通过 `RuntimeRecorder` 写入 timeline/diagnostics；终态由 Run 与 Stream 共享同一决策语义。

## 非目标

Baymax 不提供远程控制面、全局路由器、自动 Provider 探测或隐藏的跨租户状态。外部服务和宿主策略由集成方拥有。

验证：`go test ./core/runner ./context ./model/... ./tool/... ./orchestration/...`；边界检查：`scripts/check-runtime-boundaries.sh`。
