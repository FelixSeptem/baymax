# Runner

## Purpose

`core/runner` 承担 Run/Stream 主循环、阶段协调和统一终态解析。

## Owned facts/state

请求生命周期、tool-loop 迭代、终态 family/state 和 Run/Stream parity 由 runner 协调；Provider wire facts 不属于 runner。

## Inputs/outputs

输入是宿主 request、context、model/tool capabilities 和 runtime policy；输出是响应、事件、诊断和 terminal outcome。

## Dependencies and forbidden dependencies

可依赖 context、model 抽象、tool 接口和 observability；不得依赖 `mcp/http`、`mcp/stdio` 的 transport 实现。

## Lifecycle / failure / rollback

Admission → context → model → tool loop → terminal。非法 admission 在执行前 fail-fast；终态解析失败回滚到可诊断的失败状态。

## Observability

通过 `observability/event.RuntimeRecorder` 写入阶段、工具 attempt/completion 和终态。

## Verification / source links

`go test ./core/runner -count=1`；源代码：[core/runner](../../core/runner/)、[runtime lifecycle](../architecture/runtime-lifecycle.md)。
