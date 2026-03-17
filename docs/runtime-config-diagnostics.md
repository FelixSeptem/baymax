# Runtime Config & Diagnostics API

更新时间：2026-03-17

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
  backpressure: block # block|reject|drop_low_priority
  cancel_propagation_timeout: 1500ms # 取消传播收敛超时，必须 > 0
  drop_low_priority:
    priority_by_tool:
      local.search: low
    priority_by_keyword:
      cache: low
      warmup: low
    droppable_priorities: [low] # low|normal|high

diagnostics:
  max_call_records: 200
  max_run_records: 200
  max_reload_errors: 100
  max_skill_records: 200
  timeline_trend:
    enabled: true
    last_n_runs: 100
    time_window: 15m
  ca2_external_trend:
    enabled: true
    window: 15m
    thresholds:
      p95_latency_ms: 1500
      error_rate: 0.10
      hit_rate: 0.20

reload:
  enabled: true
  debounce: 200ms

provider_fallback:
  enabled: false
  providers: [openai, anthropic, gemini] # 有序候选链；enabled=true 时必须非空
  discovery_timeout: 1500ms
  discovery_cache_ttl: 5m

skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords # 当前仅支持该策略
    confidence_threshold: 0.25          # [0,1]
    tie_break: highest_priority         # highest_priority|first_registered
    suppress_low_confidence: true
    keyword_weights:
      database: 1.5
      db: 1.5
      sql: 1.6
      search: 1.2

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
        profile: http_generic # legacy: http_generic；template-pack: ragflow_like|graphrag_like|elasticsearch_like|explicit_only
        endpoint: https://retriever.example.com/search # non-file provider 必填
        method: POST # POST|PUT
        hints:
          enabled: false
          capabilities: [metadata_filter] # 小写；允许 [a-z0-9._/-]
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
    compaction:
      mode: truncate # truncate|semantic
      semantic_timeout: 800ms
      quality:
        threshold: 0.60 # [0,1]
        weights:
          coverage: 0.50
          compression: 0.30
          validity: 0.20
      semantic_template:
        prompt: "Compress... {{input}} ... {{source}} ... {{max_runes}}"
        allowed_placeholders: [input, source, max_runes, model, messages_count]
      embedding:
        enabled: false
        selector: "" # enabled=true 时必填
        provider: openai # openai|gemini|anthropic
        model: text-embedding-3-small
        timeout: 800ms
        similarity_metric: cosine # E3 仅支持 cosine
        rule_weight: 0.7
        embedding_weight: 0.3
        auth:
          api_key: ""
          base_url: ""
        provider_auth:
          openai:
            api_key: ""
            base_url: ""
          gemini:
            api_key: ""
            base_url: ""
          anthropic:
            api_key: ""
            base_url: ""
      reranker:
        enabled: false
        timeout: 500ms
        max_retries: 1
        governance:
          mode: enforce # enforce|dry_run
          profile_version: ""
          rollout_provider_models: [] # provider:model list; empty means match all
        threshold_profiles:
          openai:text-embedding-3-small: 0.62
      evidence:
        keywords: [decision, constraint, todo, risk]
        recent_window: 0

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

Skill trigger scoring 校验语义：
1. `strategy` 当前必须为 `lexical_weighted_keywords`。
2. `confidence_threshold` 必须在 `[0,1]`。
3. `tie_break` 仅支持 `highest_priority|first_registered`。
4. `keyword_weights` 必须非空，且每个权重必须 `> 0`。
5. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效）。

drop_low_priority 校验语义：
1. `concurrency.backpressure=drop_low_priority` 在 `local + mcp + skill` 调度语义上统一生效。
2. `concurrency.drop_low_priority.droppable_priorities` 必须非空，且值仅允许 `low|normal|high`。
3. `priority_by_tool` 与 `priority_by_keyword` 的 value 仅允许 `low|normal|high`。
4. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效）。

timeline_trend 校验语义：
1. `diagnostics.timeline_trend.last_n_runs` 必须 `> 0`。
2. `diagnostics.timeline_trend.time_window` 必须 `> 0`。
3. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

ca2_external_trend 校验语义：
1. `diagnostics.ca2_external_trend.window` 必须 `> 0`。
2. `diagnostics.ca2_external_trend.thresholds.p95_latency_ms` 必须 `> 0`。
3. `diagnostics.ca2_external_trend.thresholds.error_rate` 必须在 `[0,1]`。
4. `diagnostics.ca2_external_trend.thresholds.hit_rate` 必须在 `[0,1]`。
5. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

