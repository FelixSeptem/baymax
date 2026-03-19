# Baymax Agent Loop (Go)

Baymax 是一个 `library-first`、`contract-first` 的 Go Agent 运行时库，聚焦可嵌入的多代理编排能力：

- 统一 Run/Stream 主循环
- 本地工具与 MCP 双传输（HTTP/STDIO）
- 多模型 Provider 适配（OpenAI/Anthropic/Gemini）
- Context Assembler（CA1-CA4）
- A2A / Workflow / Teams / Scheduler / Composer 组合编排
- 结构化可观测性（timeline + diagnostics + RuntimeRecorder 单写）

最新进度请查看：
- `docs/development-roadmap.md`
- `openspec list --json`

当前里程碑快照（2026-03-19）：
- A17（长任务恢复边界）已归档并稳定。
- A18（统一 run/team/workflow/task 诊断检索 API）已归档并稳定。
- A19（多代理主链路性能基线门禁）进行中。

## 架构设计

Baymax 采用分层组合与单向依赖，核心结构如下：

```text
Application / Host SDK
        |
        v
core/runner + orchestration/* + a2a/*
        |
        v
context/* + tool/local + mcp/http|stdio + model/*
        |
        v
observability/event (RuntimeRecorder single-writer)
        |
        v
runtime/config + runtime/diagnostics
```

关键架构约束：

- `runtime/*` 不反向依赖 MCP 传输实现。
- Provider 协议细节收敛在 `model/<provider>`。
- 诊断写入统一经过 `observability/event.RuntimeRecorder`。
- 配置优先级固定：`env > file > default`。

边界说明见：`docs/runtime-module-boundaries.md`

## 核心模块

| 模块 | 目录 | 作用 |
| --- | --- | --- |
| Runner Core | `core/runner` | Run/Stream 状态机与终止语义 |
| Core Types | `core/types` | 跨模块 DTO、错误分类、契约接口 |
| Model Adapters | `model/openai` `model/anthropic` `model/gemini` | Provider 适配与能力探测 |
| Local Tool Runtime | `tool/local` | 本地工具注册、schema 校验、调度执行 |
| MCP Runtime | `mcp/http` `mcp/stdio` `mcp/profile` `mcp/retry` `mcp/diag` | 远程工具传输与可靠性治理 |
| Context Assembler | `context/assembler` `context/journal` `context/guard` `context/provider` | 上下文装配、检索与守卫 |
| Orchestration | `orchestration/workflow` `orchestration/teams` `orchestration/composer` `orchestration/scheduler` | 工作流、多代理协作、调度与组合入口 |
| A2A Interop | `a2a` | Agent-to-Agent 互联契约（submit/status/result） |
| Runtime Config | `runtime/config` | 配置加载、校验、热更新、回滚 |
| Diagnostics & Eventing | `runtime/diagnostics` `observability/event` `observability/trace` | 可观测性、诊断存储与查询（当前以 `Recent* + Trends` 为主） |
| Skill Loader | `skill/loader` | AGENTS/SKILL 发现、评分、bundle 组装 |
| Runtime Security | `runtime/security` | 脱敏与安全治理基础能力 |

## 组件说明索引

- [A2A Interop 说明](a2a/README.md)
- [Runner Core 说明](core/runner/README.md)
- [Core Types 说明](core/types/README.md)
- [Local Tool Runtime 说明](tool/local/README.md)
- [MCP Runtime 说明](mcp/README.md)
- [Model Adapters 说明](model/README.md)
- [Context Assembler 说明](context/README.md)
- [Orchestration 说明](orchestration/README.md)
- [Runtime Config 说明](runtime/config/README.md)
- [Runtime Diagnostics 说明](runtime/diagnostics/README.md)
- [Runtime Security 说明](runtime/security/README.md)
- [Observability 说明](observability/README.md)
- [Skill Loader 说明](skill/loader/README.md)

## 设计哲学

- **Library First**：优先提供可嵌入、可组合的 Go 库能力。
- **Contract First**：行为变更由 OpenSpec + 契约测试驱动。
- **Fail Fast**：非法配置和非法热更新快速失败并原子回滚。
- **Observability by Default**：timeline/diagnostics 是运行时原语，不是附加功能。
- **Boundary over Convenience**：严格模块边界，减少跨域语义漂移。

