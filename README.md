# Baymax Agent Loop (Go)

一个 `library-first` 的 Go Agent Loop 运行时，支持模型循环、工具调度、MCP 双传输、技能加载与可观测性。

## 当前状态（2026-03-13）

- OpenSpec changes `build-go-agent-loop-framework`、`upgrade-openai-native-stream-mapping`、`optimize-runtime-concurrency-and-async-io` 已完成并归档。
- OpenSpec change `harden-mcp-runtime-reliability-profiles` 已完成并归档。
- OpenSpec change `add-runtime-config-and-diagnostics-api-with-hot-reload` 已完成并归档。
- OpenSpec change `refactor-runtime-responsibility-boundaries-and-enrich-docs` 已完成并归档。
- OpenSpec change `unify-diagnostics-contract-and-concurrency-baseline` 已完成并归档。
- OpenSpec change `bootstrap-multi-llm-providers-m1` 已完成并归档。
- OpenSpec change `align-multi-provider-streaming-and-error-taxonomy-m2` 已完成并归档。
- OpenSpec change `add-provider-capability-detection-and-fallback-m3` 已完成实现。
- OpenSpec change `build-context-assembler-ca1-prefix-append-only-baseline` 已完成实现。
- OpenSpec change `implement-context-assembler-ca2-lazy-stage-routing-and-tail-recap` 已完成实现。
- OpenSpec change `harden-security-baseline-s1-govulncheck-and-redaction` 已完成实现。
- OpenSpec change `activate-ca2-external-retriever-spi-and-http-adapter` 已完成实现。
- OpenSpec change `standardize-action-timeline-events-h1` 已完成并归档。
- OpenSpec change `converge-action-timeline-observability-h15` 已完成实现。
- OpenSpec change `implement-context-assembler-ca3-memory-pressure-and-recovery` 已完成实现。
- OpenSpec change `implement-context-assembler-ca4-production-convergence` 已完成实现。
- 核心能力已具备可运行的 v1 基线。
- 关键测试通过：`go test ./...`。

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
- CA3 memory pressure control：
  - 五级分区：`safe|comfort|warning|danger|emergency`
  - CA4 阈值策略：`stage override -> percent/absolute 并行评估 -> 取更高压力分区`
  - 策略动作：warning/danger 触发 squash/prune；emergency 触发 spill/swap + 低优先级加载拒绝
  - 保护标记：`critical`/`immutable` 命中后不参与 squash/prune
  - Token 计数（CA4）：`sdk_preferred` 固定回退链路 `provider -> local tiktoken -> lightweight estimate`，计数失败仅 fail-open（不阻断主流程）
  - OpenAI token 计数语义：用于阈值策略估算，不承诺账单精度
  - 新增 run 诊断字段：`ca3_pressure_zone`、`ca3_pressure_reason`、`ca3_pressure_trigger`、`ca3_zone_residency_ms`、`ca3_trigger_counts`、`ca3_compression_ratio`、`ca3_spill_count`、`ca3_swap_back_count`

### 4. Skill Loader
- AGENTS-first 发现 SKILL
- 显式触发优先 + 语义触发兜底
- 冲突优先级：`system built-in > AGENTS > SKILL`
- 编译输出：`SkillBundle{SystemPromptFragments, EnabledTools, WorkflowHints}`

### 5. Observability
- 事件 schema v1，关联字段：`run_id / iteration / call_id / trace_id / span_id`
- Action Timeline（默认启用，结构化事件类型 `action.timeline`）：
  - phase：`run|context_assembler|model|tool|mcp|skill`
  - status：`pending|running|succeeded|failed|skipped|canceled`
  - 关键字段：`phase`、`status`、`reason`（可选）、`sequence`（单 run 递增）
  - H1.5 聚合字段（`RecentRuns`）：按 phase 输出 `count_total`、`failed_total`、`canceled_total`、`skipped_total`、`latency_ms`、`latency_p95_ms`
- OTel spans：`agent.run` 根 span + model/tool/mcp/skill 子 span
- JSON stdout logger（支持 trace/span/run 关联）
- 诊断写入采用 single-writer（`observability/event.RuntimeRecorder`）+ 幂等去重（`runtime/diagnostics`）
 
