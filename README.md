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

当前里程碑快照（2026-03-31）：
- 已归档并稳定：A4-A55。
- A56（Introduce React Loop And Tool Calling Parity Contract）进行中。
- A57（Introduce Sandbox Egress Governance And Adapter Allowlist Contract）进行中。

版本阶段快照：
- 当前仓库保持 `0.x` pre-1 阶段，默认不做 `1.0.0/prod-ready` 承诺。
- `0.x` 阶段允许新增能力型提案，前提是满足提案准入字段与质量门禁阻断要求。
- 提案准入规则与边界以 `docs/development-roadmap.md`、`docs/versioning-and-compatibility.md` 为准。

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
| Adapter Contracts | `adapter/manifest` `adapter/capability` `adapter/scaffold` | 外部适配契约、能力协商与脚手架治理 |
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
- [Adapter Contracts 说明](adapter/README.md)
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

### 5) Mailbox Unified Coordination

```go
mb, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
if err != nil {
	panic(err)
}
bridge := invoke.NewMailboxBridge(mb)

// sync command->result
outcome, err := bridge.InvokeSync(ctx, a2aClient, invoke.Request{
	TaskID:     "task-sync-demo",
	WorkflowID: "wf-demo",
	TeamID:     "team-demo",
	AgentID:    "agent-parent",
	PeerID:     "agent-child",
	Method:     "delegate",
	Payload:    map[string]any{"mode": "sync"},
})
_ = outcome
_ = err

// delayed command
_, err = bridge.PublishDelayedCommand(ctx, invoke.Request{
	TaskID:     "task-delayed-demo",
	WorkflowID: "wf-demo",
	TeamID:     "team-demo",
	AgentID:    "agent-parent",
	PeerID:     "agent-child",
	Method:     "delegate",
}, time.Now().Add(30*time.Second), time.Now().Add(5*time.Minute))
_ = err
```

### 6) Invocation 入口

主线调用入口统一为 `orchestration/mailbox` + `orchestration/invoke/mailbox_bridge`。

### 7) Mailbox Lifecycle Worker（A36）

- 默认值：
  - `mailbox.worker.enabled=false`
  - `mailbox.worker.poll_interval=100ms`
  - `mailbox.worker.handler_error_policy=requeue`
  - `mailbox.worker.inflight_timeout=30s`
  - `mailbox.worker.heartbeat_interval=5s`
  - `mailbox.worker.reclaim_on_consume=true`
  - `mailbox.worker.panic_policy=follow_handler_error_policy`
- worker handler 返回错误时默认按 `requeue` 收敛；panic recover 路径复用同一 policy（`requeue|nack`）。
- stale `in_flight` reclaim 默认在 consume 路径开启；reclaim reason canonical 为 `lease_expired`。
- lifecycle 诊断覆盖：`consume/ack/nack/requeue/dead_letter/expired`，并追加 `reclaimed/panic_recovered` additive 观测标记。

### 8) 能力状态

稳定能力清单（已归档）：
- Runtime 主干：Run/Stream、工具闭环、Context Assembler（CA1-CA4）、Security（S1-S4）。
- 多代理主链路：Teams/Workflow/A2A/Scheduler/Composer、sync/async/delayed、recovery boundary、统一诊断查询与 task board 查询。
- 质量门禁：shared multi-agent contracts、性能基线门禁（含 diagnostics query gate）、sandbox rollout governance gate、全链路 smoke gate、文档一致性 gate。
- 外部适配生态：template、conformance harness、scaffold、manifest、capability negotiation、profile replay gate。

当前进行中能力（最新）：
- A56 `introduce-react-loop-and-tool-calling-parity-contract-a56`：ReAct loop + tool-calling parity 提案进行中（Run/Stream tool dispatch/feedback 语义收敛、provider tool-calling normalization、`react.v1` replay fixture 与独立 gate）。
- A57 `introduce-sandbox-egress-governance-and-adapter-allowlist-contract-a57`：sandbox egress + adapter allowlist 提案进行中（`security.sandbox.egress.*`、`adapter.allowlist.*`、readiness/admission/replay/gate 一体化治理）。

近期已归档能力（摘要）：
- A55 `introduce-observability-export-and-diagnostics-bundle-contract-a55`：已归档并稳定（observability exporter profile、diagnostics bundle schema、readiness/replay/gate 一体化契约）。
- A54 `introduce-memory-provider-spi-and-builtin-filesystem-engine-contract-a54`：已归档并稳定（统一 `Query/Upsert/Delete` SPI、`external_spi|builtin_filesystem` 模式切换、mem0/zep/openviking profile pack、readiness/diagnostics/replay/conformance/gate 一体化契约）。
- A53 `introduce-mainstream-sandbox-adapter-conformance-and-migration-pack-a53`：已归档并稳定（nsjail/bwrap/OCI/windows-job profile pack、manifest compatibility、conformance matrix、template/migration mapping、profile replay 与独立 adapter gate）。
- A52 `introduce-sandbox-runtime-health-rollout-and-capacity-governance-contract-a52`：已归档并稳定（rollout phase 状态机、health budget/freeze、capacity admission action、A52 replay fixture、rollout governance gate）。
- A51 `introduce-sandbox-execution-isolation-contract-a51`：已归档并稳定（`host|sandbox|deny` 决策、`security.sandbox.*` 配置域、readiness/admission 与 replay/gate 收敛）。

