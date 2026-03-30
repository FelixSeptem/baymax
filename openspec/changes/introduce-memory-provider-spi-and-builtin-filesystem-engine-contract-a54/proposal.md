## Why

当前项目的 memory 接入主要依赖 CA2 file/external retriever 形态，缺少“统一 memory SPI + 主流框架 profile + 内置文件系统引擎”的完整契约。要实现 mem0、Zep、OpenViking 等框架顺滑接入并避免业务侧重复胶水代码，需要一次性冻结 memory 运行时 contract，并提供可回滚的模式开关（external SPI 与 builtin filesystem）。

## What Changes

- 新增统一 memory engine 契约，定义 canonical `Query/Upsert/Delete` SPI 与归一化错误/元数据语义。
- 新增运行模式开关：`external_spi|builtin_filesystem`，并定义热更新原子切换与回退规则。
- 新增内置文件系统 memory 引擎 contract（append-only WAL + 原子 compaction/index），复用现有文件系统 memory 思路并标准化读写语义。
- 新增主流 memory framework profile pack：`mem0`、`zep`、`openviking`（并支持 generic/custom profile 扩展）。
- 扩展 Context Assembler CA2 路径，统一通过 memory facade 访问 memory，避免 provider-specific 分支渗透到主流程。
- 扩展 readiness preflight + diagnostics + replay + quality gate，补齐 memory mode/switch/fallback 的可观测与阻断语义。
- 扩展 adapter template/conformance 契约，新增 memory adapter onboarding 模板、迁移映射与 conformance matrix。
- 本提案不改变现有 Run/Stream 外部契约，不引入平台化控制面能力。

## Capabilities

### New Capabilities
- `runtime-memory-engine-spi-and-filesystem-builtin`: 统一 memory SPI、内置文件系统 memory 引擎、external SPI 模式切换与回退治理契约。

### Modified Capabilities
- `context-assembler-stage-routing`: CA2 Stage2 读取路径接入 memory facade 与模式切换语义。
- `runtime-config-and-diagnostics-api`: 增加 memory mode/provider/profile/fallback 配置域与 additive 诊断字段。
- `runtime-readiness-preflight-contract`: 增加 memory backend/profile 可用性预检与 strict/non-strict 映射。
- `external-adapter-conformance-harness`: 增加 memory adapter conformance matrix（mem0/zep/openviking/generic）。
- `external-adapter-template-and-migration-mapping`: 增加 memory adapter 模板与 file->SPI 迁移映射。
- `adapter-manifest-and-runtime-compatibility`: 增加 memory adapter manifest 声明字段与兼容性校验语义。
- `diagnostics-replay-tooling`: 增加 A54 memory fixture 与 drift 分类断言。
- `go-quality-gate`: 增加 memory adapter contract gate 与独立 required-check 候选暴露。

## Impact

- 代码：
  - `memory/*`（新增 memory SPI、builtin filesystem engine、external adapter facade）
  - `context/assembler`（CA2 memory facade 接线）
  - `runtime/config`（memory 配置、校验、热更新回滚）
  - `runtime/config/readiness*`（memory readiness findings）
  - `runtime/diagnostics`、`observability/event`（memory additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（A54 fixtures 与 conformance suites）
  - `scripts/check-quality-gate.*` 与新增 memory gate 脚本
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/external-adapter-template-index.md`
  - `docs/adapter-migration-mapping.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 默认可保持现有路径不变（可配置默认 `builtin_filesystem` 或关闭）。
  - 通过 `additive + nullable + default + fail-fast` 保持兼容窗口。
