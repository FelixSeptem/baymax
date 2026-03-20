## Why

A26/A27 分别补齐了 adapter manifest 兼容校验与能力协商契约，但外部 adapter 合同仍缺少“版本化 profile + 可回放基线”治理，升级时容易出现语义回退且难以快速定位。A28 目标是引入 contract profile version 与 replay gate，把 adapter 合同演进纳入可回归阻断路径。

## What Changes

- 新增 adapter contract profile version 能力，定义 profile 版本命名、兼容窗口与升级规则。
- 在 manifest/conformance/negotiation 链路统一引入 `contract_profile_version` 字段。
- 运行时增加 profile 支持窗口校验，默认支持 `current + previous`，不满足时 fail-fast。
- 建立 adapter contract replay 基线（golden fixtures），覆盖 manifest 解析、compat 校验、negotiation/fallback 结果与 reason taxonomy。
- 新增 `check-adapter-contract-replay.sh/.ps1`，并接入 `check-quality-gate.*` 作为阻断项。
- 默认关闭 warn-only；profile 不兼容与回放漂移均返回 non-zero。
- 更新 README/roadmap/mainline index/adapter docs，补齐 profile 升级与回放维护指引。

## Capabilities

### New Capabilities
- `adapter-contract-profile-versioning-and-replay`: 定义 adapter 合同 profile 版本、兼容窗口、回放基线与阻断校验语义。

### Modified Capabilities
- `adapter-manifest-and-runtime-compatibility`: 增加 `contract_profile_version` 字段与 profile 兼容窗口校验要求。
- `adapter-capability-negotiation-and-fallback`: 增加 negotiation 结果与 reason taxonomy 的 profile 化回放一致性要求。
- `go-quality-gate`: 增加 adapter contract replay gate 阻断要求。

## Impact

- 代码与测试：
  - `adapter/profile/*`（profile 版本与兼容窗口逻辑）
  - manifest/negotiation 相关模块（profile 字段接入）
  - `integration/adapterconformance/*`（profile replay contract suites）
  - `scripts/check-adapter-contract-replay.sh`
  - `scripts/check-adapter-contract-replay.ps1`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/external-adapter-template-index.md`
  - `docs/adapter-migration-mapping.md`