### 9) ReAct 最小接入蓝图（A56）

ReAct loop 在主线默认可用（`runtime.react.enabled=true`），Run/Stream 共享同一 loop 终止 taxonomy 与预算语义。

最小配置（`env > file > default`，支持热更新 + 非法更新回滚）：

```yaml
runtime:
  react:
    enabled: true
    max_iterations: 12
    tool_call_limit: 64
    stream_tool_dispatch_enabled: true
    on_budget_exhausted: fail_fast
```

最小接入要点：
- Run 与 Stream 使用同一份 `types.RunRequest`（含 `Policy`），确保 budget/termination 口径等价。
- 为模型适配器实现 tool-calling capability 探测，并保持 provider canonical tool-call 映射一致。
- 在 loop step-boundary 做工具分发与 feedback 回灌，避免 mid-step 语义漂移。
- 通过 `runtime/config.Manager` 注入 readiness/admission，确保 `react.*` finding 与 deny path side-effect-free。

合同门禁（A56）：

```bash
bash scripts/check-react-contract.sh
```

```powershell
pwsh -File scripts/check-react-contract.ps1
```

### 10) Runtime Readiness Admission（A44）

默认值（保持历史行为不变）：
- `runtime.readiness.admission.enabled=false`
- `runtime.readiness.admission.mode=fail_fast`
- `runtime.readiness.admission.block_on=blocked_only`
- `runtime.readiness.admission.degraded_policy=allow_and_record`

启用后，managed `Composer.Run/Stream` 在执行前统一做 admission 判定：
- `blocked`：拒绝执行（fail-fast）。
- `degraded`：按 `degraded_policy` 决定 `allow_and_record` 或 `fail_fast`。
- 拒绝路径保证不触发 scheduler enqueue / mailbox publish / task lifecycle mutation。

历史已归档里程碑：
- A4-A53 的归档明细与能力范围请以 `docs/development-roadmap.md` 与 `openspec/changes/archive/INDEX.md` 为准。
- 主线契约测试映射请查看 `docs/mainline-contract-test-index.md`。

状态权威来源：
- `openspec list --json`
- `openspec/changes/archive/INDEX.md`

## 开发验证

最小建议命令：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
bash scripts/check-react-contract.sh
bash scripts/check-observability-export-and-bundle-contract.sh
bash scripts/check-memory-contract-conformance.sh
bash scripts/check-sandbox-rollout-governance-contract.sh
bash scripts/check-diagnostics-query-performance-regression.sh
```

Windows 质量门禁：

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
pwsh -File scripts/check-react-contract.ps1
pwsh -File scripts/check-observability-export-and-bundle-contract.ps1
pwsh -File scripts/check-memory-contract-conformance.ps1
pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1
pwsh -File scripts/check-diagnostics-query-performance-regression.ps1
```

PowerShell 门禁治理语义（A37）：
- required native command 默认 strict fail-fast（非零即阻断）。
- 唯一非阻断例外为 `govulncheck` 在 `BAYMAX_SECURITY_SCAN_MODE=warn` 时的告警放行。

## 示例

- `examples/01-chat-minimal`：最小单轮问答
- `examples/02-tool-loop-basic`：工具调用闭环
- `examples/03-mcp-mixed-call`：local + MCP 混合
- `examples/04-streaming-interrupt`：流式中断收敛
- `examples/05-parallel-tools-fanout`：并发工具 fanout
- `examples/06-async-job-progress`：异步任务进度回传
- `examples/07-multi-agent-async-channel`：Composer + Scheduler(Local)
- `examples/08-multi-agent-network-bridge`：Composer + Scheduler(A2A)
- `examples/09-multi-agent-full-chain-reference`：Teams + Workflow + A2A + Scheduler + Recovery（Run/Stream + async/delayed/recovery）

## 文档入口

- 路线图与阶段进度：`docs/development-roadmap.md`
- 外部适配模板索引：`docs/external-adapter-template-index.md`
- 适配迁移映射：`docs/adapter-migration-mapping.md`
- 适配一致性验收：`scripts/check-adapter-conformance.sh` / `scripts/check-adapter-conformance.ps1`
- 适配 manifest 合同校验：`scripts/check-adapter-manifest-contract.sh` / `scripts/check-adapter-manifest-contract.ps1`
- 适配能力协商合同校验：`scripts/check-adapter-capability-contract.sh` / `scripts/check-adapter-capability-contract.ps1`
- 适配合同回放校验：`scripts/check-adapter-contract-replay.sh` / `scripts/check-adapter-contract-replay.ps1`
- sandbox adapter conformance 校验：`scripts/check-sandbox-adapter-conformance-contract.sh` / `scripts/check-sandbox-adapter-conformance-contract.ps1`
- 适配脚手架漂移校验：`scripts/check-adapter-scaffold-drift.sh` / `scripts/check-adapter-scaffold-drift.ps1`
- 运行时配置与诊断：`docs/runtime-config-diagnostics.md`
- 模块边界约束：`docs/runtime-module-boundaries.md`
- 核心模块语义映射：`docs/core-module-semantic-alignment.md`
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
