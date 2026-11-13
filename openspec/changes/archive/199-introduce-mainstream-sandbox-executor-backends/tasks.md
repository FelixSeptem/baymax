## Implementation Tasks

### Contract, adapter and owner implementation

- [x] 在 `runtime/config` 注册主流沙箱运行时 → canonical backend profile 映射（e2b / open-sandbox / anthropics-sandbox-runtime → `oci_runtime`；`local` 为 onboarding-only），并在 `adapter/sandboxexternal` 实现对应的 `SandboxExecutor` adapter 骨架（含 exec spec 翻译与 `SandboxExecResult` 归一化）。
- [x] 实现 capability probe 协商：在 readiness/admission 路径对 `required_capabilities` 不满足者 deterministic deny；executor 为 nil 时返回 `sandboxReasonLaunchFailed`。
- [x] 冻结外部 adapter 失败分类（launch_failed / capability_mismatch / timeout / oom / egress_denied / backend_unavailable），复用既有 `sandbox-*-matrix`，不新建平行契约。
- [x] `runtime/config` 增加 backend 注册与校验，默认 `Disabled`，沿用 `observe → enforce` 与 `DefaultAction=host` 回退。
- [x] 扩展 `integration/sandboxconformance` harness，为每类主流后端新增 `sandbox-<backend>-matrix` fixture（capability probe / 正常 / 超时 / oom / egress deny / backend unavailable / Run-Stream parity）。实现拆为 `ExternalBackendMatrix`（canonical 映射 + 能力协商）与 `ExternalFailureFixtures`（runtime × kind 失败分类）；Run/Stream 等价由 `ExternalRunStreamParity` 校验。

### Evidence, documentation, and gates

- [x] 新增 `scripts/check-sandbox-external-executor-conformance.ps1` 与 `.sh`（shell/PowerShell parity），纳入 gate 阻断；无凭据 CI 以 `skipped-unconfigured` 标记真实远端调用。
- [x] 更新 `README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/runtime-config-diagnostics.md`、`docs/external-adapter-template-index.md`、`docs/adapter-migration-mapping.md`、`adapter/manifest/README.md`。
- [x] 在 `examples/` 增加主流后端 onboarding 示例（offline 可跑），打印 backend 注册与 conformance marker。
- [x] 向所有 change 工件补齐 `Public Semantic Documentation Commitment` 与 `Example Impact Assessment`，运行 OpenSpec/文档治理校验。（`check-openspec-example-impact-declaration` passed、`check-openspec-documentation-impact` passed、`check-docs-consistency` passed、`openspec validate --all` 154 passed/0 failed）
- [x] 运行受影响包测试、`go test -race`、`golangci-lint`、`check-quality-gate`、`check-docs-consistency`、`openspec validate --all`、`git diff --check`，记录结果后归档。

验证记录（2026-11-13）：

- 受影响包：`go test ./adapter/sandboxexternal/... ./adapter/manifest/... ./integration/sandboxconformance/... ./runtime/config/... ./tool/local/...` 全 ok；`go test -race` 同范围全 ok；`go vet` 0 问题。
- `golangci-lint run --config .golangci.yml`（新增/改动包）0 issues；默认 `run.timeout: 10m` 会在 `runtime/config` 上超时，需 `--timeout 25m`。
- `check-docs-consistency.sh` passed；`check-openspec-example-impact-declaration.sh` passed；`check-openspec-documentation-impact.sh` passed；`openspec validate --all` 154 passed / 0 failed；`git diff --check` 无空白错误。
- `check-quality-gate.sh` 91 步中 90 步 passed，最后一步 `go test ./...` 撞 1800s 总预算被终止（`step timeout`，非断言失败）。单独全量 `go test ./...` 复跑：改动相关包全绿，5 个失败全部为既有环境问题——`cmd/host-jsonl` 3 例在干净 `master` 快照上同样失败；`integration::TestDelayedDispatchContractRunStreamSemanticEquivalence` 是负载下 p95 抖动（单独复跑通过）；`tool/contributioncheck::TestAgentModeDocFirstDeliveryContractExecutionPassFailAndExceptionalBranches` 因 PS 子进程解析不到 `git` 而失败（本机 `PATHEXT` 被置为 `.CPL` 的既有环境问题）。

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
