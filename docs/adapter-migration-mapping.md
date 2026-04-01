# Adapter Migration Mapping (A21/A53/A54)

更新时间：2026-03-30

## 目标

提供统一迁移映射，按以下双维度组织：
- capability-domain（能力域）
- code-snippet（代码片段）

每条映射至少包含：
- previous pattern
- recommended pattern
- compatibility notes
- rollback notes
- conformance suite id

## Capability-Domain Mapping

| 能力域 | previous pattern | recommended pattern | compatibility notes |
| --- | --- | --- | --- |
| MCP adapter | 在业务代码中直接散落网络调用与重试逻辑 | 收敛到 `mcp/http` 或 `mcp/stdio` 客户端，并由 profile/runtime policy 管理 | additive 字段可增量引入；旧字段缺失走 default；非法配置 fail-fast |
| Model adapter | 直接在上层绑定 provider SDK 类型 | 在 `model/<provider>` 实现 `types.ModelClient` + 能力探测接口 | nullable 字段允许为空；新增能力字段保持 backward-safe |
| Tool adapter | 工具执行逻辑与业务主流程耦合 | 使用 `types.Tool` + `tool/local.Registry` 显式注册 | schema 变更需保持 additive 优先，破坏性变更需 fail-fast |
| Memory adapter | 直接依赖 legacy file-based memory 路径 | 迁移到 `runtime.memory` + `memory SPI`（`external_spi|builtin_filesystem`） | mode/profile/contract_version 对齐；required op 缺失 fail-fast；optional op 允许 deterministic downgrade |
| Adapter manifest | 接入前无统一兼容边界声明 | 为每个外部 adapter 提供 `adapter-manifest.json` 并在激活前校验 | `baymax_compat` 必须可解析；required fail-fast；optional downgrade 必须 deterministic |

## Code-Snippet Mapping

### MCP Adapter Integration

Previous pattern:

```go
// anti-pattern: network call and retry policy are mixed in business flow.
resp, err := rawHTTPCall(endpoint, payload)
if err != nil { return err }
```

Recommended pattern:

```go
client := mcpstdio.NewClient(transport, mcpstdio.Config{
    CallTimeout: 2 * time.Second,
    Retry:       1,
    Backoff:     50 * time.Millisecond,
})
defer client.Close()

res, err := client.CallTool(ctx, "echo", map[string]any{"input": "hello"})
_ = res
_ = err
```

Compatibility notes:
- additive: 允许新增可选配置字段，不影响旧调用路径。
- nullable: 可选配置为空时走默认策略。
- default: `Retry/Backoff/Timeout` 未设置时使用 runtime 默认值。
- fail-fast: 无效 endpoint/transport 初始化失败时立即报错。

### Model Adapter Integration

Previous pattern:

```go
// anti-pattern: provider SDK type leaks into upper layers.
raw := provider.NewClient(apiKey)
res := raw.Generate(...)
```

Recommended pattern:

```go
type customModelAdapter struct{}

func (customModelAdapter) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
    return types.ModelResponse{FinalAnswer: "ok"}, nil
}

func (customModelAdapter) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
    return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"})
}
```

Compatibility notes:
- additive: 新增响应字段必须不破坏已有 `RunResult` 语义。
- nullable: 可选能力字段为空时应返回明确 `unknown/unsupported`。
- default: 默认能力判定路径可回退到保守策略。
- fail-fast: 非法配置（model/key）必须在初始化阶段阻断。

### Tool Adapter Integration

Previous pattern:

```go
// anti-pattern: tool logic embedded in orchestration branch.
if action == "calc" { ... }
```

Recommended pattern:

```go
type calcTool struct{}

func (calcTool) Name() string { return "calc" }
func (calcTool) Description() string { return "calculate expression" }
func (calcTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (calcTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
    return types.ToolResult{Content: "42"}, nil
}
```

Compatibility notes:
- additive: schema 新增可选参数时保持旧参数可用。
- nullable: optional 参数为空时应有默认行为。
- default: 未配置策略时使用本地 dispatch 默认配置。
- fail-fast: schema 不合法或参数不匹配时立即失败，不静默忽略。

## Common Mistakes and Replacement Patterns

### MCP Category

- mistake: 在业务层重复实现重试与超时，导致语义漂移。
- replacement: 使用 `mcp/profile` + runtime policy 管理重试、超时与背压。

- mistake: adapter 初始化失败后继续降级执行导致隐式不一致。
- replacement: 初始化阶段 fail-fast，并在 diagnostics 中标注分类错误。

### Model Category

- mistake: provider SDK 对象直接向上暴露，污染核心接口边界。
- replacement: 收敛为 `types.ModelClient`，并保持事件/错误语义一致。

- mistake: Stream 与 Run 路径输出语义不对齐。
- replacement: 对齐终态字段与错误分类，保持 run/stream 语义等价。

### Tool Category

- mistake: 工具名称未遵循 `local.*` 命名空间，难以治理。
- replacement: 通过 `tool/local.Registry` 统一注册并显式命名。

- mistake: schema 演进时直接删除旧字段，触发下游兼容问题。
- replacement: 优先 additive 变更，删除前提供迁移窗口与替代字段。

