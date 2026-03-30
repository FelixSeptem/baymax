# External Adapter Template Index (A21/A53)

更新时间：2026-03-30

## 目标

提供外部接入的最小模板入口，降低新接入方在 `MCP / Model / Tool` 三类适配上的理解与迁移成本。
同时提供主流 sandbox backend 的 profile-pack onboarding 模板，保证接入命令与 conformance 套件可直接执行。

优先级（固定）：
1. MCP adapter template
2. Model provider adapter template
3. Tool adapter template

模板边界：
- 仅用于 onboarding skeleton 和迁移参考。
- 不提供生产级运行保障（多租户隔离、SLO、审计、全量安全策略）。
- A26 起，模板默认携带 `adapter-manifest.json`，用于接入前 compatibility contract 校验。
- A27 起，脚手架默认携带 `capability_negotiation_test.go`，用于协商与回退契约 baseline。

## 模板导航

| 优先级 | 模板 | 位置 | 运行命令 |
| --- | --- | --- | --- |
| P1 | MCP adapter template | `examples/templates/mcp-adapter-template/main.go` | `go run ./examples/templates/mcp-adapter-template` |
| P2 | Model provider adapter template | `examples/templates/model-adapter-template/main.go` | `go run ./examples/templates/model-adapter-template` |
| P3 | Tool adapter template | `examples/templates/tool-adapter-template/main.go` | `go run ./examples/templates/tool-adapter-template` |

## Mainstream Sandbox Backend Onboarding Templates（A53）

