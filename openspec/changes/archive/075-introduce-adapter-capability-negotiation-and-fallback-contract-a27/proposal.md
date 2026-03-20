## Why

A26 正在补齐 adapter manifest 与 runtime 兼容校验，但仍缺少“接入后能力协商”的统一契约：同一 adapter 在不同运行时上下文下如何处理 `required/optional` 能力、如何降级、以及 Run/Stream 是否保持一致，当前没有统一标准。A27 目标是补齐 capability negotiation 与 fallback contract，避免接入后语义漂移。

## What Changes

- 新增 adapter capability negotiation 契约能力，定义请求能力与 adapter 声明能力之间的匹配规则。
- 固化协商策略：支持 `fail_fast` 与 `best_effort`，默认 `fail_fast`。
- 固化 `required` 缺失直接拒绝、`optional` 缺失可降级并输出标准 reason taxonomy。
- 增加 Run/Stream 协商语义等价要求，避免模式切换时降级行为不一致。
- 新增协商结果诊断字段（additive + nullable + default）用于追踪 capability 命中与降级计数。
- A22 conformance harness 增加 capability negotiation 矩阵与回归用例。
- A23 脚手架增加 negotiation/fallback 测试骨架与示例配置。
- `check-quality-gate.*` 增加 capability negotiation contract gate，默认阻断。

## Capabilities

### New Capabilities
- `adapter-capability-negotiation-and-fallback`: 定义 adapter 能力协商、降级语义、reason taxonomy 与 Run/Stream 等价要求。

### Modified Capabilities
- `external-adapter-conformance-harness`: 增加 capability negotiation 矩阵校验与失败分类要求。
- `adapter-scaffold-generator-and-conformance-bootstrap`: 增加 negotiation/fallback 测试骨架和默认策略模板生成要求。
- `go-quality-gate`: 增加 capability negotiation contract 检查为阻断项。

## Impact

- 代码与测试：
  - `adapter/capability/*`（协商逻辑与类型定义）
  - runtime adapter 接入路径（协商入口）
  - `integration/adapterconformance/*`（negotiation contract suites）
  - `adapter/scaffold/*`（negotiation 测试骨架模板）
  - `scripts/check-adapter-capability-contract.sh`
  - `scripts/check-adapter-capability-contract.ps1`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 文档：
  - `README.md`
  - `docs/external-adapter-template-index.md`
  - `docs/adapter-migration-mapping.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