## Unified Compatibility Boundary

所有 adapter 迁移遵循统一边界语义：`additive + nullable + default + fail-fast`。

- additive: 新字段应以非破坏方式增量引入。
- nullable: 可选字段允许为空，并定义空值语义。
- default: 缺省值必须可预测并在文档中可查。
- fail-fast: 非法配置/非法输入必须在边界处快速失败。

该边界与仓库兼容策略保持一致：`docs/versioning-and-compatibility.md`。

## Manifest Migration Notes（A26）

迁移到 A26 后，建议在 adapter 根目录补齐 `adapter-manifest.json`，最小结构：

```json
{
  "type": "model",
  "name": "demo-model",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["model.run_stream.semantic_equivalent", "model.response.mandatory_fields"],
    "optional": ["model.capability.token_count"]
  },
  "negotiation": {
    "default_strategy": "fail_fast",
    "allow_request_override": true
  },
  "conformance_profile": "model-run-stream-downgrade"
}
```

兼容语义补充：
- `baymax_compat` 不命中时，接入边界必须 fail-fast，不允许隐式继续。
- `contract_profile_version` 必填，且必须是运行时识别的 profile 标签。
- `capabilities.required` 缺失时必须 fail-fast，错误分类保持 deterministic。
- `capabilities.optional` 缺失允许降级，并保留可回放的 downgrade reason code。
- `negotiation.default_strategy` 默认建议 `fail_fast`；非法策略值必须在接入边界 fail-fast。
- `negotiation.allow_request_override=true` 时可按请求覆盖到 `best_effort`，并记录 override reason taxonomy。
- `conformance_profile` 与执行场景不一致时，conformance harness 必须阻断。

## A22 Conformance 对齐

迁移完成后建议执行：

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

```bash
bash scripts/check-adapter-capability-contract.sh
```

```powershell
pwsh -File scripts/check-adapter-capability-contract.ps1
```

若 conformance 失败，优先检查：
- 模板实现是否仍满足 capability-domain 对照关系；
- reason taxonomy 是否保持 namespaced 规范；
- optional capability 降级行为是否仍 deterministic；
- negotiation 默认策略与 override 开关是否与 conformance profile 对齐。

## Profile Versioning Migration Notes（A28）

A28 已补齐 profile version 与 replay gate，迁移建议如下：
- 在 adapter 合同元数据中显式维护 `contract_profile_version`（当前基线 `v1alpha1`）。
- 将 profile 版本与 `conformance_profile` 一起纳入发布记录，避免“版本已升级但验收矩阵未切换”。
- 为 manifest/negotiation/reason taxonomy 维护最小 replay fixture，升级后先跑回放再放量。

约束提醒：
- 若 profile 不在 runtime 支持窗口内，应 fail-fast，而不是隐式降级继续执行。
- 回放基线出现漂移时优先修复契约差异，再更新 fixture，避免“用新基线覆盖旧问题”。

回放 gate 命令：

```bash
bash scripts/check-adapter-contract-replay.sh
```

```powershell
pwsh -File scripts/check-adapter-contract-replay.ps1
```

## A53 Sandbox Wrapper -> Profile-Pack Adapter Mapping

| legacy wrapper pattern | target profile-pack pattern | compatibility notes | rollback notes | conformance suite id |
| --- | --- | --- | --- | --- |
| Linux NSJail wrapper（脚本内拼装 nsjail args） | `sandbox_backend=linux_nsjail` + `sandbox_profile_id=linux_nsjail` + `session_modes_supported=["per_call","per_session"]` | 保持 capability taxonomy 与 reason taxonomy 不变；新增字段遵循 additive + nullable + default + fail-fast | 回滚到 `security.sandbox.enabled=false` 或切回原 wrapper 分支，保留 manifest 字段以便复跑 conformance | `sandbox-linux-nsjail-matrix` |
| Linux Bubblewrap wrapper（脚本内拼装 bwrap args） | `sandbox_backend=linux_bwrap` + `sandbox_profile_id=linux_bwrap` | host 约束固定 `host_os=linux`、`host_arch=amd64`；session mismatch 必须 fail-fast | 回滚时仅回滚 backend/profile 字段，不删除 manifest 其他兼容字段 | `sandbox-linux-bwrap-matrix` |
| OCI runtime wrapper（docker/runc wrapper） | `sandbox_backend=oci_runtime` + `sandbox_profile_id=oci_runtime` | backend/profile drift 必须由 gate 阻断；允许 optional capability downgrade | 可临时回退到 legacy wrapper，但必须保留 replay fixture 并验证 mixed profile 不回归 | `sandbox-oci-runtime-matrix` |
| Windows Job wrapper（PowerShell 启动作业对象） | `sandbox_backend=windows_job` + `sandbox_profile_id=windows_job` | host mismatch 与 session unsupported 必须 deterministic fail-fast | 回滚到旧 wrapper 时保留 `sandbox_profile_id`，避免后续迁移无法对账 | `sandbox-windows-job-matrix` |

迁移后最小验收命令：

```bash
bash scripts/check-sandbox-adapter-conformance-contract.sh
```

