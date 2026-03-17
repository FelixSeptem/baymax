# Baymax Agent Loop (Go)

一个 `library-first` 的 Go Agent Loop 运行时，支持模型循环、工具调度、MCP 双传输、技能加载与可观测性。

## 当前状态（2026-03-16）

- OpenSpec 活跃变更请以 `openspec list --json` 实时结果为准。
- 最近归档：`035-implement-ca3-semantic-embedding-adapter-e3`。
- 核心能力已覆盖：多 Provider、CA1-CA4、Action Timeline H1/H1.5、Action Gate H2、安全基线 S1。
- 归档清单见：`openspec/changes/archive/INDEX.md`。

## Open Source P0

- 版本与兼容承诺：`docs/versioning-and-compatibility.md`
- 安全响应入口：`SECURITY.md`（使用 GitHub Security Advisory 私密披露）
- 贡献与评审流程：`CONTRIBUTING.md`
- 社区行为规范：`CODE_OF_CONDUCT.md`
- 变更记录模板：`CHANGELOG.md`

## 已实现能力

### 1. Runner Loop
- 显式状态机：`Init -> ModelStep -> DecideNext -> Finalize/Abort`
- 支持 `Run`（非流式）与 `Stream`（流式）路径
- 终止条件：final answer、超时、迭代上限、策略中止

### 2. Local Tool Runtime
- 工具命名空间：`local.<name>`
- 入参 JSON schema 校验
- 单轮并发调度 + 写工具串行执行
- fail-fast / continue-on-error 策略

### 3. MCP Runtime
- `mcp/stdio`：warmup、池化、超时、重试、事件归一化
- `mcp/http`：官方 go-sdk 适配、心跳、重连、稳定 call-id、事件顺序保证
- `mcp/internal/*`：共享可靠性与可观测性核心（internal-only，供 http/stdio 复用）
- 统一运行时配置：`viper` 加载 YAML + Env 覆盖（`env > file > default`）
- 热更新：原子配置切换，非法更新自动回滚
- 诊断 API（库接口）：最近 MCP 调用/Run 摘要、脱敏生效配置查询（无 CLI）

### 3.5 Model Providers
- `model/openai`：官方 SDK，支持 `Generate` + 原生 `Stream`
- `model/anthropic`：官方 SDK，支持 `Generate` + `Stream`（tool-call complete-only）
- `model/gemini`：官方 SDK，支持 `Generate` + `Stream`（tool-call complete-only）
- 能力探测：通过各 provider 官方 SDK 的 `Models.Get`/元数据接口动态发现；无法判定时返回受控 `unknown`
- provider 级降级：model-step 前 capability preflight，按 `provider_fallback.providers` 有序尝试，候选耗尽即 fail-fast
- 错误映射：基础 `types.ErrorClass` + `provider_reason`（`auth/rate_limit/timeout/request/server/unknown`）

### 3.6 Context Assembler（CA1 + CA2 + CA3 + CA4）
- `context/assembler` 已接入 `core/runner` pre-model hook（Run/Stream 双路径）
- immutable prefix + `prefix_hash` 校验（同 session/version 漂移即 fail-fast）
- `context/journal` 本地 JSONL append-only（intent/commit）
- `context/guard` 基础规则（hash/schema/sanitize），默认 fail-fast
- storage backend：`file` 生效，`db` 在 CA1 显式返回 unsupported
- 诊断字段已写入 run 摘要：`prefix_hash`、`assemble_latency_ms`、`assemble_status`、`guard_violation`
- CA2 staged assembly：Stage1 -> Stage2 规则路由（满足条件才触发 Stage2）
- Stage2 provider：支持 `file/http/rag/db/elasticsearch`
- External Retriever：通用 SPI + HTTP 适配层，支持 profile 模板、JSON 字段映射、Bearer 与自定义鉴权头
- 支持 stage 失败策略配置（`fail_fast` / `best_effort`）
- 支持 tail recap（最小字段 `status/decisions/todo/risks`）并追加在上下文末尾
- 增强诊断字段：`assemble_stage_status`、`stage2_skip_reason`、`stage1_latency_ms`、`stage2_latency_ms`、`stage2_provider`、`stage2_profile`、`stage2_hit_count`、`stage2_source`、`stage2_reason`、`stage2_reason_code`、`stage2_error_layer`、`recap_status`
- CA2 external 观测阈值：支持 provider 维度趋势聚合（`p95_latency_ms`、`error_rate`、`hit_rate`）与静态阈值命中信号（仅观测，不自动动作）
- CA3 memory pressure control：
  - 五级分区：`safe|comfort|warning|danger|emergency`
  - CA4 阈值策略：`stage override -> percent/absolute 并行评估 -> 取更高压力分区`
  - 策略动作：warning/danger 触发 squash/prune；emergency 触发 spill/swap + 低优先级加载拒绝
