## Why

当前运行时参数主要以代码内常量和初始化参数为主，缺少统一的配置加载、可观测诊断与在线调整能力，导致在不同部署环境下调优成本高、故障定位慢。现在引入统一配置与诊断 API，可以在不重启或最小化干预的前提下完成配置收敛与问题排查。

## What Changes

- 新增运行时配置能力：支持 YAML 文件加载、环境变量覆盖、默认值回填与配置校验。
- 明确配置优先级：`env > file > default`，并提供脱敏后的生效配置导出接口。
- 新增诊断能力（仅库 API）：提供最近运行状态和最近 MCP 调用摘要查询接口，不提供 CLI 命令。
- 新增热更新能力：监听配置文件变更并原子切换配置；若新配置校验失败则保留旧配置并返回错误。
- 将 MCP 运行时（HTTP/STDIO）对齐到统一配置源与诊断数据源，保持语义一致并采用 fail-fast 错误策略。
- 更新 README 与 docs 下相关文档及变更产物说明，补充配置字段、环境变量映射、热更新边界与使用示例。

## Capabilities

### New Capabilities
- `runtime-config-and-diagnostics-api`: 统一配置加载、热更新与诊断查询的运行时 API 能力。

### Modified Capabilities
- `mcp-runtime-reliability-profiles`: 运行时 profile 的参数来源改为可配置化，并要求配置错误时立即终止。

## Impact

- 影响目录：`mcp/runtime`、`mcp/http`、`mcp/stdio`、`internal/config`（或等价新目录）、`README.md`、`docs/*`。
- 新增依赖：`github.com/spf13/viper`（配置加载与变更监听）。
- API 影响：新增公开库接口（配置加载/热更新/诊断查询），无 CLI 兼容负担。
- 质量与验证：补充并发安全、热更新一致性和配置优先级测试；接入/更新 `golangci-lint` 建议配置。