ca2 stage2 external hint/template 校验语义：
1. `context_assembler.ca2.stage2.external.profile` 仅支持 `http_generic|ragflow_like|graphrag_like|elasticsearch_like|explicit_only`。
2. `context_assembler.ca2.stage2.external.hints.enabled=true` 时，`hints.capabilities` 必须非空。
3. `hints.capabilities[*]` 必须使用小写并满足字符集 `[a-z0-9._/-]`。
4. 非法 hint/template 配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

ca3 compaction 校验语义：
1. `context_assembler.ca3.compaction.mode` 仅允许 `truncate|semantic`。
2. `context_assembler.ca3.compaction.semantic_timeout` 必须 `> 0`。
3. `context_assembler.ca3.compaction.quality.threshold` 必须在 `[0,1]`；`weights.* >= 0` 且总和 `> 0`。
4. `context_assembler.ca3.compaction.semantic_template.prompt` 必须非空，且占位符必须在 `allowed_placeholders` 白名单内。
5. `context_assembler.ca3.compaction.embedding.similarity_metric` 当前必须为 `cosine`。
6. `context_assembler.ca3.compaction.embedding.rule_weight|embedding_weight` 必须在 `[0,1]`，且两者和 `> 0`。
7. `context_assembler.ca3.compaction.embedding.enabled=true` 时必须提供 `embedding.selector`、`embedding.provider`（`openai|gemini|anthropic`）、`embedding.model`、`embedding.timeout>0`。
8. `context_assembler.ca3.compaction.reranker.enabled=true` 时必须满足：
   - `embedding.enabled=true`
   - `reranker.timeout>0`
   - `reranker.threshold_profiles` 非空
   - 且包含当前 `embedding.provider:embedding.model` 对应 key。
9. `context_assembler.ca3.compaction.reranker.governance.mode` 仅允许 `enforce|dry_run`。
10. `context_assembler.ca3.compaction.reranker.governance.rollout_provider_models[*]` 必须满足 `provider:model` 格式。
11. `context_assembler.ca3.compaction.evidence.recent_window` 必须 `>= 0`。
12. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

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
- `Manager.TimelineTrends(query)`：跨 run Action Timeline 趋势聚合（窗口模式：`last_n_runs|time_window`）。
- `Manager.CA2ExternalTrends(query)`：CA2 external retriever provider 维度趋势聚合（窗口模式：`time_window`）。
- `Manager.EffectiveConfigSanitized()`：脱敏后的生效配置快照。
- `Manager.PrecheckStage2External(provider, external)`：CA2 external retriever 预检查（warning 可继续，error 需 fail-fast）。

默认调试路径仍为 library-first；D1 补充了可选离线回放命令：`go run ./cmd/diagnostics-replay -input diagnostics.json`。

## 诊断回放（D1）

- 输入：diagnostics JSON（`timeline_events` 或 `events`）。
- 输出：精简 timeline 视图（`run_id/sequence/phase/status/reason/timestamp`）。
- 目标：离线排障与契约回归，不依赖在线 runtime API。

详细使用说明见：`docs/diagnostics-replay.md`。

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
- `stage2_template_profile`：Stage2 模板解析使用的 template profile（可与 `stage2_profile` 并存，增量兼容）。
- `stage2_template_resolution_source`：模板解析来源（`profile_defaults_only|profile_defaults_then_explicit_overrides|explicit_only`）。
- `stage2_hint_applied`：本次 Stage2 是否成功应用 capability hints。
- `stage2_hint_mismatch_reason`：hint 不匹配原因（例如 `hint.unsupported`；仅观测，不自动动作）。
- `stage2_hit_count`：Stage2 本次命中的 chunk 数量。
- `stage2_source`：Stage2 数据源标识（provider 或响应映射字段）。
- `stage2_reason`：Stage2 执行原因/结果摘要（如 `ok`/`empty`/`timeout`/`fetch_error`）。
- `stage2_reason_code`：Stage2 机器可读原因码（如 `ok`/`timeout`/`http_status`/`upstream_error`）。
- `stage2_error_layer`：Stage2 错误分层（`transport|protocol|semantic`，成功时为空）。
- `recap_status`：tail recap 状态（`disabled|appended|truncated|failed`）。