- CA3 compaction 策略：
  - `context_assembler.ca3.compaction.mode`：`truncate|semantic`（默认 `truncate`）
  - `semantic` 通过当前 model client 路径执行语义压缩；`best_effort` 失败回退 `truncate`，`fail_fast` 失败即终止
  - quality gate：`context_assembler.ca3.compaction.quality.threshold + weights`（coverage/compression/validity）
  - reranker：`context_assembler.ca3.compaction.reranker.*`（默认关闭）
    - `enabled`、`timeout`、`max_retries`
    - `threshold_profiles`（key=`provider:model`，开启 reranker 时必须包含当前 provider/model）
    - `governance.mode`：`enforce|dry_run`（`dry_run` 仅评估不改变最终 gate）
    - `governance.profile_version`：阈值配置版本标签（用于观测与排障）
    - `governance.rollout_provider_models`：按 `provider:model` 灰度匹配
    - 支持 provider-specific 扩展接口（`assembler.WithSemanticReranker`），未注册时走内置默认 reranker
  - semantic template：`context_assembler.ca3.compaction.semantic_template.prompt + allowed_placeholders`（启动/热更新 fail-fast 校验）
  - embedding adapter：支持 `openai|gemini|anthropic` provider 选择，默认 `cosine` 混合评分（`rule_weight=0.7`、`embedding_weight=0.3`）
  - embedding 凭证：支持 `embedding.auth.*` 独立配置与 `embedding.provider_auth.<provider>.*` 覆盖
  - Anthropic embedding：E4 起提供可用路径（不再是 unsupported-only 分支）
  - prune 证据保留：`context_assembler.ca3.compaction.evidence.keywords` + `recent_window`
- 保护标记：`critical`/`immutable` 命中后不参与 squash/prune
- Token 计数（CA4）：`sdk_preferred` 固定回退链路 `provider -> local tiktoken -> lightweight estimate`，计数失败仅 fail-open（不阻断主流程）
- OpenAI token 计数语义：用于阈值策略估算，不承诺账单精度
- 新增 run 诊断字段：`ca3_pressure_zone`、`ca3_pressure_reason`、`ca3_pressure_trigger`、`ca3_zone_residency_ms`、`ca3_trigger_counts`、`ca3_compression_ratio`、`ca3_spill_count`、`ca3_swap_back_count`、`ca3_compaction_mode`、`ca3_compaction_fallback`、`ca3_compaction_fallback_reason`、`ca3_compaction_quality_score`、`ca3_compaction_quality_reason`、`ca3_compaction_embedding_provider`、`ca3_compaction_embedding_similarity`、`ca3_compaction_embedding_contribution`、`ca3_compaction_embedding_status`、`ca3_compaction_embedding_fallback_reason`、`ca3_compaction_reranker_used`、`ca3_compaction_reranker_provider`、`ca3_compaction_reranker_model`、`ca3_compaction_reranker_threshold_source`、`ca3_compaction_reranker_threshold_hit`、`ca3_compaction_reranker_fallback_reason`、`ca3_compaction_reranker_profile_version`、`ca3_compaction_reranker_rollout_hit`、`ca3_compaction_reranker_threshold_drift`、`ca3_compaction_retained_evidence_count`

