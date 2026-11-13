## Design

### Owner and scope

`adapter/` 成为主流沙箱运行时 adapter 的唯一 owner：`core/types.SandboxExecutor` SPI 不变，仅新增把外部运行时接入该 SPI 的实现与 manifest 注册。既有 `tool/local` 的 `trySandboxInvoke` / `invokeSandboxAdapter` 调度路径与 `host|sandbox|deny` 决策契约不变；`runtime/config` 增加 backend 注册与校验。外部运行时默认 `Disabled`，沿用 `observe → enforce` 渐进启用与 `DefaultAction=host` 回退。

### Adapter registration and capability negotiation

每个主流运行时映射为一个 canonical backend profile 变体，映射表落在 `runtime/config`（`SecuritySandboxExternalBackendCanonical`），实现在 `adapter/sandboxexternal`；`adapter/manifest` 继续负责 canonical profile pack 与 manifest 侧兼容校验：

- e2b → `oci_runtime`（受控容器，远端 API）
- open-sandbox → `oci_runtime`
- anthropics/sandbox-runtime → `oci_runtime`（Firecracker 微 VM，受控 API）

adapter 在 `readiness`/`admission` 路径执行 capability probe（`SandboxCapabilityProbe`），对 `required_capabilities` 不满足者 deterministic deny；executor 为 nil 时返回 `sandboxReasonLaunchFailed`。

### Execution and failure semantics

adapter 将 `SandboxExecSpec`（command/env/mount/network/resource/session/timeout）翻译为目标运行时的调用参数，并把结果归一化回 `SandboxExecResult`（exit code / timeout / oom / violation codes / resource summary）。失败分类冻结为：`launch_failed`、`capability_mismatch`、`timeout`、`oom`、`egress_denied`、`backend_unavailable`，复用既有 `sandbox-*-matrix`，不新建平行契约。出网由既有 egress 治理统一裁决（默认 deny）。

### Conformance and offline evidence

扩展 `integration/sandboxconformance` harness，对每类主流后端加一组 `sandbox-<backend>-matrix` fixture，覆盖 capability probe、正常执行、超时、oom、egress deny、backend unavailable 与 Run/Stream parity。新增 `scripts/check-sandbox-external-executor-conformance.*`（shell/PowerShell parity）纳入 gate 阻断。真实远端调用在 CI 无凭据时以 `skipped-unconfigured` 标记，不写成成功或失败。

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

## Test strategy

- Table-driven conformance：每类主流后端覆盖 capability probe、正常执行、超时、oom、egress deny、backend unavailable。
- 使用本地 fake/容器化 backend 做 offline 验收；远端 API 在无凭据 CI 中以 `skipped-unconfigured` 标记。
- Run/Stream parity：相同 spec 在 host 与 sandbox 路径下结果语义一致。
- 负向：executor nil、capability mismatch、egress deny、backend unavailable 均 fail-fast 并返回稳定 reason code。
- 配置校验：未注册的 external runtime、或 canonical 映射与 `executor.backend` 不一致时，`runtime/config.Validate` 与热更新 fail-fast 拒绝，错误路径为 `security.sandbox.executor.external_backend`。
