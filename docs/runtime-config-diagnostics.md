# Runtime Config & Diagnostics API

更新时间：2026-03-16

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

action_gate:
  enabled: true
  policy: require_confirm # allow|require_confirm|deny
  timeout: 15s            # resolver 超时统一按 deny 处理
  tool_names: []          # 首期风险规则：tool name
  keywords: []            # 首期风险规则：keyword（匹配 input + args 文本）
  decision_by_tool:       # 可选：按 tool 定制决策
    shell: require_confirm
    delete: deny
  decision_by_keyword:    # 可选：按 keyword 定制决策
    "rm -rf": deny
    "drop table": require_confirm
  parameter_rules:
    - id: require-confirm-shell-danger
      tool_names: [shell]
      action: require_confirm # 可选，不填则继承 action_gate.policy
      condition:
        all:
          - path: cmd
            operator: contains # eq|ne|contains|regex|in|not_in|gt|gte|lt|lte|exists
            expected: rm -rf
          - path: force
            operator: eq
            expected: true

clarification:
  enabled: true
  timeout: 30s
  timeout_policy: cancel_by_user # 当前仅支持 cancel_by_user

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
      provider: file       # file|http|rag|db|elasticsearch
      file_path: /tmp/baymax/context-stage2.jsonl
      external:
        profile: http_generic # http_generic|ragflow_like|graphrag_like|elasticsearch_like
        endpoint: https://retriever.example.com/search # non-file provider 必填
        method: POST # POST|PUT
        auth:
          bearer_token: ${RETRIEVER_TOKEN}
          header_name: Authorization
        headers:
          X-Tenant: demo
        mapping:
          request:
            mode: plain # plain|jsonrpc2
            method_name: "" # mode=jsonrpc2 时必填
            jsonrpc_version: "2.0"
            query_field: query
            session_id_field: session_id
            run_id_field: run_id
            max_items_field: max_items
          response:
            chunks_field: chunks
            source_field: source
            reason_field: reason
            error_field: error
            error_message_field: error.message
    routing:
      min_input_chars: 120
      trigger_keywords: [search, retrieve, reference, lookup]
      require_system_guard: true
    tail_recap:
      enabled: true
      max_items: 4
      max_field_chars: 256
  ca3:
    enabled: true
    max_context_tokens: 128000
    goldilocks_min_percent: 35
    goldilocks_max_percent: 60
    percent_thresholds:
      safe: 20
      comfort: 40
      warning: 60
      danger: 75
      emergency: 90
    absolute_thresholds:
      safe: 24000
      comfort: 48000
      warning: 72000
      danger: 96000
      emergency: 115200
    stage1:
      percent_thresholds: {}  # 可选，空表示沿用全局阈值
      absolute_thresholds: {}
    stage2:
      percent_thresholds: {}
      absolute_thresholds: {}
    protection:
      critical_keywords: [critical]
      immutable_keywords: [immutable]
    squash:
      enabled: true
      max_content_runes: 320
    prune:
      enabled: true
      target_percent: 55
      keyword_priority: [error, decision, constraint, risk, todo]
    emergency:
      reject_low_priority: true
      high_priority_tokens: [urgent, critical, incident]
    spill:
      enabled: true
      backend: file # file|db|object（db/object 当前仅占位）
      path: /tmp/baymax/context-spill.jsonl
      swap_back_limit: 4
    tokenizer:
      mode: sdk_preferred # sdk_preferred|estimate_only
      provider: anthropic # anthropic|gemini|openai
      model: claude-3-5-sonnet-latest
      small_delta_tokens: 256
      sdk_refresh_interval: 1200ms

security:
  scan:
    mode: strict # strict|warn
    govulncheck_enabled: true
  redaction:
    enabled: true
    strategy: keyword # 当前仅支持 keyword，后续可扩展
    keywords: [token, password, secret, api_key, apikey]