### 3.7 HITL（H2 + H3 + H4）
- 工具执行前 Gate：在 `core/runner` 的 tool dispatch 前执行风险判定（首期规则仅 `tool name + keyword`）。
- 默认策略：`require_confirm`（若需要确认但未配置 resolver，直接 deny + fail-fast）。
- 超时策略：resolver 超时统一按 deny（`timeout-deny`）。
- Run/Stream 语义：`allow/deny/timeout` 的错误分类与 timeline reason code 保持一致。
- timeline reason code：`gate.require_confirm`、`gate.denied`、`gate.timeout`。
- run 诊断最小字段：`gate_checks`、`gate_denied_count`、`gate_timeout_count`。
- H3 Clarification：支持运行中 `await_user -> resumed -> canceled_by_user` 生命周期（单进程）。
- 结构化事件：`hitl.clarification.requested`，payload 内包含 `clarification_request`（`request_id/questions/context_summary/timeout_ms`）。
- 默认超时策略：`cancel_by_user`（fail-fast 终止当前 run）。
- H3 timeline reason code：`hitl.await_user`、`hitl.resumed`、`hitl.canceled_by_user`。
- H3 run 诊断最小字段：`await_count`、`resume_count`、`cancel_by_user_count`。
- H4 参数规则：支持 `path + operator + expected` 和复合条件（AND/OR）。
- H4 操作符：`eq`、`ne`、`contains`、`regex`、`in`、`not_in`、`gt`、`gte`、`lt`、`lte`、`exists`。
- H4 优先级：参数规则 > `decision_by_tool/decision_by_keyword` > 既有默认规则路径。
- H4 timeline reason code：`gate.rule_match`。
- H4 run 诊断最小字段：`gate_rule_hit_count`、`gate_rule_last_id`。

### 3.8 Runner 并发基线（R5）
- 默认背压策略为 `block`，高 fanout 且命中排队场景会发射 `backpressure.block` timeline reason。
- 取消传播场景新增 `cancel.propagated` timeline reason，Run/Stream 保持语义一致。
- 新增并发诊断字段：`cancel_propagated_count`、`backpressure_drop_count`、`backpressure_drop_count_by_phase`、`inflight_peak`。
- 新增 runtime 配置字段：`concurrency.cancel_propagation_timeout`（`env > file > default`，无效值 fail-fast）。
- 背压模式 `drop_low_priority` 作用于 `local + mcp + skill` 调度语义，timeline reason 为 `backpressure.drop_low_priority`。
- `drop_low_priority` 下若任一调度路径在同一轮调用中全量被丢弃，runner 立即 fail-fast 终止。

### 4. Skill Loader
- AGENTS-first 发现 SKILL
- 显式触发优先 + 语义触发兜底
- 语义触发评分：默认 `lexical_weighted_keywords`（关键词加权 + 阈值命中）
- tie-break：默认 `highest_priority`（同分时按 `SkillSpec.Priority` 决策）
- 低置信度抑制：默认开启（`suppress_low_confidence=true`）
- 冲突优先级：`system built-in > AGENTS > SKILL`
- 评分策略可通过 `runtime.yaml` 的 `skill.trigger_scoring.*` 配置；embedding scorer 仅保留 TODO 扩展口（本期不启用）
- 编译输出：`SkillBundle{SystemPromptFragments, EnabledTools, WorkflowHints}`

### 5. Observability
- 事件 schema v1，关联字段：`run_id / iteration / call_id / trace_id / span_id`
- Action Timeline（默认启用，结构化事件类型 `action.timeline`）：
  - phase：`run|context_assembler|model|tool|mcp|skill|hitl`
  - status：`pending|running|succeeded|failed|skipped|canceled`
  - 关键字段：`phase`、`status`、`reason`（可选）、`sequence`（单 run 递增）
  - H1.5 聚合字段（`RecentRuns`）：按 phase 输出 `count_total`、`failed_total`、`canceled_total`、`skipped_total`、`latency_ms`、`latency_p95_ms`
  - H16 趋势聚合字段（`TimelineTrends`）：按 `phase+status` 输出 `count_total`、`failed_total`、`canceled_total`、`skipped_total`、`latency_avg_ms`、`latency_p95_ms`、`window_start`、`window_end`
