# 模块 Ownership 与依赖边界

| 模块 | 负责 | 不负责 |
| --- | --- | --- |
| `core/runner` | Run/Stream 主循环、终态协调 | Provider wire format、MCP transport |
| `context` | canonical context、预算投影、压缩 | Provider SDK、网络传输 |
| `model/<provider>` | Provider request/response projection | 全局路由、runtime 状态 |
| `tool/local` | 本地工具生命周期和错误隔离 | MCP transport |
| `mcp/http`, `mcp/stdio` | MCP 传输与 adapter 边界 | runtime 决策 |
| `orchestration`, `a2a` | workflow/teams/scheduler 组合 | 单个 Provider 协议 |
| `runtime/config` | 配置解析、校验、热更新回滚 | 业务执行 |
| `observability` | event recorder、诊断和导出 | 重新执行 runtime |
| `adapter`, `integration` | 外部接入模板、conformance/replay | 隐式全局注册 |
| `skill` | skill 发现和加载 | runtime 终态 |

依赖方向和禁止边界以 [runtime-module-boundaries.md](../runtime-module-boundaries.md) 为准；本页只做责任导航。

验证：`scripts/check-runtime-boundaries.sh` 与 `scripts/check-docs-consistency.sh`。
