# adapter/manifest

Adapter manifest 装载、兼容协商与 sandbox profile 校验收敛点。

## 职责边界

- 解析 adapter manifest（JSON/YAML）并做字段级校验。
- 协商 conformance contract profile 版本窗口与 required capability。
- 校验 sandbox profile 与 canonical backend 的兼容性（host OS/arch、session mode）。
- 校验 adapter allowlist 条目（publisher/version/signature status）。

该包只做「声明面」校验，不实现任何真实沙箱隔离执行器；执行器由 `core/types.SandboxExecutor` 接口注入，具体后端见 `adapter/sandboxexternal`。

## Canonical Sandbox Backends

`defaultSandboxProfilePack` 固定四种 canonical backend，外部运行时必须映射到其中之一：

| Canonical backend | Host OS | Session modes |
| --- | --- | --- |
| `linux_nsjail` | linux | `per_call`, `per_session` |
| `linux_bwrap` | linux | `per_call`, `per_session` |
| `oci_runtime` | linux | `per_call`, `per_session` |
| `windows_job` | windows | `per_call`, `per_session` |

对外入口：

- `SandboxProfileByID(id)`：按 profile ID 取 canonical 画像。
- `SupportedSandboxBackends()` / `IsSupportedSandboxBackend(backend)`：枚举与判定 canonical backend。

## External Runtime Mapping

主流沙箱运行时通过 `runtime/config` 的 `security.sandbox.executor.external_backend` 选择，映射关系是唯一事实源（`SecuritySandboxExternalBackendCanonical`）：

| External runtime selector | Canonical backend | Adapter 包 |
| --- | --- | --- |
| `e2b` | `oci_runtime` | `adapter/sandboxexternal` |
| `open-sandbox` | `oci_runtime` | `adapter/sandboxexternal` |
| `anthropic-sandbox-runtime` | `oci_runtime` | `adapter/sandboxexternal` |
| `local` | 任意 canonical（onboarding-only，不声明隔离） | `adapter/sandboxexternal` |

映射不一致会在启动/热更新时 fail-fast，错误码路径为 `security.sandbox.executor.external_backend`。默认值为空字符串，即 `Disabled`。

## 失败分类

manifest 侧 sandbox 相关错误码：

- `adapter-manifest.sandbox-profile-unknown`
- `adapter-manifest.sandbox-backend-unsupported`
- `adapter-manifest.sandbox-host-mismatch`
- `adapter-manifest.sandbox-session-mode-unsupported`

外部执行器失败分类（`adapter/sandboxexternal`）复用既有 sandbox taxonomy：`sandbox.launch_failed`、`sandbox.capability_mismatch`、`sandbox.timeout`、`sandbox.oom`、`sandbox.egress_denied`、`sandbox.backend_unavailable`。

## 验收

```bash
go test ./adapter/manifest/... ./adapter/sandboxexternal/... ./integration/sandboxconformance/... -count=1
bash scripts/check-sandbox-external-executor-conformance.sh
```

```powershell
go test ./adapter/manifest/... ./adapter/sandboxexternal/... ./integration/sandboxconformance/... -count=1
pwsh -File scripts/check-sandbox-external-executor-conformance.ps1
```
