# Adapters 与 integration

## Purpose

adapter/integration 提供外部模型、工具、MCP 和测试 fixture 的 onboarding、manifest、conformance 与 replay。

## Owned facts/state

adapter capability、manifest、版本和 conformance evidence 属于 adapter owner；runtime 不自动发现外部 adapter。

## Inputs/outputs

输入是显式注册的 adapter 和 fixture；输出是 capability verdict、replay evidence 和迁移提示。

## Dependencies and forbidden dependencies

适配器通过公开接口接入，不得依赖 `mcp/internal` 或隐式全局 registry。

## Lifecycle / failure / rollback

manifest 校验 → capability negotiation → conformance/replay。失败时拒绝准入，回滚到上一个显式版本。

## Observability

记录 adapter id、版本、能力和失败 reason code，不记录 credential。

## Verification / source links

`bash scripts/check-adapter-conformance.sh`；[template index](../external-adapter-template-index.md)、[migration mapping](../adapter-migration-mapping.md)。