说明：H1.5 已完成 timeline 聚合可观测收敛；同一 run 的 timeline 重放不重复计数（幂等保证）。

### 6. Security Baseline (S1)
- 统一脱敏管线：关键词基线（`token/password/secret/api_key/apikey`）+ 扩展 matcher 口
- 脱敏覆盖路径：`runtime/diagnostics`、`observability/event`、`context/assembler`
- 质量门禁新增 `govulncheck`，默认 `strict`（发现漏洞即失败）
- 安全扫描模式支持 `strict|warn`，通过环境变量控制

### 6. Integration + Benchmark
- fake model/tool/mcp 组件
- E2E 测试：多轮 tool loop、mixed local/MCP、streaming 因果顺序
- benchmark：迭代延迟、工具扇出、MCP 重连开销、`BenchmarkCA4PressureEvaluation`（含 `p95-ns/op`）

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

Context Assembler（CA1 + CA2）最小配置示例：
```yaml
context_assembler:
  enabled: true
  journal_path: /tmp/baymax/context-journal.jsonl # 默认值在运行时按 os.TempDir() 计算
  prefix_version: ca1
  storage:
    backend: file # CA1: file|db，其中 db 会 fail-fast 返回 unsupported
  guard:
    fail_fast: true
  ca2:
    enabled: true
    routing_mode: rules # rules|agentic(agentic 预留，当前返回 not-ready)
    stage_policy:
      stage1: fail_fast    # fail_fast|best_effort
      stage2: best_effort  # fail_fast|best_effort
    timeout:
      stage1: 80ms
      stage2: 120ms
    stage2:
      provider: http       # file|http|rag|db|elasticsearch
      file_path: /tmp/baymax/context-stage2.jsonl
      external:
        profile: http_generic # http_generic|ragflow_like|graphrag_like|elasticsearch_like
        endpoint: https://retriever.example.com/search
        method: POST       # POST|PUT
        auth:
          bearer_token: "${RETRIEVER_TOKEN}"
          header_name: Authorization # 留空时默认为 Authorization
        headers:
          X-Tenant: demo
        mapping:
          request:
            mode: plain     # plain|jsonrpc2
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

# 可选：启动前做 external retriever 配置预检查（warning 可继续，error 会阻断）
# result := runtimeconfig.PrecheckStage2External("http", cfg.ContextAssembler.CA2.Stage2.External)
# if err := result.FirstError(); err != nil { panic(err) }
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
    emergency:
      reject_low_priority: true
      high_priority_tokens: [urgent, critical, incident]
    spill:
      enabled: true
      backend: file # file|db|object（db/object 预留接口，当前未实现）
      path: /tmp/baymax/context-spill.jsonl
      swap_back_limit: 4
    tokenizer:
      mode: sdk_preferred # sdk_preferred|estimate_only
      provider: anthropic # anthropic|gemini|openai
      model: claude-3-5-sonnet-latest
      small_delta_tokens: 256
      sdk_refresh_interval: 1200ms
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
  - `scripts/check-repo-hygiene.sh`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-runtime-boundaries.sh`
  - `scripts/check-docs-consistency.ps1`
  - benchmark smoke（`go test ./integration -run ^$ -bench Benchmark -benchtime=50ms`）

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
| `examples/07-multi-agent-async-channel` | Multi-Agent + Structure | 单进程 channel 协作 |
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
- 示例扩容计划：`docs/examples-expansion-plan.md`
- 性能回归策略：`docs/performance-policy.md`
- MCP 可靠性 profile：`docs/mcp-runtime-profiles.md`
- 运行时配置与诊断 API：`docs/runtime-config-diagnostics.md`
- Runtime 模块边界：`docs/runtime-module-boundaries.md`
- Context Assembler 分期计划：`docs/context-assembler-phased-plan.md`
- 模块化评审矩阵：`docs/modular-e2e-review-matrix.md`
- 主干契约测试索引：`docs/mainline-contract-test-index.md`