- OTel spans：`agent.run` 根 span + model/tool/mcp/skill 子 span
- JSON stdout logger（支持 trace/span/run 关联）
- 诊断写入采用 single-writer（`observability/event.RuntimeRecorder`）+ 幂等去重（`runtime/diagnostics`）
 
说明：H1.5/H16 已完成 timeline 单 run 聚合 + 跨 run 趋势聚合收敛；同一 run 的 timeline 重放不重复计数（幂等保证）。

### 6. Security Baseline (S1)
- 统一脱敏管线：关键词基线（`token/password/secret/api_key/apikey`）+ 扩展 matcher 口
- 脱敏覆盖路径：`runtime/diagnostics`、`observability/event`、`context/assembler`
- 质量门禁新增 `govulncheck`，默认 `strict`（发现漏洞即失败）
- 安全扫描模式支持 `strict|warn`，通过环境变量控制

### 7. Integration + Benchmark
- fake model/tool/mcp 组件
- E2E 测试：多轮 tool loop、mixed local/MCP、streaming 因果顺序
- benchmark：迭代延迟、工具扇出、MCP 重连开销、`BenchmarkCA4PressureEvaluation`（含 `p95-ns/op`）
- benchmark：`BenchmarkCA3SemanticCompactionLatency`、`BenchmarkCA3SemanticCompactionLatencyEmbeddingEnabled`、`BenchmarkCA3SemanticCompactionLatencyRerankerGovernanceEnabled`（CA3 semantic 路径，纳入相对百分比回归策略）
- benchmark：`BenchmarkToolFanOutCancelStorm` 输出 `p95-ns/op` + `goroutine-peak`，用于取消风暴回归对比
- benchmark：`BenchmarkDiagnosticsTimelineTrendQuery` 输出趋势查询性能指标（含 `p95-ns/op`）

## 快速开始

### 环境
- Go `1.26+`

### Quickstart（最小示例）
下面示例使用 OpenAI 官方 Go SDK 适配器，直接跑一个单轮问答（不启用工具）。

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	openaiadapter "github.com/FelixSeptem/baymax/model/openai"
)

func main() {
	model := openaiadapter.NewClient(openaiadapter.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4.1-mini",
	})

	engine := runner.New(model)
	res, err := engine.Run(context.Background(), types.RunRequest{
		Input: "用一句话介绍 Baymax Agent Loop。",
	}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.FinalAnswer)
}
```

运行：
```bash
export OPENAI_API_KEY="<your-api-key>"
go run ./examples/01-chat-minimal
```

### 安装依赖
```bash
go mod tidy
```

### Runtime Config（可选）
建议通过 `runtime/config.Manager` 启用统一配置与热更新：

```go
mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
    FilePath:        "runtime.yaml",
    EnvPrefix:       "BAYMAX",
    EnableHotReload: true,
})
if err != nil {
    panic(err) // fail-fast
}
defer mgr.Close()
```

更多字段与环境变量映射见：`docs/runtime-config-diagnostics.md`

Action Gate（H2）最小配置示例：
```yaml
action_gate:
  enabled: true
  policy: require_confirm # allow|require_confirm|deny
  timeout: 15s            # resolver 超时按 deny
  tool_names: []          # tool name 风险规则
  keywords: []            # keyword 风险规则（input + args）
  decision_by_tool:
    shell: require_confirm
    delete: deny
  decision_by_keyword:
    "rm -rf": deny
  parameter_rules:
    - id: require-confirm-shell-danger
      tool_names: [shell]
      action: require_confirm # 缺省继承 policy
      condition:
        all:
          - path: cmd
            operator: contains
            expected: rm -rf
          - path: force
            operator: eq
            expected: true