## 快速开始

### 1) 环境要求

- Go `1.26+`

### 2) 安装依赖

```bash
go mod tidy
```

### 3) 最小运行示例

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
		Input: "用一句话介绍 Baymax。",
	}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.FinalAnswer)
}
```

可直接运行示例：

```bash
go run ./examples/01-chat-minimal
```

### 4) Composer 最小接入

```go
comp, err := composer.NewBuilder(model).
	WithRuntimeManager(mgr).
	WithEventHandler(dispatcher).
	Build()
if err != nil {
	panic(err)
}

res, err := comp.Run(ctx, types.RunRequest{
	RunID: "run-composer-demo",
	Input: "hello composer",
}, nil)
_ = res
```

### 5) A2A 异步提交与独立回报（A12）

```go
sink := a2a.NewCallbackReportSink(func(ctx context.Context, report a2a.AsyncReport) error {
	// report_key 可用于下游幂等收敛
	fmt.Printf("async report task=%s status=%s key=%s\n", report.TaskID, report.Status, report.ReportKey)
	return nil
})

ack, err := a2aClient.SubmitAsync(ctx, a2a.TaskRequest{
	TaskID: "task-async-demo",
	AgentID: "agent-parent",
	PeerID: "agent-child",
	Method: "delegate",
}, sink)
if err != nil {
	panic(err)
}

fmt.Printf("accepted task=%s at=%s\n", ack.TaskID, ack.AcceptedAt.Format(time.RFC3339Nano))
```

### 6) Scheduler 延后调度（A13）

```go
notBefore := time.Now().Add(30 * time.Second)
record, err := comp.SpawnChild(ctx, composer.ChildDispatchRequest{
	Task: scheduler.Task{
		TaskID:    "task-delayed-demo",
		RunID:     "run-delayed-demo",
		NotBefore: notBefore,
	},
})
if err != nil {
	panic(err)
}
fmt.Printf("delayed task=%s not_before=%s\n", record.Task.TaskID, record.Task.NotBefore.Format(time.RFC3339Nano))
```

### 7) Workflow Graph Composability（A15，默认关闭）

先在 runtime config 显式开启：

```yaml
workflow:
  graph_composability:
    enabled: true
```

最小 composable DSL 示例（`subgraphs + use_subgraph + condition_templates`）：

```yaml
workflow_id: wf-a15-demo
condition_templates:
  gate: "{{when}}"
subgraphs:
  prepare:
    steps:
      - step_id: fetch
        kind: runner
      - step_id: validate
        kind: runner
        depends_on: [fetch]
steps:
  - step_id: prep
    use_subgraph: prepare
    alias: prepare
  - step_id: finalize
    kind: runner
    depends_on: [prep]
    condition_template: gate
    template_vars:
      when: on_success
```

展开后稳定 step_id 形态为 `<subgraph_alias>/<step_id>`，例如 `prepare/fetch`、`prepare/validate`。

### 8) Collaboration Primitives（A16，默认关闭）

先在 runtime config 显式开启：

```yaml
composer:
  collab:
    enabled: true
    default_aggregation: all_settled
    failure_policy: fail_fast
    retry:
      enabled: false
```

库级原语入口位于 `orchestration/collab`，最小聚合示例：

```go
cfg := collab.DefaultConfig()
cfg.Enabled = true

res, err := collab.Execute(ctx, cfg, collab.Request{
	Primitive: collab.PrimitiveAggregation,
	Strategy:  collab.AggregationAllSettled,
	Aggregation: []collab.Branch{
		{
			ID:       "delegate-a",
			Required: true,
			Execute: func(context.Context) (collab.Outcome, error) {
				return collab.Outcome{Status: collab.StatusSucceeded}, nil
			},
		},
	},
})
_ = res
_ = err
```

更多配置字段与诊断口径：`docs/runtime-config-diagnostics.md`

### 9) Long-Running Recovery Boundary（A17）

恢复开启时，A17 默认启用以下边界策略：

- `resume_boundary=next_attempt_only`
- `inflight_policy=no_rewind`
- `timeout_reentry_policy=single_reentry_then_fail`
- `timeout_reentry_max_per_task=1`

最小配置示例：

```yaml
recovery:
  enabled: true
  backend: file
  path: /tmp/baymax/recovery
  conflict_policy: fail_fast
  resume_boundary: next_attempt_only
  inflight_policy: no_rewind
  timeout_reentry_policy: single_reentry_then_fail
  timeout_reentry_max_per_task: 1