### CA2 E3 Template-Pack YAML 样例（精简）

```yaml
# graphrag_like
context_assembler:
  ca2:
    stage2:
      provider: http
      external:
        profile: graphrag_like
        endpoint: https://retriever.example.com/graphrag/search
        hints:
          enabled: true
          capabilities: [metadata_filter]
```

```yaml
# ragflow_like
context_assembler:
  ca2:
    stage2:
      provider: rag
      external:
        profile: ragflow_like
        endpoint: https://retriever.example.com/ragflow/query
        hints:
          enabled: true
          capabilities: [metadata_filter, rerank_metadata]
```

```yaml
# elasticsearch_like
context_assembler:
  ca2:
    stage2:
      provider: elasticsearch
      external:
        profile: elasticsearch_like
        endpoint: https://retriever.example.com/es/search
        hints:
          enabled: true
          capabilities: [dsl_query, metadata_filter]
```

### Run 诊断新增字段（Context Assembler CA3）

- `ca3_pressure_zone`：CA3 当前压力分区（`safe|comfort|warning|danger|emergency`）。
- `ca3_pressure_reason`：分区触发来源（`usage_percent_trigger|absolute_token_trigger`）。
- `ca3_pressure_trigger`：本次最终触发分区（双触发冲突时记录被选中的更高压力分区）。
- `ca3_zone_residency_ms`：各分区累计停留时长（毫秒）。
- `ca3_trigger_counts`：各分区触发次数。
- `ca3_compression_ratio`：本次装配压缩率（`0~1`）。
- `ca3_spill_count`：本次 spill 计数。
- `ca3_swap_back_count`：本次 swap-back 计数。
- `ca3_compaction_mode`：本次 CA3 压缩模式（`truncate|semantic`）。
- `ca3_compaction_fallback`：语义压缩失败后是否发生 `truncate` 回退（`best_effort` 下可能为 true）。
- `ca3_compaction_fallback_reason`：语义回退原因（如 `quality_below_threshold`、`semantic_compaction_error`）。
- `ca3_compaction_quality_score`：语义压缩质量分（`0~1`）。
- `ca3_compaction_quality_reason`：质量判定原因（可多值，如 `quality_pass`、`coverage_low`）。
- `ca3_compaction_embedding_provider`：embedding 评分选中的 provider（`openai|gemini|anthropic`）。
- `ca3_compaction_embedding_similarity`：embedding cosine 相似度（归一化到 `0~1`）。
- `ca3_compaction_embedding_contribution`：embedding 分量对最终 quality 分的贡献值。
- `ca3_compaction_embedding_status`：embedding 评分状态（如 `used|fallback_rule_only|disabled`）。
- `ca3_compaction_embedding_fallback_reason`：embedding 回退原因（如 `embedding_score_error|embedding_hook_not_bound`）。
- `ca3_compaction_reranker_used`：是否执行了 reranker 阶段。
- `ca3_compaction_reranker_provider`：reranker 使用的 provider。
- `ca3_compaction_reranker_model`：reranker 使用的 model。
- `ca3_compaction_reranker_threshold_source`：阈值来源（如 `provider_model_profile`）。
- `ca3_compaction_reranker_threshold_hit`：是否命中 reranker 阈值（`score < threshold`）。
- `ca3_compaction_reranker_fallback_reason`：reranker 回退原因（如 `reranker_error`）。
- `ca3_compaction_reranker_profile_version`：治理阈值 profile 版本标签。
- `ca3_compaction_reranker_rollout_hit`：是否命中 `provider:model` 灰度匹配。
- `ca3_compaction_reranker_threshold_drift`：治理阈值与基础阈值的差值绝对值（用于漂移观测）。
- `ca3_compaction_retained_evidence_count`：本次 prune 过程中被证据保留规则保护的消息数量。

语义说明：
- `semantic` 模式通过当前 model-step 选中的 model client 执行压缩。
- quality gate 在 semantic 路径执行（coverage/compression/validity 规则评分）。
- semantic prompt 由 runtime 模板渲染，模板变量受白名单约束。
- embedding adapter 支持 `openai|gemini|anthropic`；当前相似度指标固定为 `cosine`。
- reranker 支持 provider-specific 扩展注册（`assembler.WithSemanticReranker`），未注册时走内置默认实现。
- 阈值治理模式支持 `enforce|dry_run`：`dry_run` 只评估治理阈值，不改变最终 gate 决策。
- 允许使用独立 embedding 凭证（`embedding.auth.*`）并支持 provider 级覆盖（`embedding.provider_auth.*`）。
- 若 stage policy 为 `best_effort`，语义压缩失败会回退 `truncate` 并记录 `ca3_compaction_fallback=true`。
- 若 stage policy 为 `fail_fast`，语义压缩失败会立即终止当前装配流程。

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