```

并发基线（R5）最小配置示例：
```yaml
concurrency:
  local_max_workers: 8
  local_queue_size: 32
  backpressure: drop_low_priority # block|reject|drop_low_priority
  cancel_propagation_timeout: 1500ms
  drop_low_priority:
    priority_by_tool:
      local.search: low
    priority_by_keyword:
      cache: low
    droppable_priorities: [low]
```

Action Timeline 趋势聚合（H16）最小配置示例：
```yaml
diagnostics:
  timeline_trend:
    enabled: true
    last_n_runs: 100
    time_window: 15m
```

CA2 External Retriever 趋势与阈值（E2）最小配置示例：
```yaml
diagnostics:
  ca2_external_trend:
    enabled: true
    window: 15m
    thresholds:
      p95_latency_ms: 1500
      error_rate: 0.10
      hit_rate: 0.20
```

Context Assembler（最小配置示例）：
```yaml
context_assembler:
  enabled: true
  journal_path: /tmp/baymax/context-journal.jsonl
  prefix_version: ca1
  storage:
    backend: file # file|db（db 当前为占位，启动 fail-fast）
  ca2:
    enabled: true
    routing_mode: rules
    stage2:
      provider: http # file|http|rag|db|elasticsearch
      external:
        profile: http_generic
        endpoint: https://retriever.example.com/search
  ca3:
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
        prompt: "Compress... Keep output under {{max_runes}} characters.\n\nUser input:\n{{input}}\n\nSource:\n{{source}}"
        allowed_placeholders: [input, source, max_runes, model, messages_count]
      embedding:
        enabled: false
        selector: ""
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
          openai: { api_key: "", base_url: "" }
          gemini: { api_key: "", base_url: "" }
          anthropic: { api_key: "", base_url: "" }
      reranker:
        enabled: false
        timeout: 500ms
        max_retries: 1
        governance:
          mode: enforce
          profile_version: ""
          rollout_provider_models: []
        threshold_profiles:
          openai:text-embedding-3-small: 0.62
      evidence:
        keywords: [decision, constraint, todo, risk]
        recent_window: 0

clarification:
  enabled: true
  timeout: 30s
  timeout_policy: cancel_by_user
