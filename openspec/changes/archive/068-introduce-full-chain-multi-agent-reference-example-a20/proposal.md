## Why

当前示例集已覆盖分段能力，但仍缺少“单个可运行样例”串联 `team + workflow + a2a + scheduler + recovery` 的全链路参考路径。A20 目标是在不引入平台化能力的前提下提供 lib-first 参考示例，降低开源使用者的接入与排障门槛。

## What Changes

- 新增全链路参考示例能力，提供一个可直接运行的 `examples/09-*` 入口。
- 示例默认使用 in-memory A2A 路径，不依赖外部网络或服务。
- 示例同时覆盖 `Run` 与 `Stream` 两条调用语义路径。
- 示例内提供 async + delayed + recovery 的最小组合路径与可观测输出。
- 更新教程文档导航与运行说明，明确场景边界、配置入口与观测点。
- 新增示例 smoke 校验，并纳入 quality gate 阻断路径（Shell/PowerShell 一致）。

## Capabilities

### New Capabilities
- `multi-agent-full-chain-reference-example`: 定义 team/workflow/a2a/scheduler/recovery 全链路参考示例的功能边界、运行语义与观测输出契约。

### Modified Capabilities
- `tutorial-examples-expansion`: 扩展教程能力要求，新增全链路参考示例与 Run/Stream 双路径覆盖要求。
- `go-quality-gate`: 增加全链路参考示例 smoke 校验并纳入标准门禁阻断。

## Impact

- 代码：
  - `examples/09-multi-agent-full-chain-reference/*`
  - `examples/README` 或各示例 README 导航条目
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
  - （可选）新增 `scripts/check-examples-smoke.*` 或复用现有 gate 路径
- 测试与验证：
  - 新增示例 smoke 测试或脚本断言（Run/Stream + async/delayed/recovery 组合路径）
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `examples/09-multi-agent-full-chain-reference/README.md`
