# Runtime Config & Diagnostics API

更新时间：2026-03-12

## 目标

为运行时提供统一配置入口（YAML + Env + Default）、热更新能力，以及仅库接口的诊断查询能力。

## 配置优先级

固定优先级：`env > file > default`

- `default`：由 `runtime/config.DefaultConfig()` 提供。
- `file`：YAML 文件（通过 `viper` 加载）。
- `env`：环境变量覆盖（前缀 + key 映射）。

## 环境变量映射

- 默认前缀：`BAYMAX`
- key 规则：`.` 替换为 `_`
- 示例：
  - `mcp.active_profile` -> `BAYMAX_MCP_ACTIVE_PROFILE`
  - `mcp.profiles.default.retry` -> `BAYMAX_MCP_PROFILES_DEFAULT_RETRY`
  - `reload.debounce` -> `BAYMAX_RELOAD_DEBOUNCE`

## YAML Schema（核心字段）

```yaml
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 10s
      retry: 1
      backoff: 50ms
      queue_size: 32
      backpressure: block # block|reject
      read_pool_size: 4
      write_pool_size: 1

concurrency:
  local_max_workers: 8
  local_queue_size: 32
  backpressure: block # block|reject

diagnostics:
  max_call_records: 200
  max_run_records: 200
  max_reload_errors: 100
  max_skill_records: 200

reload:
  enabled: true
  debounce: 200ms

provider_fallback:
  enabled: false
  providers: [openai, anthropic, gemini] # 有序候选链；enabled=true 时必须非空
  discovery_timeout: 1500ms
  discovery_cache_ttl: 5m

context_assembler:
  enabled: true # 默认 true
  journal_path: /tmp/baymax/context-journal.jsonl # 默认值由 os.TempDir() + /baymax/context-journal.jsonl 计算
  prefix_version: ca1
  storage:
    backend: file # CA1 支持 file；db 会 fail-fast 返回 unsupported
  guard:
    fail_fast: true # 默认 true
  ca2:
    enabled: false
    routing_mode: rules # rules|agentic（agentic 当前为预留占位）
    stage_policy:
      stage1: fail_fast    # fail_fast|best_effort
      stage2: best_effort  # fail_fast|best_effort
    timeout:
      stage1: 80ms
      stage2: 120ms
    stage2:
      provider: file       # file|rag|db（rag/db 当前返回 not-ready）
      file_path: /tmp/baymax/context-stage2.jsonl
    routing:
      min_input_chars: 120
      trigger_keywords: [search, retrieve, reference, lookup]
      require_system_guard: true
    tail_recap:
      enabled: true
      max_items: 4
      max_field_chars: 256

security:
  scan:
    mode: strict # strict|warn
    govulncheck_enabled: true
  redaction:
    enabled: true
    strategy: keyword # 当前仅支持 keyword，后续可扩展
    keywords: [token, password, secret, api_key, apikey]
```

## 使用示例（最小）

```go
mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
    FilePath:        "runtime.yaml",
    EnvPrefix:       "BAYMAX",
    EnableHotReload: true,
})
if err != nil {
    // fail-fast: 配置无效直接报错
    panic(err)
}
defer mgr.Close()

client := httpmcp.NewClient(httpmcp.Config{
    RuntimeManager: mgr,
    Profile:        mcpprofile.Default,
    Connect:        connector,
})
```

## 诊断 API（Library Only）

- `Manager.RecentCalls(n)`：最近 N 次 MCP 调用摘要。
- `Manager.RecentRuns(n)`：最近 N 次 run 摘要。
- `Manager.RecentReloads(n)`：最近 N 次热更新结果。
- `Manager.RecentSkills(n)`：最近 N 次 skill 生命周期摘要（discover/trigger/compile/failure）。
- `Manager.EffectiveConfigSanitized()`：脱敏后的生效配置快照。

当前不提供 CLI 诊断命令。

### Run 诊断新增字段（能力探测/降级）

- `model_provider`：最终执行 model step 的 provider。
- `fallback_used`：本次 run 是否发生 provider 降级。
- `fallback_initial`：候选链中的首选 provider。
- `fallback_path`：最终命中的 provider 路径（`a->b->...`）。
- `required_capabilities`：本次 preflight 的能力需求（逗号分隔）。
- `fallback_reason`：降级/终止原因摘要（例如 `capability_preflight_failed`）。