```

CA4 阈值解析顺序：
1. 若 stage1/stage2 覆盖阈值被配置（且校验通过），该 stage 使用覆盖值，不与全局阈值混用。
2. percent 与 absolute 阈值并行计算分区。
3. 两者冲突时选取更高压力分区，并写入 `ca3_pressure_reason` + `ca3_pressure_trigger`。

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
- `Manager.PrecheckStage2External(provider, external)`：CA2 external retriever 预检查（warning 可继续，error 需 fail-fast）。

当前不提供 CLI 诊断命令。

## CA3 Token Count 职责分工

- `context/assembler`：只负责“何时计数”的策略决策（`sdk_preferred`、`small_delta_tokens`、`sdk_refresh_interval`），不直接依赖 provider SDK 细节。
  - 预估路径：优先使用本地 `tiktoken` 进行估算；若本地 tokenizer 初始化失败（如离线环境未缓存词表），回退到轻量字符估算以保证主流程不中断。
- `model/*`：负责“如何计数”的 provider 实现与官方 SDK 调用：
  - `model/anthropic`：`Messages.CountTokens`。
  - `model/gemini`：优先 `genai/tokenizer` 本地计数，失败时回退 `Models.CountTokens`。
  - `model/openai`：当前适配层未提供官方可复用 token-count API，返回 unsupported 并由上层回退预估。
- 语义要求：
  - 小增量优先预估，降低高频计数调用成本。
  - SDK 计数失败不阻断主流程，回退预估值继续执行。
  - OpenAI 路径的 token 数用于 CA3/CA4 阈值策略控制，不承诺账单精度语义。

## Action Timeline 事件（默认启用）

- 事件类型：`action.timeline`
- 产出路径：由 `core/runner` 发射，经 `observability/event` 统一输出（logger/handler 可直接消费）。
- phase 枚举：`run|context_assembler|model|tool|mcp|skill|hitl`
- status 枚举：`pending|running|succeeded|failed|skipped|canceled`
- payload 最小字段：
  - `phase`：动作阶段
  - `status`：阶段状态
  - `reason`：可选，失败/跳过/取消原因
  - `sequence`：单 run 内递增序号（用于稳定排序）

兼容性：该事件为增量新增，不替换既有 `run.* / model.* / tool.* / skill.*` 事件。

### External Retriever 预检查语义

- 预检查输出包含 findings（`warning`/`error`）与归一化配置快照。
- `warning`：非阻断，可继续启动或热更新，但建议记录到观测并尽快修复。
- `error`：阻断，启动或热更新必须 fail-fast，保留旧快照。

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
- `stage2_provider`：Stage2 使用的 provider（file/http/rag/db/elasticsearch）。
- `stage2_profile`：Stage2 external profile（`http_generic|ragflow_like|graphrag_like|elasticsearch_like|file`）。
- `stage2_hit_count`：Stage2 本次命中的 chunk 数量。
- `stage2_source`：Stage2 数据源标识（provider 或响应映射字段）。
- `stage2_reason`：Stage2 执行原因/结果摘要（如 `ok`/`empty`/`timeout`/`fetch_error`）。
- `stage2_reason_code`：Stage2 机器可读原因码（如 `ok`/`timeout`/`http_status`/`upstream_error`）。
- `stage2_error_layer`：Stage2 错误分层（`transport|protocol|semantic`，成功时为空）。
- `recap_status`：tail recap 状态（`disabled|appended|truncated|failed`）。

### Run 诊断新增字段（Context Assembler CA3）

- `ca3_pressure_zone`：CA3 当前压力分区（`safe|comfort|warning|danger|emergency`）。
- `ca3_pressure_reason`：分区触发来源（`usage_percent_trigger|absolute_token_trigger`）。
- `ca3_pressure_trigger`：本次最终触发分区（双触发冲突时记录被选中的更高压力分区）。
- `ca3_zone_residency_ms`：各分区累计停留时长（毫秒）。
- `ca3_trigger_counts`：各分区触发次数。
- `ca3_compression_ratio`：本次装配压缩率（`0~1`）。
- `ca3_spill_count`：本次 spill 计数。
- `ca3_swap_back_count`：本次 swap-back 计数。

### Run 诊断新增字段（Action Timeline H1.5 聚合）

- `timeline_phases.<phase>.count_total`：phase 终态计数（`succeeded|failed|canceled|skipped`）。
- `timeline_phases.<phase>.failed_total`：phase 失败计数。
- `timeline_phases.<phase>.canceled_total`：phase 取消计数。
- `timeline_phases.<phase>.skipped_total`：phase 跳过计数。
- `timeline_phases.<phase>.latency_ms`：phase 累计耗时（毫秒）。
- `timeline_phases.<phase>.latency_p95_ms`：phase P95 耗时（毫秒）。

说明：
- 聚合维度为“单 run 内按 phase 聚合”。
- 同一 run 的 timeline 重放按 `sequence+phase+status` 去重，不重复累计。

### Run 诊断新增字段（Action Gate H2）

- `gate_checks`：本次 run 触发的 gate 检查次数（高风险规则命中计数）。
- `gate_denied_count`：本次 run 被 gate 拒绝的次数（含 deny/timeout/resolver 错误拒绝）。
- `gate_timeout_count`：本次 run 因确认超时导致拒绝的次数。

Action Timeline reason code（gate 相关）：
- `gate.rule_match`：命中参数规则（H4）。
- `gate.require_confirm`：命中规则且进入确认流程。
- `gate.denied`：被 gate 拒绝（含未配置 resolver 的 fail-fast 拒绝）。
- `gate.timeout`：确认超时后拒绝（timeout-deny）。

Action Gate 规则优先级（H4）：
1. `action_gate.parameter_rules`（参数规则，支持 AND/OR 复合条件）
2. `action_gate.decision_by_tool` / `action_gate.decision_by_keyword`
3. `action_gate.tool_names` / `action_gate.keywords` + 全局 `action_gate.policy`
4. 默认 allow

### Run 诊断新增字段（Action Gate H4）

- `gate_rule_hit_count`：本次 run 命中的参数规则次数。
- `gate_rule_last_id`：本次 run 最近一次命中的参数规则 ID（未命中为空字符串）。

### Run 诊断新增字段（HITL Clarification H3）

- `await_count`：本次 run 进入 `await_user` 的次数。
- `resume_count`：本次 run 成功恢复（`resumed`）的次数。
- `cancel_by_user_count`：本次 run 因超时策略 `cancel_by_user` 取消的次数。

Action Timeline reason code（H3 相关）：
- `hitl.await_user`：进入澄清等待态。
- `hitl.resumed`：收到澄清输入并恢复执行。
- `hitl.canceled_by_user`：澄清等待超时，按策略取消当前 run。

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

### TODO（后续演进）

- H1.5 已完成 phase 级聚合；后续可按需要补充跨 run 维度聚合（窗口化趋势、分位线面板等）并保持库接口优先。

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
- context assembler CA2 的 `agentic` routing mode 当前仍为占位；配置后会返回明确 not-ready 错误。

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