```

完整字段、默认值、校验与诊断口径请以 `docs/runtime-config-diagnostics.md` 为准。

CA3 阈值调优（离线工具，最小 markdown 输出）：
```bash
go run ./cmd/ca3-threshold-tuning -input tuning-input.json -output tuning-report.md
```

### 运行测试
```bash
go test ./...
```

并发安全基线（race）建议使用：
```bash
bash scripts/check-quality-gate.sh
```
Windows PowerShell：
```powershell
pwsh -File scripts/check-quality-gate.ps1
```

安全扫描会调用 `govulncheck`。如未安装，可先执行：
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

可选扫描策略（默认 strict）：
```bash
export BAYMAX_SECURITY_SCAN_MODE=warn
export BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED=true
```

### 运行 Lint
```bash
golangci-lint run --config .golangci.yml
```

在受限环境下可指定缓存目录：
```bash
GOLANGCI_LINT_CACHE=.gocache/golangci-lint golangci-lint run --config .golangci.yml
```

### 跑基准测试
```bash
go test ./integration -run ^$ -bench Benchmark -benchtime=100ms
```

性能回归策略（相对提升百分比）见：`docs/performance-policy.md`
CA4 基准回归门禁可单独执行：
```bash
bash scripts/check-ca4-benchmark-regression.sh
```

### CI
- 仓库内置 CI 工作流：`.github/workflows/ci.yml`
- 默认执行：
  - `scripts/check-quality-gate.sh`
  - `scripts/check-runtime-boundaries.sh`
  - `scripts/check-docs-consistency.ps1`
  - benchmark smoke（`go test ./integration -run ^$ -bench Benchmark -benchtime=50ms`）

### 安全报告
- 漏洞报告请使用 GitHub Security Advisory（私密报告流程），详见：`SECURITY.md`

## 脚本清单（当前保留）

- `scripts/check-repo-hygiene.sh`：仓库卫生检查（禁止临时/备份产物，如 `*.go.<random>`）。
- `scripts/check-repo-hygiene.ps1`：Windows 等价仓库卫生检查脚本。
- `scripts/check-quality-gate.sh`：Linux 质量门禁（`repo hygiene` + `go test` + `go test -race` + `golangci-lint` + `govulncheck`）。
- `scripts/check-quality-gate.ps1`：Windows 质量门禁等价脚本（同上语义）。
- `scripts/check-ca4-benchmark-regression.sh`：CA4 benchmark 回归检查（`ns/op` + `p95-ns/op` 相对百分比门禁）。
- `scripts/check-ca4-benchmark-regression.ps1`：Windows 等价 CA4 benchmark 回归检查脚本。
- `scripts/ca4-benchmark-baseline.env`：CA4 benchmark 基线与默认阈值配置。
- `scripts/check-runtime-boundaries.sh`：runtime 模块边界静态检查。
- `scripts/check-docs-consistency.ps1`：README/docs 引用与关键章节一致性检查。
- `scripts/openspec-archive-seq.ps1`：OpenSpec 归档序号规范化与归档索引维护。

## Examples Pattern Index

| Example | Pattern | 说明 |
| --- | --- | --- |
| `examples/01-chat-minimal` | Sequential | 单轮最小调用链路 |
| `examples/02-tool-loop-basic` | Tool Call + Sequential | 工具调用闭环 |
| `examples/03-mcp-mixed-call` | Tool Call + Routing | local/MCP 混合路径 |
| `examples/04-streaming-interrupt` | Structure | 流式中断与收敛 |
| `examples/05-parallel-tools-fanout` | Parallel | goroutine fanout 并发工具执行 |
| `examples/06-async-job-progress` | Map-Reduce-like + Parallel | 异步任务进度回传与聚合 |
| `examples/07-multi-agent-async-channel` | Multi-Agent + Structure + HITL Clarification | 单进程 channel 协作与 await/resume 演示 |
| `examples/08-multi-agent-network-bridge` | Multi-Agent + Structure (Network) | HTTP + JSON-RPC 2.0 网络桥接 |

运行示例：
```bash
go run ./examples/05-parallel-tools-fanout
go run ./examples/06-async-job-progress
go run ./examples/07-multi-agent-async-channel
go run ./examples/08-multi-agent-network-bridge
```

## 目录结构

- `core/types`: 公共接口与 DTO
- `core/runner`: 主循环状态机
- `tool/local`: 本地工具注册与调度
- `mcp/stdio`, `mcp/http`: MCP 适配层
- `model/openai`: OpenAI 官方 SDK 适配
- `model/anthropic`: Anthropic 官方 SDK 适配
- `model/gemini`: Gemini 官方 SDK 适配
- `skill/loader`: AGENTS/SKILL 发现与编译
- `context/provider`: CA2 stage2 retrieval SPI、file provider 与 external HTTP adapter
- `observability/event`, `observability/trace`: 事件与 trace
- `integration/`: E2E 与 benchmark
- `docs/`: 验收文档与 roadmap

## 文档

- V1 验收与限制：`docs/v1-acceptance.md`
- 开发路线图：`docs/development-roadmap.md`
- 版本与兼容策略：`docs/versioning-and-compatibility.md`
- 示例扩容计划：`docs/examples-expansion-plan.md`
- 性能回归策略：`docs/performance-policy.md`
- MCP 可靠性 profile：`docs/mcp-runtime-profiles.md`
- 运行时配置与诊断 API：`docs/runtime-config-diagnostics.md`
- Runtime 模块边界：`docs/runtime-module-boundaries.md`
- Context Assembler 分期计划：`docs/context-assembler-phased-plan.md`
- 模块化评审矩阵：`docs/modular-e2e-review-matrix.md`
- 主干契约测试索引：`docs/mainline-contract-test-index.md`
