# Tools

## Purpose

工具层提供本地 tool 生命周期、schema、调用隔离和统一结果；MCP transport 由独立组件负责。

## Owned facts/state

tool schema、调用 attempt/completion、超时、取消和错误隔离属于 tool owner。

## Inputs/outputs

输入是已准入的 tool call；输出是结构化 result、错误和 diagnostics。

## Dependencies and forbidden dependencies

可依赖 context、runtime policy 和 observability；不得绕过 runner 直接改变 terminal outcome。

## Lifecycle / failure / rollback

schema admission → execute → normalize → record。单工具失败默认隔离并按 policy 决定是否终止；无隐式重试。

## Observability

所有调用通过 recorder 记录 bounded 输入摘要、耗时、结果类别和 rollback note。

## Verification / source links

`go test ./tool/... -count=1`；[tool schema audit](../../tool/schemaaudit/README.md)、[local tool](../../tool/local/README.md)。