### 诊断新增字段（Action Timeline H16 趋势聚合）

`Manager.TimelineTrends(query)` 返回跨 run 趋势记录，最小字段：

- `phase`：阶段维度（`run|context_assembler|model|tool|mcp|skill|hitl`）。
- `status`：状态维度（`pending|running|succeeded|failed|skipped|canceled`）。
- `count_total`：窗口内该 bucket 的终态计数。
- `failed_total`：窗口内失败计数。
- `canceled_total`：窗口内取消计数。
- `skipped_total`：窗口内跳过计数。
- `latency_avg_ms`：窗口内平均耗时（毫秒）。
- `latency_p95_ms`：窗口内 P95 耗时（毫秒）。
- `window_start`：查询窗口起始时间。
- `window_end`：查询窗口结束时间。

窗口模式：
- `last_n_runs`：按最近 N 条 run 记录聚合（默认 `N=100`）。
- `time_window`：按最近时间窗口聚合（默认 `15m`）。

语义约束：
- 趋势聚合默认启用，可通过 `diagnostics.timeline_trend.enabled` 关闭。
- 空窗口返回空集合，不伪造统计。
- 复用 single-writer + idempotency 口径，replay/duplicate 不重复累计。

### 诊断新增字段（CA2 External Retriever E2 趋势聚合）

`Manager.CA2ExternalTrends(query)` 返回 provider 维度趋势记录，最小字段：

- `provider`：Stage2 provider 名称（如 `http|rag|db|elasticsearch|file`）。
- `window_start`：窗口起始时间。
- `window_end`：窗口结束时间。
- `p95_latency_ms`：窗口内 provider 的 Stage2 P95 延迟（毫秒）。
- `error_rate`：窗口内错误占比（按 `stage2_reason_code/stage2_error_layer` 判定）。
- `hit_rate`：窗口内 `stage2_hit_count > 0` 的占比。

扩展字段：
- `threshold_hits`：命中的静态阈值列表（`p95_latency_ms|error_rate|hit_rate`）。
- `error_layer_distribution`：错误层分布（基线 `transport|protocol|semantic`，允许新增枚举扩展）。

语义约束：
- 阈值命中仅输出观测信号，不触发自动降级/切换动作。
- 保持 `fail_fast/best_effort` 既有行为不变。
- Run/Stream 在等价负载下保持趋势统计语义一致。

### Run 诊断新增字段（Action Gate H2）

- `gate_checks`：本次 run 触发的 gate 检查次数（高风险规则命中计数）。
- `gate_denied_count`：本次 run 被 gate 拒绝的次数（含 deny/timeout/resolver 错误拒绝）。
- `gate_timeout_count`：本次 run 因确认超时导致拒绝的次数。

Action Timeline reason code（gate 相关）：
- `gate.rule_match`：命中参数规则（H4）。
- `gate.require_confirm`：命中规则且进入确认流程。
- `gate.denied`：被 gate 拒绝（含未配置 resolver 的 fail-fast 拒绝）。
- `gate.timeout`：确认超时后拒绝（timeout-deny）。
- `backpressure.block`：命中 block 背压排队路径（用于并发基线可观测）。
- `backpressure.drop_low_priority`：命中 drop_low_priority 背压丢弃路径（`local|mcp|skill` 统一语义）。
- `cancel.propagated`：父上下文取消已传播到当前执行分支（Run/Stream 对齐）。

Action Gate 规则优先级（H4）：
1. `action_gate.parameter_rules`（参数规则，支持 AND/OR 复合条件）
2. `action_gate.decision_by_tool` / `action_gate.decision_by_keyword`
3. `action_gate.tool_names` / `action_gate.keywords` + 全局 `action_gate.policy`
4. 默认 allow

### Run 诊断新增字段（Action Gate H4）

- `gate_rule_hit_count`：本次 run 命中的参数规则次数。
- `gate_rule_last_id`：本次 run 最近一次命中的参数规则 ID（未命中为空字符串）。

