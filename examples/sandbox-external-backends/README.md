# Sandbox External Backends Onboarding

## 目的

演示主流沙箱运行时如何接入既有 `SandboxExecutor` 契约：`e2b`、`open-sandbox`、
`anthropic-sandbox-runtime` 统一映射到 canonical `oci_runtime`，`local` 作为
offline onboarding/一致性验证后端。

示例完全离线可跑，不访问网络、不需要任何凭据。它验证四件事：

1. 默认姿态不变：`external_backend` 为空（`Disabled`）、`mode=observe`、`default_action=host`。
2. `e2b` 必须映射到 `oci_runtime`；映射不一致时配置校验 fail-fast。
3. `integration/sandboxconformance` 的外部后端矩阵通过（capability / session mode / fallback）。
4. `local` 后端真实执行一条命令；未配置端点的 `e2b` 返回 `sandbox.backend_unavailable`。

## 运行

```bash
go run ./examples/sandbox-external-backends
```

PowerShell：

```powershell
go run ./examples/sandbox-external-backends
```

期望输出（顺序固定，命令输出内容随平台略异）：

```text
== sandbox external backends onboarding ==
registered backends: anthropic-sandbox-runtime, e2b, local, open-sandbox
DEFAULT enabled=false mode=observe default_action=host external=""
MAPPING external="e2b" canonical="oci_runtime" resolved=true
MISMATCH_REJECTED security.sandbox.executor.external_backend "e2b" requires backend "oci_runtime", got "windows_job"
CONFORMANCE=pass scenarios=3
LOCAL_EXEC exit=0 stdout="hello from sandbox external onboarding"
CLOUD_EXEC reason=sandbox.backend_unavailable
RESULT=ok
```

## 接真实运行时（可选）

云端 adapter 的请求会发往 `security.sandbox.executor.external_base_url` 下的
归一化 exec 端点（各 adapter 的 path 见 `adapter/sandboxexternal`）。onboarding
阶段推荐用 docker 起本地服务，再通过配置或环境变量指过去：

```yaml
security:
  sandbox:
    enabled: true
    mode: observe
    executor:
      backend: oci_runtime
      external_backend: e2b
      external_base_url: http://127.0.0.1:18080
```

```bash
export BAYMAX_SECURITY_SANDBOX_EXECUTOR_EXTERNAL_BACKEND=e2b
export BAYMAX_SECURITY_SANDBOX_EXECUTOR_EXTERNAL_BASE_URL=http://127.0.0.1:18080
```

`external_base_url` 为空时 adapter 保持 `sandbox.backend_unavailable`，不会阻断进程启动，
也不会把失败伪装成成功。

## 边界

- `local` 后端直接在宿主进程树执行命令，**不是隔离后端**，只用于 onboarding 与离线一致性验证。
- 本示例不声明生产级隔离；真实隔离由 cloud adapter 或 canonical nsjail/bwrap/oci/windows-job 提供。
- 未配置端点时真实远端调用保持 skipped/unavailable，不写成代码失败或成功。