### Run 诊断新增字段（Context Assembler CA1）

- `prefix_hash`：本次 run 最近一次 assemble 的 immutable prefix 哈希。
- `assemble_latency_ms`：assemble 阶段耗时（毫秒）。
- `assemble_status`：assemble 状态（`success|failed|bypass`）。
- `guard_violation`：guard 命中摘要（失败时用于 fail-fast 诊断）。

### Run 诊断新增字段（Context Assembler CA2）

- `assemble_stage_status`：CA2 阶段结果（`stage1_only|stage2_used|degraded|bypass|failed`）。
- `stage2_skip_reason`：Stage2 跳过或降级原因（规则未命中/provider 不可用等）。
- `stage1_latency_ms`：Stage1 耗时（毫秒）。
- `stage2_latency_ms`：Stage2 耗时（毫秒）。
- `stage2_provider`：Stage2 使用的 provider（file/rag/db）。
- `recap_status`：tail recap 状态（`disabled|appended|truncated|failed`）。

## 诊断写入口径（Single Writer + Idempotency）

- 统一写入入口：`observability/event.RuntimeRecorder`。
- `core/runner`、`skill/loader` 负责产生标准事件，不直接落库诊断。
- `runtime/diagnostics.Store` 对 run/skill 记录执行幂等去重，避免重试/重放导致重复样本。

### 幂等键语义

- run 记录：按 `run_id + status`（无 `run_id` 时退化到稳定字段组合）去重。
- skill 记录：按 `run_id + skill_name + action + status + error_class + payload-hash` 去重。
- 动态字段（如 `latency_ms/time`）不参与 payload hash，保证重复重放可合并。

### 统一状态语义

- run 状态：`success | failed`
- skill 状态：`success | warning | failed`
- 错误分类：沿用 `types.ErrorClass` 语义（如 `ErrModel`、`ErrTool`、`ErrSkill`、`ErrSecurity`）

## 安全基线（S1）

- 质量门禁脚本（Linux/PowerShell）与 CI 默认执行 `govulncheck` 且使用 strict 语义。
- 可通过环境变量降级为 warn：`BAYMAX_SECURITY_SCAN_MODE=warn`。
- 可通过 `BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED=false` 临时关闭扫描（不建议在 CI 中关闭）。
- 统一脱敏管线覆盖：
  - `runtime/diagnostics`（配置快照与诊断 payload）
  - `observability/event`（JSON logger 与 runtime recorder）
  - `context/assembler`（stage2 payload 与 tail recap）
- 脱敏策略默认按 key 关键词匹配，支持扩展 matcher 接口（后续阶段可接入更复杂策略）。

## 热更新语义

- 触发机制：监听配置文件变更。
- 执行路径：`parse -> validate -> build snapshot -> atomic swap`。
- 失败策略：任一步失败则拒绝本次更新，保留旧快照，并写入 reload 诊断记录。

## 限制

- `mcp/stdio` 的 `read_pool_size` / `write_pool_size` 当前在 client 初始化时生效；热更新后不动态重建池大小。
- 脱敏规则基于 key 命名匹配（`secret/token/password/api_key`），后续可按需要扩展。
- `security.redaction.strategy` 当前仅支持 `keyword`，配置其他值会 fail-fast。
- provider fallback 仅在 model-step 边界进行，不支持流式响应开始后的 mid-stream 切换。
- context assembler CA1 仅提供文件 journal（append-only）；数据库后端仅接口占位，配置为 `db` 会启动即 fail-fast。
- context assembler CA2 的 `agentic` routing mode 与 `rag/db` provider 当前仅接口占位；配置后会返回明确 not-ready 错误。

## 迁移映射（功能命名）

- 全局运行时配置：`runtime/config`
- 全局运行时诊断：`runtime/diagnostics`
- MCP profile 语义：`mcp/profile`
- MCP retry 语义：`mcp/retry`
- MCP 调用摘要模型：`mcp/diag`

### 迁移示例

推荐写法：

```go
mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: "runtime.yaml"})
```
