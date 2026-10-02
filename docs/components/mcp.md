# MCP transport

## Purpose

`mcp/http` 和 `mcp/stdio` 提供 MCP transport、session 和协议适配，供 tool 层显式调用。

## Owned facts/state

连接状态、transport 错误、MCP request/response 和 capability facts 属于 MCP owner。

## Inputs/outputs

输入为 MCP client/server 配置和 tool request；输出为 normalized tool result 与 transport diagnostics。

## Dependencies and forbidden dependencies

transport 可以依赖 `mcp/internal`；runtime、context 和非 MCP 包不得依赖 `mcp/internal`。

## Lifecycle / failure / rollback

连接 → capability handshake → call → close。连接失败应释放资源并返回可分类错误，不改变其它工具状态。

## Observability

记录 endpoint profile、transport、耗时和错误类别；敏感 header 必须脱敏。

## Verification / source links

`go test ./mcp/... -count=1`；[MCP README](../../mcp/README.md)、[runtime profiles](../mcp-runtime-profiles.md)。
