# Runtime services

## Purpose

`runtime/*` 提供配置、诊断、安全、checkpoint 和运行辅助能力，保持与 MCP transport 的边界。

## Owned facts/state

配置快照、诊断查询、session/run metadata、安全 redaction 和 checkpoint facts 由对应 runtime 子包拥有。

## Inputs/outputs

输入是宿主配置、事件和查询；输出是 validated snapshot、diagnostics 和 replay bundle。

## Dependencies and forbidden dependencies

runtime 不得依赖 `mcp/http` 或 `mcp/stdio`；跨层调用通过抽象接口。

## Lifecycle / failure / rollback

配置采用 `env > file > default`；热更新先校验，非法值 fail-fast + 原子回滚。

## Observability

runtime 产生的事件统一进入 RuntimeRecorder，查询字段遵循 additive + nullable + default。

## Verification / source links

`go test ./runtime/... -count=1`；[配置诊断](../runtime-config-diagnostics.md)、[模块边界](../runtime-module-boundaries.md)。
