## Why

A47 正在收敛 `readiness + timeout + adapter health` 的交叉 replay fixture 门禁，但主线仍缺少一个明确的“跨域 primary reason 裁决契约”，导致同一事件组合在不同路径下可能产生不一致的 primary code/domain。若不先冻结裁决规则，后续 replay fixture 只能发现漂移，无法提供稳定且可解释的权威语义。

## What Changes

- 新增 cross-domain primary reason arbitration 契约，固化 `timeout/readiness/adapter-health` 冲突场景下的 primary reason 选取优先级。
- 固化 tie-break 规则（同优先级冲突时的确定性排序）并确保 Run/Stream 与 replay 路径语义等价。
- 新增 runtime 诊断 additive 字段（`runtime_primary_domain`、`runtime_primary_code`、`runtime_primary_source`、冲突计数）。
- 将 primary reason arbitration 结果接入 readiness preflight 与 admission 决策解释层，避免跨模块 reclassification drift。
- 扩展 diagnostics replay tooling 与 quality gate 套件，覆盖 arbitration drift、taxonomy drift、idempotency drift。
- 同步更新主线契约索引、runtime 配置/诊断文档与 roadmap 状态口径。

## Capabilities

### New Capabilities
- `cross-domain-primary-reason-arbitration`: 定义跨域 primary reason 的优先级、冲突裁决、可观测性输出与回放一致性契约。

### Modified Capabilities
- `runtime-readiness-preflight-contract`: 增加 readiness 输出与 cross-domain primary reason 对账一致性要求。
- `runtime-readiness-admission-guard-contract`: 增加 admission deny/allow 解释链路对 primary reason 的消费一致性要求。
- `runtime-operation-profiles-and-timeout-resolution-contract`: 增加 timeout 结果在跨域裁决中的优先级与稳定来源要求。
- `adapter-runtime-health-probe-contract`: 增加 adapter-health 输出参与 primary reason 裁决时的 canonical taxonomy 约束。
- `runtime-config-and-diagnostics-api`: 增加 arbitration additive 字段与兼容窗口语义。
- `diagnostics-replay-tooling`: 增加 arbitration fixture 的规范化比对与漂移分类。
- `go-quality-gate`: 增加 arbitration contract suites 阻断映射与 shell/PowerShell parity 要求。

## Impact

- 代码：
  - `runtime/config/readiness*`（跨域 primary reason 裁决与解释）
  - `runtime/diagnostics/*` 与 `observability/event/*`（additive 字段与聚合）
  - `tool/diagnosticsreplay/*`（arbitration fixture 比对与 drift 分类）
  - `integration/*`（Run/Stream/replay parity 套件）
  - `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 采用固定优先级（非可配置）收敛语义；
  - 新增字段遵循 `additive + nullable + default`；
  - 不改变 Run/Stream 业务终态，仅收敛解释与诊断层。