```

run 摘要会新增 recovery-boundary 诊断字段：
- `recovery_resume_boundary`
- `recovery_inflight_policy`
- `recovery_timeout_reentry_total`
- `recovery_timeout_reentry_exhausted_total`

### 10) Unified Diagnostics Query（A18）

`runtime/config.Manager` 新增统一 run 诊断检索入口：
- `QueryRuns(query)`

查询能力：
- 过滤字段：`run_id`、`team_id`、`workflow_id`、`task_id`、`status`、`time_range`
- 多条件语义：`AND`
- 分页默认：`page_size=50`，上限 `200`
- 排序默认：`time desc`
- 游标：opaque cursor（不暴露内部 offset/index）

最小调用示例：

```go
pageSize := 20
res, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{
	TeamID:     "team-alpha",
	WorkflowID: "wf-alpha",
	Status:     "failed",
	PageSize:   &pageSize,
})
if err != nil {
	panic(err)
}
for _, item := range res.Items {
	fmt.Printf("run=%s status=%s time=%s\n", item.RunID, item.Status, item.Time.Format(time.RFC3339Nano))
}
if res.NextCursor != "" {
	next, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{
		TeamID:     "team-alpha",
		WorkflowID: "wf-alpha",
		Status:     "failed",
		PageSize:   &pageSize,
		Cursor:     res.NextCursor,
	})
	_ = next
	_ = err
}
```

兼容说明：
- `RecentRuns/RecentCalls/RecentSkills` 与趋势查询接口保持兼容不变。
- 对合法但无匹配的 `task_id`，返回空结果集而非错误。

### 11) Multi-Agent Mainline Performance Gate（A19）

A19 增加多代理主链路性能回归门禁，覆盖：
- `BenchmarkMultiAgentMainlineSyncInvocation`
- `BenchmarkMultiAgentMainlineAsyncReporting`
- `BenchmarkMultiAgentMainlineDelayedDispatch`
- `BenchmarkMultiAgentMainlineRecoveryReplay`

回归检查命令（本地/CI 一致）：

```bash
bash scripts/check-multi-agent-performance-regression.sh
```

```powershell
pwsh -File scripts/check-multi-agent-performance-regression.ps1
```

默认参数与阈值（可被环境变量覆盖）：
- `benchtime=200ms`
- `count=5`
- `ns/op` 最大退化 `8%`
- `p95-ns/op` 最大退化 `12%`
- `allocs/op` 最大退化 `10%`

## 开发验证

最小建议命令：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

Windows 质量门禁：

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
```

## 示例

- `examples/01-chat-minimal`：最小单轮问答
- `examples/02-tool-loop-basic`：工具调用闭环
- `examples/03-mcp-mixed-call`：local + MCP 混合
- `examples/04-streaming-interrupt`：流式中断收敛
- `examples/05-parallel-tools-fanout`：并发工具 fanout
- `examples/06-async-job-progress`：异步任务进度回传
- `examples/07-multi-agent-async-channel`：Composer + Scheduler(Local)
- `examples/08-multi-agent-network-bridge`：Composer + Scheduler(A2A)

## 文档入口

- 路线图与阶段进度：`docs/development-roadmap.md`
- 运行时配置与诊断：`docs/runtime-config-diagnostics.md`
- 模块边界约束：`docs/runtime-module-boundaries.md`
- 主干契约测试索引：`docs/mainline-contract-test-index.md`
- V1 验收与限制：`docs/v1-acceptance.md`
- 版本与兼容策略：`docs/versioning-and-compatibility.md`
- Diagnostics Replay 指南：`docs/diagnostics-replay.md`

## 开源与治理

- 贡献指南：`CONTRIBUTING.md`
- 行为规范：`CODE_OF_CONDUCT.md`
- 安全策略：`SECURITY.md`
- 许可证：`LICENSE`（Apache License 2.0）
- 变更记录：`CHANGELOG.md`
