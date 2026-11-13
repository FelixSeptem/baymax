## Why

既有 `SandboxExecutor` 接口（`core/types/types.go`）、`adapter/manifest/sandbox_profile_pack.go`（nsjail/bwrap/oci_runtime/windows_job 四种后端画像）与 `integration/sandboxconformance` harness 已冻结沙箱决策契约与一致性矩阵；但本仓只有测试 fake，没有生产级执行器实现。默认 `Security.Sandbox.Enabled=false`、`DefaultAction=host`，工具直接在宿主进程内执行，sandbox 只是一条决策契约而非执行保障。

宿主需要把 anthropics/sandbox-runtime、open-sandbox.ai、e2b 等主流沙箱运行时作为真实 `SandboxExecutor` 后端接入，才能把既有 `host|sandbox|deny` 决策、capability 协商、egress 治理与可回放诊断落到真实隔离执行上。

## What Changes

- 新增 `sandbox-external-executor-backends` 能力，定义主流沙箱运行时的 `SandboxExecutor` 适配接入契约：`BuildSandboxExecSpec` → 远端/容器运行时调用、capability probe 协商、egress 治理对齐、session 生命周期与 fallback。
- 将主流运行时归类为既有四种 canonical backend 的实现变体（e2b / open-sandbox / anthropics-sandbox-runtime 均映射到 `oci_runtime`；`local` 为 onboarding-only），在 `runtime/config` 注册映射、由 `adapter/sandboxexternal` 实现，并在 `sandboxconformance` harness 验收。
- 冻结外部 adapter 接入契约与失败分类（launch_failed / capability_mismatch / timeout / oom / egress_denied / backend_unavailable），复用既有 `sandbox-*-matrix`，不新建平行契约。
- 提供 adapter onboarding 模板与 offline replay fixture，纳入 gate 阻断（shell/PowerShell parity）。
- 不在本仓从零自建隔离层；外部运行时默认不开启，沿用 `observe → enforce` 渐进启用与 `DefaultAction=host` 回退。

## Example Impact Assessment

新增示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md`, `docs/external-adapter-template-index.md`, `docs/adapter-migration-mapping.md`, `adapter/manifest/README.md`
- 维护 owner：adapter/runtime maintainers
- 验证证据：`pwsh -File scripts/check-sandbox-external-executor-conformance.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `go test ./adapter/... ./tool/local/... ./integration/sandboxconformance/... -count=1`

## Documentation Impact Assessment

| 文档域 | 变更 | 路径 | Owner | 证据 |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | adapter/runtime maintainers | adapter/sandboxexternal 仅依赖 core/types 与 runtime/config，未新增或反转既有模块边界约束 |
| components | 修改文档 | `adapter/manifest/README.md` | adapter maintainers | adapter package contract |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | backend registration keys/defaults |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md`, `docs/external-adapter-template-index.md` | adapter/runtime maintainers | public matrix and test index |
| examples | 修改文档 | `docs/adapter-migration-mapping.md` | adapter maintainers | migration pack |
| best practices | 修改文档 | `README.md` | runtime maintainers | capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active status and scope |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | external backend 配置键、默认值与 backend_unavailable 诊断边界 |
| CLI/integration | 修改文档 | `docs/mainline-contract-test-index.md`, `docs/external-adapter-template-index.md` | adapter/runtime maintainers | 外部后端矩阵、onboarding 模板与 gate 入口 |

## Capabilities

### New Capabilities
- `sandbox-external-executor-backends`: 定义主流沙箱运行时的 `SandboxExecutor` 适配接入契约与 conformance 验收。

### Modified Capabilities
- `sandbox-execution-isolation`: 扩展可执行后端集合，从"仅契约 + fake"到"真实外部后端变体"。
- `sandbox-adapter-conformance-and-migration-pack`: 新增主流沙箱后端矩阵与 onboarding 模板。
- `tool-security-governance-s2`: sandbox 分发路径不变，仅后端实现扩展，fallback 语义保持一致。

## Impact

- 代码：
  - `adapter/`（主流沙箱运行时 adapter 实现、manifest 注册）
  - `core/types`（`SandboxExecutor` SPI 不变，仅新增实现）
  - `runtime/config`（backend 注册、校验）
  - `tool/local`（sandbox 分发路径不变）
  - `integration/sandboxconformance`（新增 backend 矩阵与 fixture）
  - `scripts/check-sandbox-external-executor-conformance.*`（新增 gate，shell/PowerShell parity）
- 文档：`README.md`, `docs/runtime-config-diagnostics.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/external-adapter-template-index.md`, `docs/adapter-migration-mapping.md`, `adapter/manifest/README.md`
- 兼容性：默认 `Security.Sandbox.Enabled=false`、`DefaultAction=host` 不变；外部后端默认不开启，异常回退到 host 模式。