适用 backend（固定）：
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job`

模板字段索引（profile-pack adapter manifest）：

| 字段 | 类型 | 默认值（模板） | 说明 |
| --- | --- | --- | --- |
| `sandbox_backend` | string | 无（必填） | 必须是 `linux_nsjail|linux_bwrap|oci_runtime|windows_job` 之一 |
| `sandbox_profile_id` | string | 与 `sandbox_backend` 相同 | 绑定 profile-pack 条目 |
| `host_os` | string | 由 profile-pack 决定（Linux 或 Windows） | 激活边界 host 兼容校验 |
| `host_arch` | string | `amd64` | 激活边界 host 兼容校验 |
| `session_modes_supported` | []string | `["per_call","per_session"]` | adapter 声明支持的 session 模式 |

后端模板映射（onboarding skeleton）：

| backend | profile-pack id | conformance suite id | Linux/macOS | Windows |
| --- | --- | --- | --- | --- |
| `linux_nsjail` | `linux_nsjail` | `sandbox-linux-nsjail-matrix` | `bash scripts/check-sandbox-adapter-conformance-contract.sh` | `pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1` |
| `linux_bwrap` | `linux_bwrap` | `sandbox-linux-bwrap-matrix` | `bash scripts/check-sandbox-adapter-conformance-contract.sh` | `pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1` |
| `oci_runtime` | `oci_runtime` | `sandbox-oci-runtime-matrix` | `bash scripts/check-sandbox-adapter-conformance-contract.sh` | `pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1` |
| `windows_job` | `windows_job` | `sandbox-windows-job-matrix` | `bash scripts/check-sandbox-adapter-conformance-contract.sh` | `pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1` |

manifest 片段（profile declaration + manifest snippet）：

```json
{
  "type": "tool",
  "name": "sandbox-tool",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast",
  "sandbox_backend": "linux_nsjail",
  "sandbox_profile_id": "linux_nsjail",
  "host_os": "linux",
  "host_arch": "amd64",
  "session_modes_supported": ["per_call", "per_session"]
}
```

## MCP Adapter Template（P1）

最小可运行片段（`examples/templates/mcp-adapter-template/main.go`）展示：
- `mcp/stdio` 客户端接入；
- transport 协议骨架（`Initialize/ListTools/CallTool/Close`）；
- 本地最小调用链路。

边界说明：
- 模板使用 `fakeTransport` 演示结构，不代表真实网络传输实现。
- 生产接入时需要补齐：鉴权、重试预算、连接生命周期、告警与审计。
- 适配层不应穿透 `core/runner` 内部状态机语义。

## Model Adapter Template（P2）

最小可运行片段（`examples/templates/model-adapter-template/main.go`）展示：
- 实现 `types.ModelClient`（`Generate/Stream`）；
- 通过 `runner.New` 进入统一 Run 链路。

边界说明：
- 模板聚焦接口收敛，不覆盖 provider SDK 的生产化治理策略。
- provider-specific 协议映射应保持在 `model/<provider>` 子域，不泄漏到 `core/*`。
- 生产接入需补齐错误分类与 capability discovery。

## Tool Adapter Template（P3）

最小可运行片段（`examples/templates/tool-adapter-template/main.go`）展示：
- 实现 `types.Tool`；
- 通过 `tool/local.Registry` 注册；
- 在 runner 闭环中触发工具调用并收敛终态。

边界说明：
- 模板为单工具骨架，不包含复杂 schema 演进与安全过滤策略。
- 工具超时、限流、幂等与副作用控制需由业务侧补齐。
- 生产场景建议结合 `runtime/security` 与 action gate 规则治理。

## 相关文档

- 迁移映射：`docs/adapter-migration-mapping.md`
- API 参考入口：`docs/api-reference-d1.md`
- 运行时配置与诊断：`docs/runtime-config-diagnostics.md`

## Manifest Template Guidance（A26）

从 A26 起，脚手架模板默认生成 `adapter-manifest.json`，字段包含：
- `type`
- `name`
- `version`
- `contract_profile_version`
- `baymax_compat`
- `capabilities.required`
- `capabilities.optional`
- `conformance_profile`

约束要点：
- `baymax_compat` 使用 semver range，可包含 `-rc` 预发布表达。
- `required` 能力缺失时必须 fail-fast。
- `optional` 能力缺失允许降级，但必须输出 deterministic downgrade reason。
- `conformance_profile` 必须与 conformance bootstrap 场景 ID 一致（避免模板漂移）。

## Conformance 验收入口（A22）

模板交付后，必须通过 A22 一致性验收：

```bash
bash scripts/check-adapter-conformance.sh
```

```powershell
pwsh -File scripts/check-adapter-conformance.ps1
```

```bash
bash scripts/check-adapter-manifest-contract.sh
```

```powershell
pwsh -File scripts/check-adapter-manifest-contract.ps1
```

验收覆盖：
- MCP/Model/Tool 最小矩阵（优先级 `MCP > Model > Tool`）
- 默认离线执行（stub/fake）
- run/stream 语义等价（适用项）
- 错误分类与 reason taxonomy 归一
- mandatory input fail-fast

## Capability Negotiation Scaffold Guidance（A27）

从 A27 起，脚手架在保留 `adapter-manifest.json` 的同时，新增：
- `capability_negotiation_test.go`：覆盖 fail-fast、best-effort override、run/stream 协商等价基线。
- manifest `negotiation` 段默认值：
  - `default_strategy: fail_fast`
  - `allow_request_override: true`

协商语义约束：
- required capability 缺失必须 fail-fast（`adapter.capability.missing_required`）。
- optional capability 在 `best_effort` 下允许降级（`adapter.capability.optional_downgraded`）。
- 请求策略覆盖生效时记录 `adapter.capability.strategy_override_applied`。

A27 契约验收入口：

```bash
bash scripts/check-adapter-capability-contract.sh
```

```powershell
pwsh -File scripts/check-adapter-capability-contract.ps1
```

## Profile Versioning & Replay Guidance（A28）

A28 在 adapter 合同链路补齐了 profile version 与 replay gate：
- 在 manifest/conformance/negotiation 链路统一引入 `contract_profile_version`。
- runtime 侧执行 profile 支持窗口校验（默认 `current + previous`，不命中 fail-fast）。
- 增加 replay 基线用于识别契约语义漂移（manifest/compat/negotiation/reason taxonomy）。

回放 gate 命令：

```bash
bash scripts/check-adapter-contract-replay.sh
```

```powershell
pwsh -File scripts/check-adapter-contract-replay.ps1
```

Sandbox adapter conformance gate（A53）：

```bash
bash scripts/check-sandbox-adapter-conformance-contract.sh
```

```powershell
pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1
```