### Run 诊断新增字段（并发基线 R5）

- `cancel_propagated_count`：本次 run 内取消传播生效次数（非负整数）。
- `backpressure_drop_count`：本次 run 背压丢弃次数（`block` 策略下应为 `0`，`drop_low_priority` 可大于 `0`）。
- `backpressure_drop_count_by_phase`：本次 run 背压丢弃分桶计数（`local|mcp|skill`）。
- `inflight_peak`：本次 run 观测到的在途并发峰值（run 级）。

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

## 安全基线（S1）

- 质量门禁脚本（Linux/PowerShell）与 CI 默认执行 `govulncheck` 且使用 strict 语义。
- 可通过环境变量降级为 warn：`BAYMAX_SECURITY_SCAN_MODE=warn`。
- 可通过 `BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED=false` 临时关闭扫描（不建议在 CI 中关闭）。
- 统一脱敏管线覆盖：
  - `runtime/diagnostics`（配置快照与诊断 payload）
  - `observability/event`（JSON logger 与 runtime recorder）
  - `context/assembler`（stage2 payload 与 tail recap）
- 脱敏策略默认按 key 关键词匹配，支持扩展 matcher 接口（后续阶段可接入更复杂策略）。

## 安全治理（S2）

新增 S2 策略入口，统一覆盖 `namespace+tool` 权限、进程级限流、模型输入/输出过滤：

```yaml
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+shell: deny
    rate_limit:
      enabled: true
      scope: process
      window: 1m
      limit: 120
      by_tool_limit:
        local+search: 30
      exceed_action: deny
  model_io_filtering:
    enabled: true
    require_registered_filter: false
    input:
      enabled: true
      block_action: deny
    output:
      enabled: true
      block_action: deny
```

校验与热更新语义：

1. `namespace+tool` 选择器格式非法（例如 `local.echo`）会在启动与热更新阶段 fail-fast。
2. `mode/scope/block_action` 非法枚举值会 fail-fast。
3. 热更新遵循原子切换：有效配置立即生效；无效更新回滚到上一有效快照。

新增 run 诊断字段（增量兼容）：

- `policy_kind`: `permission|rate_limit|io_filter`
- `namespace_tool`: 匹配到的 `namespace+tool`
- `filter_stage`: `input|output`
- `decision`: `allow|match|deny`
- `reason_code`: 归一化原因码（如 `security.permission_denied`、`security.rate_limit_exceeded`、`security.io_filter_match`、`security.io_filter_denied`）

独立安全门禁（required-check 候选）：

- Linux/macOS: `bash scripts/check-security-policy-contract.sh`
- Windows: `pwsh -File scripts/check-security-policy-contract.ps1`
- CI Job: `security-policy-gate`（仅 PR 触发）

## 安全事件治理（S3）

在 S2 阻断基础上，新增统一安全事件 taxonomy 与 deny-only callback 告警契约：

```yaml
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only # 当前仅支持 deny_only
      sink: callback            # 当前仅支持 callback
      callback:
        require_registered: false
    severity:
      default: high             # low|medium|high
      by_policy_kind:
        permission: high
        rate_limit: high
        io_filter: high
      by_reason_code:
        security.io_filter_match: medium
```

运行时语义：

1. 统一事件字段：`policy_kind|namespace_tool|filter_stage|decision|reason_code|severity`。
2. 仅 `decision=deny` 触发 callback；`allow|match` 只保留观测，不触发告警。
3. callback 失败不会改变原有安全决策（仍保持 deny），仅追加告警投递失败诊断。
4. Run/Stream 在等价输入与配置下需保持 `policy_kind|decision|reason_code|severity|alert_dispatch_status` 语义等价。

新增 run 诊断字段（增量兼容）：

- `severity`: 归一化严重级别（`low|medium|high`）
- `alert_dispatch_status`: 告警投递状态（`disabled|not_triggered|skipped|succeeded|failed`）
- `alert_dispatch_failure_reason`: 告警投递失败原因码（如 `alert.callback_missing|alert.callback_error`）

独立安全事件门禁（required-check 候选）：

- Linux/macOS: `bash scripts/check-security-event-contract.sh`
- Windows: `pwsh -File scripts/check-security-event-contract.ps1`
- CI Job: `security-event-gate`（仅 PR 触发）

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
