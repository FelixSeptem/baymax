# Runtime Config & Diagnostics API

更新时间：2026-03-20

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
  - `composer.collab.enabled` -> `BAYMAX_COMPOSER_COLLAB_ENABLED`
  - `composer.collab.default_aggregation` -> `BAYMAX_COMPOSER_COLLAB_DEFAULT_AGGREGATION`
  - `composer.collab.failure_policy` -> `BAYMAX_COMPOSER_COLLAB_FAILURE_POLICY`
  - `composer.collab.retry.enabled` -> `BAYMAX_COMPOSER_COLLAB_RETRY_ENABLED`
  - `teams.remote.enabled` -> `BAYMAX_TEAMS_REMOTE_ENABLED`
  - `teams.remote.require_peer_id` -> `BAYMAX_TEAMS_REMOTE_REQUIRE_PEER_ID`
  - `workflow.graph_composability.enabled` -> `BAYMAX_WORKFLOW_GRAPH_COMPOSABILITY_ENABLED`
  - `workflow.remote.enabled` -> `BAYMAX_WORKFLOW_REMOTE_ENABLED`
  - `workflow.remote.default_retry_max_attempts` -> `BAYMAX_WORKFLOW_REMOTE_DEFAULT_RETRY_MAX_ATTEMPTS`
  - `a2a.async_reporting.enabled` -> `BAYMAX_A2A_ASYNC_REPORTING_ENABLED`
  - `a2a.async_reporting.sink` -> `BAYMAX_A2A_ASYNC_REPORTING_SINK`
  - `a2a.async_reporting.retry.max_attempts` -> `BAYMAX_A2A_ASYNC_REPORTING_RETRY_MAX_ATTEMPTS`
  - `a2a.async_reporting.retry.backoff_initial` -> `BAYMAX_A2A_ASYNC_REPORTING_RETRY_BACKOFF_INITIAL`
  - `a2a.async_reporting.retry.backoff_max` -> `BAYMAX_A2A_ASYNC_REPORTING_RETRY_BACKOFF_MAX`
  - `mailbox.enabled` -> `BAYMAX_MAILBOX_ENABLED`
  - `mailbox.backend` -> `BAYMAX_MAILBOX_BACKEND`
  - `mailbox.path` -> `BAYMAX_MAILBOX_PATH`
  - `mailbox.retry.max_attempts` -> `BAYMAX_MAILBOX_RETRY_MAX_ATTEMPTS`
  - `mailbox.retry.backoff_initial` -> `BAYMAX_MAILBOX_RETRY_BACKOFF_INITIAL`
  - `mailbox.retry.backoff_max` -> `BAYMAX_MAILBOX_RETRY_BACKOFF_MAX`
  - `mailbox.retry.jitter_ratio` -> `BAYMAX_MAILBOX_RETRY_JITTER_RATIO`
  - `mailbox.ttl` -> `BAYMAX_MAILBOX_TTL`
  - `mailbox.dlq.enabled` -> `BAYMAX_MAILBOX_DLQ_ENABLED`
  - `mailbox.query.page_size_default` -> `BAYMAX_MAILBOX_QUERY_PAGE_SIZE_DEFAULT`
  - `mailbox.query.page_size_max` -> `BAYMAX_MAILBOX_QUERY_PAGE_SIZE_MAX`
  - `scheduler.enabled` -> `BAYMAX_SCHEDULER_ENABLED`
  - `scheduler.backend` -> `BAYMAX_SCHEDULER_BACKEND`
  - `scheduler.lease_timeout` -> `BAYMAX_SCHEDULER_LEASE_TIMEOUT`
  - `scheduler.heartbeat_interval` -> `BAYMAX_SCHEDULER_HEARTBEAT_INTERVAL`
  - `scheduler.qos.mode` -> `BAYMAX_SCHEDULER_QOS_MODE`
  - `scheduler.qos.fairness.max_consecutive_claims_per_priority` -> `BAYMAX_SCHEDULER_QOS_FAIRNESS_MAX_CONSECUTIVE_CLAIMS_PER_PRIORITY`
  - `scheduler.async_await.report_timeout` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_REPORT_TIMEOUT`
  - `scheduler.async_await.late_report_policy` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_LATE_REPORT_POLICY`
  - `scheduler.async_await.timeout_terminal` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_TIMEOUT_TERMINAL`
  - `scheduler.async_await.reconcile.enabled` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_RECONCILE_ENABLED`
  - `scheduler.async_await.reconcile.interval` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_RECONCILE_INTERVAL`
  - `scheduler.async_await.reconcile.batch_size` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_RECONCILE_BATCH_SIZE`
  - `scheduler.async_await.reconcile.jitter_ratio` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_RECONCILE_JITTER_RATIO`
  - `scheduler.async_await.reconcile.not_found_policy` -> `BAYMAX_SCHEDULER_ASYNC_AWAIT_RECONCILE_NOT_FOUND_POLICY`
  - `scheduler.dlq.enabled` -> `BAYMAX_SCHEDULER_DLQ_ENABLED`
  - `scheduler.retry.backoff.enabled` -> `BAYMAX_SCHEDULER_RETRY_BACKOFF_ENABLED`
  - `scheduler.retry.backoff.initial` -> `BAYMAX_SCHEDULER_RETRY_BACKOFF_INITIAL`
  - `scheduler.retry.backoff.max` -> `BAYMAX_SCHEDULER_RETRY_BACKOFF_MAX`
  - `scheduler.retry.backoff.multiplier` -> `BAYMAX_SCHEDULER_RETRY_BACKOFF_MULTIPLIER`
  - `scheduler.retry.backoff.jitter_ratio` -> `BAYMAX_SCHEDULER_RETRY_BACKOFF_JITTER_RATIO`
  - `subagent.max_depth` -> `BAYMAX_SUBAGENT_MAX_DEPTH`
  - `subagent.max_active_children` -> `BAYMAX_SUBAGENT_MAX_ACTIVE_CHILDREN`
  - `subagent.child_timeout_budget` -> `BAYMAX_SUBAGENT_CHILD_TIMEOUT_BUDGET`
  - `recovery.enabled` -> `BAYMAX_RECOVERY_ENABLED`
  - `recovery.backend` -> `BAYMAX_RECOVERY_BACKEND`
  - `recovery.path` -> `BAYMAX_RECOVERY_PATH`
  - `recovery.conflict_policy` -> `BAYMAX_RECOVERY_CONFLICT_POLICY`
  - `recovery.resume_boundary` -> `BAYMAX_RECOVERY_RESUME_BOUNDARY`
  - `recovery.inflight_policy` -> `BAYMAX_RECOVERY_INFLIGHT_POLICY`
  - `recovery.timeout_reentry_policy` -> `BAYMAX_RECOVERY_TIMEOUT_REENTRY_POLICY`
  - `recovery.timeout_reentry_max_per_task` -> `BAYMAX_RECOVERY_TIMEOUT_REENTRY_MAX_PER_TASK`

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

composer:
  collab:
    enabled: false                   # A16 feature flag，默认关闭
    default_aggregation: all_settled # all_settled|first_success
    failure_policy: fail_fast        # fail_fast|best_effort
    retry:
      enabled: false                 # A16：协作原语层重试固定关闭

teams:
  enabled: false
  default_strategy: serial # serial|parallel|vote
  task_timeout: 3s          # 必须 > 0
  parallel:
    max_workers: 4          # 并行策略 worker 上限，必须 > 0
  vote:
    tie_break: highest_priority # highest_priority|first_task_id
  remote:
    enabled: false             # 启用 remote task 时，要求 teams.enabled=true
    require_peer_id: true      # remote task 是否强制 peer_id

workflow:
  enabled: false
  planner_validation_mode: strict # strict|warn
  default_step_timeout: 3s        # 必须 > 0
  checkpoint_backend: memory      # memory|file
  checkpoint_path: /tmp/baymax/workflow-checkpoints # backend=file 时必填
  graph_composability:
    enabled: false                # A15，默认关闭；开启后启用 subgraphs/use_subgraph/condition_templates 编译展开
  remote:
    enabled: false                # 启用 a2a remote step 时，要求 workflow.enabled=true
    require_peer_id: true         # a2a step 是否强制 peer_id
    default_retry_max_attempts: 2 # 必须 >= 0

a2a:
  enabled: false
  client_timeout: 1500ms          # 必须 > 0
  delivery:
    mode: callback                # callback|sse
    fallback_mode: callback       # callback|sse
    callback_retry:
      max_attempts: 3             # 必须 > 0
      backoff: 100ms              # 必须 >= 0
    sse_reconnect:
      max_attempts: 3             # 必须 > 0
      backoff: 100ms              # 必须 >= 0
  card:
    version_policy:
      mode: strict_major          # 当前仅支持 strict_major
      min_supported_minor: 0      # 必须 >= 0
  capability_discovery:
    enabled: true
    require_all: true
    max_candidates: 16            # 必须 > 0
  async_reporting:
    enabled: false
    sink: callback                # callback|channel
    retry:
      max_attempts: 3             # 必须 > 0
      backoff_initial: 50ms       # 必须 >= 0
      backoff_max: 500ms          # 必须 >= backoff_initial

mailbox:
  enabled: false
  backend: memory                 # memory|file
  path: /tmp/baymax/mailbox-state.json # backend=file 时必填
  retry:
    max_attempts: 3               # 必须 > 0
    backoff_initial: 50ms         # 必须 >= 0
    backoff_max: 500ms            # 必须 >= backoff_initial
    jitter_ratio: 0.2             # 必须在 [0,1]
  ttl: 15m                        # 必须 >= 0（0 表示不启用 TTL）
  dlq:
    enabled: false
  query:
    page_size_default: 50         # 必须 > 0 且 <= page_size_max
    page_size_max: 200            # 必须 > 0 且 <= 200

scheduler:
  enabled: false
  backend: memory                 # memory|file
  path: /tmp/baymax/scheduler-state.json # backend=file 时建议显式配置
  lease_timeout: 2s               # 必须 > 0
  heartbeat_interval: 500ms       # 必须 > 0 且 < lease_timeout
  queue_limit: 1024               # 必须 > 0
  retry_max_attempts: 3           # 必须 > 0
  qos:
    mode: fifo                    # fifo|priority（默认 fifo）
    fairness:
      max_consecutive_claims_per_priority: 3 # 必须 > 0
  async_await:
    report_timeout: 15m           # 必须 > 0
    late_report_policy: drop_and_record # 当前仅支持 drop_and_record
    timeout_terminal: failed      # failed|dead_letter（dead_letter 需配合 scheduler.dlq.enabled=true）
    reconcile:
      enabled: false
      interval: 5s                # 必须 > 0
      batch_size: 64              # 必须 > 0
      jitter_ratio: 0.2           # 必须在 [0,1]
      not_found_policy: keep_until_timeout # 当前仅支持 keep_until_timeout
  dlq:
    enabled: false                # 默认 false
  retry:
    backoff:
      enabled: false              # 默认 false（兼容 A6/A7 立即可 claim 语义）
      initial: 100ms              # enabled=true 时必须 > 0
      max: 2s                     # enabled=true 时必须 > 0 且 >= initial
      multiplier: 2.0             # enabled=true 时必须 > 1
      jitter_ratio: 0.2           # enabled=true 时必须在 [0,1]

recovery:
  enabled: false
  backend: memory                 # memory|file
  path: /tmp/baymax/recovery      # backend=file 时必填
  conflict_policy: fail_fast      # 当前仅支持 fail_fast
  resume_boundary: next_attempt_only           # 当前仅支持 next_attempt_only
  inflight_policy: no_rewind                   # 当前仅支持 no_rewind
  timeout_reentry_policy: single_reentry_then_fail # 当前仅支持 single_reentry_then_fail
  timeout_reentry_max_per_task: 1              # 当前固定为 1

subagent:
  max_depth: 4                    # 必须 > 0
  max_active_children: 8          # 必须 > 0
  child_timeout_budget: 5s        # 必须 > 0

skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords # lexical_weighted_keywords|lexical_plus_embedding
    confidence_threshold: 0.25          # [0,1]
    tie_break: highest_priority         # highest_priority|first_registered
    suppress_low_confidence: true
    max_semantic_candidates: 5          # semantic 候选预算上限，必须 > 0
    lexical:
      tokenizer_mode: mixed_cjk_en      # D3: 当前仅支持 mixed_cjk_en
    budget:
      mode: adaptive                    # D4: fixed|adaptive，默认 adaptive
      adaptive:
        min_k: 1
        max_k: 5                        # 必须 <= max_semantic_candidates
        min_score_margin: 0.08          # [0,1]
    keyword_weights:
      database: 1.5
      db: 1.5
      sql: 1.6
      search: 1.2
    embedding:
      enabled: false
      provider: openai # openai|gemini|anthropic
      model: text-embedding-3-small
      timeout: 300ms
      similarity_metric: cosine # D2 仅支持 cosine
      lexical_weight: 0.7
      embedding_weight: 0.3

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
    routing_mode: rules # rules|agentic
    agentic:
      decision_timeout: 80ms
      failure_policy: best_effort_rules # 当前仅支持 best_effort_rules
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
1. `strategy` 仅支持 `lexical_weighted_keywords|lexical_plus_embedding`。
2. `confidence_threshold` 必须在 `[0,1]`。
3. `tie_break` 仅支持 `highest_priority|first_registered`。
4. `lexical.tokenizer_mode` 当前仅支持 `mixed_cjk_en`。
5. `max_semantic_candidates` 必须 `> 0`。
6. `budget.mode` 仅支持 `fixed|adaptive`。
7. `budget.adaptive.min_k` 必须 `> 0`。
8. `budget.adaptive.max_k` 必须 `>= min_k` 且 `<= max_semantic_candidates`。
9. `budget.adaptive.min_score_margin` 必须在 `[0,1]`。
10. `keyword_weights` 必须非空，且每个权重必须 `> 0`。
11. `embedding.timeout` 必须 `> 0`。
12. `embedding.similarity_metric` 当前必须为 `cosine`。
13. `embedding.lexical_weight|embedding_weight` 必须在 `[0,1]` 且和 `> 0`。
14. `strategy=lexical_plus_embedding` 时，`embedding.enabled` 必须为 `true`。
15. `embedding.enabled=true` 时，`embedding.provider` 必须是 `openai|gemini|anthropic` 且 `embedding.model` 非空。
16. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

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

teams baseline 校验语义：
1. `teams.default_strategy` 仅支持 `serial|parallel|vote`。
2. `teams.task_timeout` 必须 `> 0`。
3. `teams.parallel.max_workers` 必须 `> 0`。
4. `teams.vote.tie_break` 仅支持 `highest_priority|first_task_id`。
5. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

workflow baseline 校验语义：
1. `workflow.planner_validation_mode` 仅支持 `strict|warn`。
2. `workflow.default_step_timeout` 必须 `> 0`。
3. `workflow.checkpoint_backend` 仅支持 `memory|file`。
4. `workflow.checkpoint_backend=file` 时，`workflow.checkpoint_path` 必须非空。
5. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

a2a baseline 校验语义：
1. `a2a.client_timeout` 必须 `> 0`。
2. `a2a.delivery.mode` / `a2a.delivery.fallback_mode` 仅支持 `callback|sse`。
3. `a2a.delivery.callback_retry.max_attempts` 必须 `> 0`。
4. `a2a.delivery.callback_retry.backoff` 必须 `>= 0`。
5. `a2a.delivery.sse_reconnect.max_attempts` 必须 `> 0`。
6. `a2a.delivery.sse_reconnect.backoff` 必须 `>= 0`。
7. `a2a.card.version_policy.mode` 当前仅支持 `strict_major`。
8. `a2a.card.version_policy.min_supported_minor` 必须 `>= 0`。
9. `a2a.capability_discovery.max_candidates` 必须 `> 0`。
10. `a2a.async_reporting.sink` 仅支持 `callback|channel`。
11. `a2a.async_reporting.retry.max_attempts` 必须 `> 0`。
12. `a2a.async_reporting.retry.backoff_initial` 必须 `>= 0`。
13. `a2a.async_reporting.retry.backoff_max` 必须 `>= a2a.async_reporting.retry.backoff_initial`。
14. A2A 配置键必须保持在 `a2a.*` 域内，避免与 `teams.*`/`workflow.*` 命名重叠。
15. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

mailbox baseline 校验语义：
1. `mailbox.backend` 仅支持 `memory|file`。
2. `mailbox.backend=file` 时，`mailbox.path` 必须非空。
3. `mailbox.retry.max_attempts` 必须 `> 0`。
4. `mailbox.retry.backoff_initial` 必须 `>= 0`。
5. `mailbox.retry.backoff_max` 必须 `>= mailbox.retry.backoff_initial`。
6. `mailbox.retry.jitter_ratio` 必须在 `[0,1]`。
7. `mailbox.ttl` 必须 `>= 0`。
8. `mailbox.query.page_size_default` 必须 `> 0` 且 `<= page_size_max`。
9. `mailbox.query.page_size_max` 必须 `> 0` 且 `<= 200`。
10. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

mailbox 诊断查询入口（A30）：
1. `runtime/config.Manager.QueryMailbox(query)`：支持 `message_id/idempotency_key/correlation_id/kind/state/run_id/task_id/workflow_id/team_id/time_range` 过滤、默认 `page_size=50`、上限 `200`、默认 `time desc`、opaque cursor。
2. `runtime/config.Manager.MailboxAggregates(filter)`：返回聚合计数（`by_kind/by_state/retry_total/dead_letter_total/expired_total/reason_code_totals`），用于与 run/task 视图组合排障。
3. mailbox 诊断记录保留关联键：`run_id/task_id/workflow_id/team_id`。

a2a 同步调用契约（A11）：
1. orchestration 统一复用 `orchestration/invoke` 的 `submit + wait + normalize` 调用路径。
2. `poll_interval` 缺省使用兼容默认值 `20ms`，调用方可按路径覆盖。
3. 同步调用以调用方 `context` 为单一权威，取消/超时优先于轮询等待。
4. 失败输出统一归一为 `transport|protocol|semantic` 错误层与 `retryable` 提示。
5. A11 不新增破坏性配置键；保持既有 `a2a.*`、`teams.*`、`workflow.*`、`scheduler.*` 配置兼容。

scheduler/subagent baseline 校验语义：
1. `scheduler.backend` 仅支持 `memory|file`。
2. `scheduler.backend=file` 时 `scheduler.path` 必须非空。
3. `scheduler.lease_timeout` 必须 `> 0`。
4. `scheduler.heartbeat_interval` 必须 `> 0` 且 `< scheduler.lease_timeout`。
5. `scheduler.queue_limit` 与 `scheduler.retry_max_attempts` 必须 `> 0`。
6. `scheduler.qos.mode` 仅支持 `fifo|priority`。
7. `scheduler.qos.fairness.max_consecutive_claims_per_priority` 必须 `> 0`。
8. `scheduler.dlq.enabled` 为布尔值开关，默认 `false`。
9. `scheduler.retry.backoff.enabled=false` 时不强制校验其余 backoff 参数范围。
10. `scheduler.retry.backoff.enabled=true` 时：`initial>0`、`max>=initial`、`multiplier>1`、`jitter_ratio` 在 `[0,1]`。
11. `scheduler.async_await.report_timeout` 必须 `> 0`。
12. `scheduler.async_await.late_report_policy` 当前仅支持 `drop_and_record`。
13. `scheduler.async_await.timeout_terminal` 仅支持 `failed|dead_letter`。
14. `scheduler.async_await.reconcile.interval` 必须 `> 0`。
15. `scheduler.async_await.reconcile.batch_size` 必须 `> 0`。
16. `scheduler.async_await.reconcile.jitter_ratio` 必须在 `[0,1]`。
17. `scheduler.async_await.reconcile.not_found_policy` 当前仅支持 `keep_until_timeout`。
18. `scheduler.task.not_before` 为可选字段；空值表示立即可领取，未来时间表示延后可领取，过去时间按立即可领取处理。
19. claim 可领取条件为 delayed gate 与 retry gate 的组合：`not_before<=now` 且（若存在）`next_eligible_at<=now`。
20. `subagent.max_depth`、`subagent.max_active_children`、`subagent.child_timeout_budget` 必须 `> 0`。
21. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

recovery boundary（A17）校验语义：
1. `recovery.conflict_policy` 当前仅支持 `fail_fast`。
2. `recovery.resume_boundary` 当前仅支持 `next_attempt_only`。
3. `recovery.inflight_policy` 当前仅支持 `no_rewind`。
4. `recovery.timeout_reentry_policy` 当前仅支持 `single_reentry_then_fail`。
5. `recovery.timeout_reentry_max_per_task` 当前固定为 `1`（用于单次 timeout 重入预算）。
6. `recovery.enabled=false` 时 recovery-boundary 治理不激活，行为保持基线路径不变。
7. 非法配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

ca2 agentic routing 校验语义：
1. `context_assembler.ca2.agentic.decision_timeout` 必须 `> 0`。
2. `context_assembler.ca2.agentic.failure_policy` 当前仅允许 `best_effort_rules`。
3. 非法 agentic 配置在启动与热更新阶段均 fail-fast（拒绝生效并回滚旧快照）。

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

`runtime/config.Manager` 对外提供诊断读取与观测辅助接口（只读）：

- `RecentCalls(n)`
- `RecentRuns(n)`
- `RecentReloads(n)`
- `RecentSkills(n)`
- `RecentMailbox(n)`
- `QueryRuns(query)`
- `QueryMailbox(query)`
- `MailboxAggregates(filter)`
- `TimelineTrends(query)`
- `CA2ExternalTrends(query)`
- `EffectiveConfigSanitized()`
- `PrecheckStage2External(provider, external)`

`orchestration/scheduler.Scheduler` 额外提供任务看板只读查询：

- `QueryTasks(ctx, query)`

### Unified Query API（Run）

过滤字段（多条件按 `AND` 组合）：
- `run_id`
- `team_id`
- `workflow_id`
- `task_id`
- `status`（当前仅支持 `success|failed`）
- `time_range`

分页/排序语义：
- 默认 `page_size=50`
- 最大 `page_size<=200`（越界 fail-fast）
- 默认排序 `time desc`（`sort.field` 当前仅支持 `time`）
- 游标为 `opaque cursor`，且与查询边界绑定

错误与空集语义：
- 非法参数（非法状态、非法排序、非法游标、非法时间区间）均 fail-fast。
- 合法但无匹配时返回空集合，不返回错误。
- 合法但无匹配时也可表述为 `empty result set`（与“返回空集合”语义等价）。

### Task Board Query API（Scheduler）

过滤字段（多条件按 `AND` 组合）：
- `task_id`
- `run_id`
- `workflow_id`
- `team_id`
- `state`（`queued|running|awaiting_report|succeeded|failed|dead_letter`）
- `priority`
- `agent_id`
- `peer_id`
- `parent_run_id`
- `time_range`

分页/排序语义：
- 默认 `page_size=50`
- 最大 `page_size<=200`（越界 fail-fast）
- 默认排序 `updated_at desc`
- 支持排序字段：`updated_at|created_at`
- 游标为 `opaque cursor`，且与查询边界绑定

范围约束：
- 该接口为 scheduler 快照读路径，只读，不改变 enqueue/claim/heartbeat/requeue/commit 语义。
- async-await additive 观测字段：
  - `resolution_source`（`callback|reconcile_poll|timeout`）
  - `remote_task_id`
  - `terminal_conflict_recorded`（可空）

### Mailbox Query API

过滤字段（多条件按 `AND` 组合）：
- `message_id`
- `idempotency_key`
- `correlation_id`
- `kind`（`command|event|result`）
- `state`（`queued|in_flight|acked|nacked|dead_letter|expired`）
- `run_id`
- `task_id`
- `workflow_id`
- `team_id`
- `time_range`

分页/排序语义：
- 默认 `page_size=50`
- 最大 `page_size<=200`（越界 fail-fast）
- 默认排序 `time desc`（`sort.field` 当前仅支持 `time`）
- 游标为 `opaque cursor`，且与查询边界绑定

`MailboxAggregates(filter)` 当前返回：
- `total_records`
- `total_messages`
- `by_kind`
- `by_state`
- `retry_total`
- `dead_letter_total`
- `expired_total`
- `reason_code_totals`

### RunRecord 字段分组（当前实现）

`runtime/diagnostics.RunRecord` 采用“按能力域分组”的 additive 模型（见 `runtime/diagnostics/store.go`）：

- 基础运行摘要：`run_id/status/iterations/tool_calls/latency_ms/error_class`
- Provider 与降级：`model_provider/fallback_* / required_capabilities`
- Context Assembler：`prefix_hash`、`assemble_*`、`stage2_*`、`ca3_*`、`recap_status`
- 编排聚合：`team_*`、`workflow_*`、`a2a_*`、`scheduler_*`、`subagent_*`、`collab_*`
  - A31 additive 字段：`async_await_total`、`async_timeout_total`、`async_late_report_total`、`async_report_dedup_total`
  - A32 additive 字段：`async_reconcile_poll_total`、`async_reconcile_terminal_by_poll_total`、`async_reconcile_error_total`、`async_terminal_conflict_total`
- 恢复与治理：`recovery_*`、`gate_*`、`await_count/resume_count/cancel_by_user_count`
- 并发与背压：`cancel_propagated_count`、`backpressure_drop_count*`、`inflight_peak`
- Timeline 聚合：`timeline_phases.<phase>.*`

兼容约束保持：`additive + nullable + default`
- 新增字段不得改变既有字段语义。
- 缺失字段按约定默认值解释（`0`/`false`/空字符串）。
- 未识别新增字段应被安全忽略。

Compatibility Window (A12/A13)
- 兼容窗口规则：`additive + nullable + default`
- missing additive fields resolve to documented default values
- unknown future additive fields are safely ignored
- pre-existing field semantics remain unchanged

Composed summary additive fields（contract markers）：
- `composer_managed`
- `scheduler_backend_fallback`
- `scheduler_backend_fallback_reason`
- `collab_handoff_total`
- `collab_delegation_total`
- `collab_aggregation_total`
- `collab_aggregation_strategy`
- `collab_fail_fast_total`
- `team_remote_task_total`
- `team_remote_task_failed`
- `workflow_remote_step_total`
- `workflow_remote_step_failed`
- `scheduler_backend`
- `scheduler_queue_total`
- `scheduler_claim_total`
- `scheduler_reclaim_total`
- `scheduler_qos_mode`
- `scheduler_priority_claim_total`
- `scheduler_fairness_yield_total`
- `scheduler_retry_backoff_total`
- `scheduler_dead_letter_total`
- `scheduler_delayed_task_total`
- `scheduler_delayed_claim_total`
- `scheduler_delayed_wait_ms_p95`
- `async_reconcile_poll_total`
- `async_reconcile_terminal_by_poll_total`
- `async_reconcile_error_total`
- `async_terminal_conflict_total`
- `subagent_child_total`
- `subagent_child_failed`
- `subagent_budget_reject_total`
- `recovery_enabled`
- `recovery_resume_boundary`
- `recovery_inflight_policy`
- `recovery_recovered`
- `recovery_replay_total`
- `recovery_timeout_reentry_total`
- `recovery_timeout_reentry_exhausted_total`
- `recovery_conflict`
- `recovery_conflict_code`
- `recovery_fallback_used`
- `recovery_fallback_reason`

## 诊断回放（D1）

离线回放命令：

```bash
go run ./cmd/diagnostics-replay -input diagnostics.json
```

语义：
- 输入：diagnostics JSON（`timeline_events` 或 `events`）
- 输出：精简 timeline 视图（`run_id/sequence/phase/status/reason/timestamp`）
- 目标：离线排障与契约回归，不依赖在线 runtime API

详细使用说明见：`docs/diagnostics-replay.md`。

## CA3 Token Count 职责分工

- `context/assembler` 负责“何时计数”的策略决策（如 `sdk_preferred`、`small_delta_tokens`、`sdk_refresh_interval`）。
- `model/*` 负责“如何计数”的 provider 具体实现与 SDK 对接。
- 小增量优先预估；SDK 计数失败回退预估，不阻断主流程。

## Action Timeline 事件（默认启用）

- 事件类型：`action.timeline`
- 产出路径：`core/runner` 发射，`observability/event` 统一分发与记录
- phase 枚举：`run|context_assembler|model|tool|mcp|skill|hitl`
- status 枚举：`pending|running|succeeded|failed|skipped|canceled`
- payload 最小字段：`phase/status/sequence`，可选 `reason`

reason code 命名保持按域稳定扩展：
- gate / 背压：如 `gate.denied`、`gate.timeout`、`backpressure.block`
- teams / workflow：如 `team.dispatch`、`workflow.schedule`
- a2a / scheduler / recovery：如 `a2a.submit`、`scheduler.claim`、`scheduler.awaiting_report`、`scheduler.async_reconcile`、`scheduler.async_timeout`、`scheduler.async_late_report`、`recovery.restore`
- hitl：如 `hitl.await_user`、`hitl.resumed`

Composed reason contract markers（必须保持字面稳定）：
- teams: `team.dispatch_remote`、`team.collect_remote`、`team.handoff`、`team.delegation`、`team.aggregation`
- workflow: `workflow.dispatch_a2a`、`workflow.handoff`、`workflow.delegation`、`workflow.aggregation`
- a2a async: `a2a.async_submit`、`a2a.async_report_deliver`、`a2a.async_report_retry`、`a2a.async_report_dedup`、`a2a.async_report_drop`
- scheduler/subagent/recovery: `scheduler.enqueue`、`scheduler.delayed_enqueue`、`scheduler.delayed_wait`、`scheduler.delayed_ready`、`scheduler.claim`、`scheduler.heartbeat`、`scheduler.lease_expired`、`scheduler.requeue`、`scheduler.qos_claim`、`scheduler.fairness_yield`、`scheduler.retry_backoff`、`scheduler.dead_letter`、`subagent.spawn`、`subagent.join`、`subagent.budget_reject`、`recovery.restore`、`recovery.replay`、`recovery.conflict`

兼容约束：
- `action.timeline` 为增量事件，不替换既有 `run.* / model.* / tool.* / skill.*` 事件。
- timeline 聚合沿用 single-writer + idempotency 口径，replay/duplicate 不重复累计。

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
- 脱敏策略默认按 key 关键词段匹配（非任意子串），支持扩展 matcher 接口（后续阶段可接入更复杂策略）。

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

## 安全事件与投递治理（S3/S4）

在 S2 阻断基础上，S3 提供统一安全事件 taxonomy 与 deny-only callback 契约，S4 补充 callback 投递可靠性治理（async 队列、drop_old、重试、熔断）：

```yaml
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only # 当前仅支持 deny_only
      sink: callback            # 当前仅支持 callback
      callback:
        require_registered: false
    delivery:
      mode: async               # sync|async，默认 async
      queue:
        size: 128               # 有界队列
        overflow_policy: drop_old
      timeout: 1200ms           # 单次 callback 调用超时
      retry:
        max_attempts: 3         # 最大尝试次数（含首调）
        backoff_initial: 120ms
        backoff_max: 800ms
      circuit_breaker:
        failure_threshold: 5
        open_window: 5s
        half_open_probes: 1
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
3. `mode=async` 下 deny 主路径只保证“入队/快速失败”，不等待 callback 完成；`mode=sync` 下主路径等待 callback 执行结果。
4. 队列满时按 `drop_old` 丢弃最旧待发送事件，保留最新告警。
5. callback 失败不会改变原有安全决策（仍保持 deny），仅追加告警投递失败诊断。
6. 熔断状态机采用 `closed|open|half_open`（Hystrix 风格）：`open` 期间快速失败，窗口到期后进入 `half_open` 试探恢复。
7. Run/Stream 在等价输入与配置下需保持 `policy_kind|decision|reason_code|severity|alert_dispatch_status|alert_delivery_mode|alert_retry_count|alert_circuit_state` 语义等价。

新增 run 诊断字段（增量兼容）：

- `severity`: 归一化严重级别（`low|medium|high`）
- `alert_dispatch_status`: 告警投递状态（`disabled|not_triggered|skipped|queued|succeeded|failed`）
- `alert_dispatch_failure_reason`: 告警投递失败原因码（如 `alert.callback_missing|alert.callback_timeout|alert.retry_exhausted|alert.circuit_open`）
- `alert_delivery_mode`: 投递模式（`sync|async`）
- `alert_retry_count`: 当前事件实际重试次数（不含首调）
- `alert_queue_dropped`: 当前事件入队时是否触发队列丢弃
- `alert_queue_drop_count`: 当前事件入队触发的丢弃数量
- `alert_circuit_state`: 熔断状态（`closed|open|half_open`）
- `alert_circuit_open_reason`: 熔断打开原因码（如 `alert.callback_error|alert.callback_timeout`）

独立安全门禁（required-check 候选）：

- Linux/macOS: `bash scripts/check-security-event-contract.sh`
- Windows: `pwsh -File scripts/check-security-event-contract.ps1`
- CI Job: `security-event-gate`（仅 PR 触发）
- Linux/macOS: `bash scripts/check-security-delivery-contract.sh`
- Windows: `pwsh -File scripts/check-security-delivery-contract.ps1`
- CI Job: `security-delivery-gate`（仅 PR 触发）

## 热更新语义

- 触发机制：监听配置文件变更。
- 执行路径：`parse -> validate -> build snapshot -> atomic swap`。
- 失败策略：任一步失败则拒绝本次更新，保留旧快照，并写入 reload 诊断记录。

## 限制

- `mcp/stdio` 的 `read_pool_size` / `write_pool_size` 当前在 client 初始化时生效；热更新后不动态重建池大小。
- 脱敏规则基于 key 关键词段匹配（如 `secret/token/password/api_key`）；`tokenizer_mode` 等观测字段不会因包含 `token` 子串而被误脱敏。
- `security.redaction.strategy` 当前仅支持 `keyword`，配置其他值会 fail-fast。
- provider fallback 仅在 model-step 边界进行，不支持流式响应开始后的 mid-stream 切换。
- context assembler CA1 仅提供文件 journal（append-only）；数据库后端仅接口占位，配置为 `db` 会启动即 fail-fast。

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

