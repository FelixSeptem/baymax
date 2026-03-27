## Why

A49 已建立 cross-domain arbitration explainability（primary/secondary/hint/rule_version）基础，但还缺少“规则版本治理”契约。随着规则持续演进，如果没有统一的版本选择、兼容窗口与漂移阻断，replay 对账和升级迁移会快速失稳。

## What Changes

- 新增 arbitration rule version governance 契约，定义版本选择、兼容窗口与不兼容处理策略。
- 新增 `runtime.arbitration.version.*` 配置域（default version、compatibility window、unsupported/mismatch policy），并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 固化 rule version 解析来源与落盘字段（requested/effective/source），保证 Run/Stream/readiness/admission 语义一致。
- 在 readiness preflight 与 admission guard 中补齐版本治理 finding/reason 映射（如 unsupported version、compatibility mismatch）。
- 扩展 diagnostics replay tooling，新增 cross-version fixture 与 drift 分类（`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`）。
- 将 cross-version replay suites 接入 `check-quality-gate.*` 阻断流程并保持 shell/PowerShell parity。
- 同步更新 README、roadmap、runtime diagnostics 文档与主线契约索引。

## Capabilities

### New Capabilities
- `cross-domain-arbitration-version-governance`: 定义 arbitration 规则版本治理、兼容窗口与跨版本回放契约。

### Modified Capabilities
- `cross-domain-primary-reason-arbitration`: 从固定单版本裁决扩展到受控版本选择与跨版本兼容治理。
- `runtime-config-and-diagnostics-api`: 增加 arbitration version 配置与 additive 诊断字段，保持 `additive + nullable + default` 兼容窗口。
- `runtime-readiness-preflight-contract`: 增加 arbitration version 兼容性预检与 strict/non-strict 分类语义。
- `runtime-readiness-admission-guard-contract`: 增加版本不兼容场景的 admission fail-fast/allow-and-record 契约。
- `diagnostics-replay-tooling`: 增加 cross-version fixture schema 与版本漂移分类断言。
- `go-quality-gate`: 增加 A50 cross-version suites 的阻断映射。

## Impact

- 代码：
  - `runtime/config`、`runtime/config/readiness*`（版本治理配置与分类输出）
  - `runtime/diagnostics/*`、`observability/event/*`（version governance additive 字段）
  - `tool/diagnosticsreplay/*`（cross-version fixture loader/normalizer/assert）
  - `integration/*`（Run/Stream parity + replay parity + drift guard）
  - `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 不改变 A49 的业务终态机，仅增加规则版本治理与回放阻断契约。
  - 默认策略保持 deterministic（推荐 `on_unsupported=fail_fast`、`on_mismatch=fail_fast`）。
