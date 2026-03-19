## Why

A21 正在补齐外部适配样板与迁移映射，但仓库仍缺少统一的“外部适配一致性验收”执行入口，无法持续验证第三方 adapter 是否遵循 Baymax 契约。A22 目标是建立 conformance harness 并纳入质量门禁，避免样板与实现长期漂移。

## What Changes

- 新增外部适配 conformance harness 能力，覆盖 `MCP > Model > Tool` 三类最小一致性矩阵。
- 固化一致性校验维度：Run/Stream 语义等价（适用项）、错误层级/错误码归一、降级语义、fail-fast 边界。
- harness 默认离线执行（stub/fake 驱动），不依赖外网与外部服务。
- conformance gate 默认阻断并 fail-fast，作为质量门禁必过项。
- 新增 `scripts/check-adapter-conformance.sh/.ps1` 并接入 `check-quality-gate.*`。
- 更新主干索引与文档导航，建立“样板 -> conformance -> gate”可追溯链路。

## Capabilities

### New Capabilities
- `external-adapter-conformance-harness`: 定义外部适配一致性测试框架、最小矩阵、默认离线模式与失败语义。

### Modified Capabilities
- `go-quality-gate`: 增加 adapter conformance gate 的标准阻断要求与跨平台一致执行约束。

## Impact

- 代码与测试：
  - `integration/*` 或 `tool/*` 下新增 conformance harness 与测试夹具
  - `scripts/check-adapter-conformance.sh`
  - `scripts/check-adapter-conformance.ps1`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 文档：
  - `docs/mainline-contract-test-index.md`
  - `README.md`
  - `docs/development-roadmap.md`
  - A21 相关样板文档（增加 conformance 验收指引链接）