```powershell
pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1
```

## A54 Memory Legacy Path -> Unified SPI Mapping

| legacy memory pattern | target pattern | compatibility notes | rollback notes | conformance suite id |
| --- | --- | --- | --- | --- |
| 直接访问历史 file-based memory 目录与文件协议 | `runtime.memory.mode=builtin_filesystem` + `runtime.memory.builtin.*`（WAL + compaction） | Query/Upsert/Delete 语义保持一致；reason taxonomy 统一到 `memory.*`；readiness 覆盖 `memory.filesystem_path_invalid` | 可回滚到 legacy 路径，但保留 `runtime.memory.*` 字段与 replay 夹具，保证可对账 | `memory-builtin-filesystem-switch` |
| provider SDK 直接嵌入主流程 | `runtime.memory.mode=external_spi` + profile-pack（`mem0|zep|openviking|generic`） + manifest `memory.*` | manifest `memory.provider/profile/contract_version/operations/fallback` 必填；required op 缺失 fail-fast；optional op 输出 deterministic downgrade | 回滚 builtin 时仅切换 mode/profile，不删除 manifest memory 字段，避免后续 conformance 漂移 | `memory-mem0-matrix` / `memory-zep-matrix` / `memory-openviking-matrix` / `memory-generic-matrix` |

迁移 acceptance checklist（每条迁移记录必须绑定）：
- diagnostics 字段：`memory_mode`、`memory_provider`、`memory_profile`、`memory_contract_version`、`memory_query_total`、`memory_upsert_total`、`memory_delete_total`、`memory_error_total`、`memory_fallback_total`、`memory_fallback_reason_code`。
- readiness finding：`memory.mode_invalid`、`memory.profile_missing`、`memory.provider_not_supported`、`memory.spi_unavailable`、`memory.filesystem_path_invalid`、`memory.contract_version_mismatch`、`memory.fallback_policy_conflict`、`memory.fallback_target_unavailable`。
- conformance suites：`integration/adapterconformance/memory_matrix_test.go`、`tool/diagnosticsreplay/arbitration_test.go`（`memory.v1` fixtures）。

## A57 Sandbox Egress + Adapter Allowlist Migration Mapping

| legacy pattern | target pattern | compatibility notes | rollback notes | conformance suite id |
| --- | --- | --- | --- | --- |
| 未治理的 sandbox 外呼（默认放开网络） | `security.sandbox.egress.default_action=deny` + `allowlist/by_tool/on_violation` 显式策略 | deny-first 保持安全默认；允许按 host/tool 精确放开；输出 `sandbox_egress_action/policy_source` additive 字段 | 紧急回滚可先把 `security.sandbox.egress.enabled=false` 或切回 `default_action=deny + 空 allowlist`，保留字段与 fixture 以便审计对账 | `sandbox-egress-deny-matrix` |
| 仅按“全局开关”放开网络，无 host 维度边界 | `security.sandbox.egress.allowlist` + `security.sandbox.egress.by_tool` 组合策略 | allowlist 匹配与 selector override 优先级固定，避免“同请求不同入口”判定漂移 | 若回滚 selector 规则，保留 allowlist 与 replay fixture，避免 taxonomy 漂移 | `sandbox-egress-allow-matrix` / `sandbox-egress-selector-override-precedence` |
| 违规外呼直接硬失败，缺少观测降级路径 | `security.sandbox.egress.on_violation=allow_and_record`（仅在显式需要时启用） | 可在不放松默认 deny 的前提下保留主链路可观测降级；reason taxonomy 固定 `sandbox.egress_allow_and_record` | 回滚时恢复 `on_violation=deny` 并保留计数器字段，保证趋势连续性 | `sandbox-egress-allow-and-record-matrix` |
| adapter 激活不做供应链准入（仅靠运行时兜底） | `adapter.allowlist.*` 激活前 fail-fast（id/publisher/version/signature） | 未命中 entry 与签名非法在激活边界阻断；readiness/admission 可解释透传 | 回滚时可切 `adapter.allowlist.enabled=false`，但保留 manifest allowlist identity 字段与 case 数据 | `adapter-allowlist-missing-entry-enforce` / `adapter-allowlist-signature-invalid-enforce` |
| allowlist 策略枚举不统一（不同模块含义不一致） | enforcement `observe|enforce` + unknown-signature `deny|allow_and_record` 固化 | 策略冲突在配置/激活边界 fail-fast；taxonomy 固定 `adapter.allowlist.*` | 回滚时不得删除策略字段，避免热更新路径出现 schema 漂移 | `adapter-allowlist-policy-conflict` |
| allowlist 命中成功路径无稳定验收 | 命中路径进入正常激活并维持 Run/Stream 语义等价 | 成功路径不引入额外阻断副作用，且 diagnostics additive 字段可回放 | 回滚仅调整策略，不回滚 conformance case id 绑定 | `adapter-allowlist-allowed-path-enforce` |

A57 最小验收命令：

```bash
bash scripts/check-sandbox-egress-allowlist-contract.sh
```

```powershell
pwsh -File scripts/check-sandbox-egress-allowlist-contract.ps1
```
