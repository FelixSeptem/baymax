## Why

A21-A23 已完成外部 adapter 的模板、conformance 与脚手架链路，但仓库仍缺少统一的 adapter 清单契约，运行时无法在接入前标准化判断“兼容 Baymax 版本与能力边界”。A26 目标是补齐 manifest + runtime compatibility contract，避免接入后才暴露语义漂移。

## What Changes

- 新增 adapter manifest 契约能力，定义外部 adapter 的最小元数据、兼容范围与能力声明格式。
- 运行时在 adapter 装配入口增加 manifest 兼容校验（fail-fast），对缺失/非法/不兼容配置给出标准化错误分类。
- 支持 semver 范围表达（含 `-rc`），默认校验当前 Baymax 版本是否落入 `baymax_compat`。
- 能力声明采用 `required + optional` 双层语义，缺失 required 时 fail-fast，optional 缺失允许降级并记录原因。
- A23 脚手架默认生成 manifest 文件模板，包含三类 adapter 的最小可执行示例。
- A22 conformance harness 增加 manifest profile 对齐检查，验证 manifest 与实际实现语义不漂移。
- `check-quality-gate.*` 增加 manifest 合法性与兼容性检查阻断路径（shell/PowerShell 一致）。

## Capabilities

### New Capabilities
- `adapter-manifest-and-runtime-compatibility`: 定义 adapter manifest schema、runtime 兼容校验、required/optional 能力语义与失败分类。

### Modified Capabilities
- `adapter-scaffold-generator-and-conformance-bootstrap`: 增加 manifest 模板生成与默认字段策略要求。
- `external-adapter-conformance-harness`: 增加 manifest profile 与实现行为一致性校验要求。
- `go-quality-gate`: 增加 adapter manifest 合法性与兼容性检查的阻断要求。

## Impact

- 代码与测试：
  - `adapter/manifest/*`（或等效模块）
  - `runtime/config` / adapter 装配路径（兼容校验接入点）
  - `integration/adapterconformance/*`（manifest 对齐 case）
  - `adapter/scaffold/*`（manifest 模板生成）
  - `scripts/check-adapter-manifest-contract.sh`
  - `scripts/check-adapter-manifest-contract.ps1`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 文档：
  - `README.md`
  - `docs/external-adapter-template-index.md`
  - `docs/adapter-migration-mapping.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
