# Provider adapters

## Purpose

`model/<provider>` 将 canonical request 投影为 OpenAI、Anthropic、Gemini 等 provider-native 请求，并将响应还原为 runtime 事实。

## Owned facts/state

协议字段映射、能力声明、原生错误分类和请求投影属于 provider adapter；全局路由和 runtime 选择不属于 adapter。

## Inputs/outputs

输入为 SDK-neutral model request；输出为 normalized response、usage、tool calls 和 provider diagnostics。

## Dependencies and forbidden dependencies

只在 `model/<provider>` 引入官方 SDK；不得让 context、runtime 或 orchestration 直接依赖 SDK。

## Lifecycle / failure / rollback

能力 admission → native projection → call → normalize。投影失败不发送请求；重试/回滚由宿主 policy 决定。

## Observability

通过 recorder 记录 provider、model、usage 和可公开错误分类，禁止写入 credential 或完整 prompt。

## Verification / source links

`go test ./model/... -count=1`；[model README](../../model/)、[API reference](../api-reference-d1.md)。
