# Extensions

## Purpose

Extensions 通过公开生命周期接口扩展 tool、model、memory、MCP 等能力，不改变 core/runtime ownership。

## Owned facts/state

extension manifest、版本、lifecycle hook 和 conformance evidence 由 extension owner 管理。

## Inputs/outputs

输入是宿主显式加载的 extension；输出是 capability、hook result 和 diagnostics。

## Dependencies and forbidden dependencies

只能依赖公开 extension API 和稳定类型；禁止 monkey patch runner 或旁路 recorder。

## Lifecycle / failure / rollback

discover → validate → initialize → run → close。初始化失败不污染宿主已生效配置；回滚通过卸载或上一个 manifest 完成。

## Observability

每个 hook 记录 extension id、阶段、错误和耗时，遵守隐私边界。

## Verification / source links

`bash scripts/check-extension-lifecycle-contract-replay.sh`；[external extension guide](../external-adapter-template-index.md)。
