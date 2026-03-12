# Baymax Agent Loop (Go)

一个 `library-first` 的 Go Agent Loop 运行时，支持模型循环、工具调度、MCP 双传输、技能加载与可观测性。

## 当前状态（2026-03-11）

- OpenSpec changes `build-go-agent-loop-framework`、`upgrade-openai-native-stream-mapping`、`optimize-runtime-concurrency-and-async-io` 已完成并归档。
- OpenSpec change `harden-mcp-runtime-reliability-profiles` 已完成并归档。
- OpenSpec change `add-runtime-config-and-diagnostics-api-with-hot-reload` 已完成并归档。
- OpenSpec change `refactor-runtime-responsibility-boundaries-and-enrich-docs` 已完成并归档。
- OpenSpec change `unify-diagnostics-contract-and-concurrency-baseline` 已完成并归档。
- OpenSpec change `bootstrap-multi-llm-providers-m1` 进行中（Anthropic/Gemini 非流式最小适配）。
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
- `model/anthropic`：官方 SDK，M1 支持最小 `Generate`（非流式）
- `model/gemini`：官方 SDK，M1 支持最小 `Generate`（非流式）
- TODO（R3 M2）：补齐 Anthropic/Gemini streaming 语义对齐与细粒度错误映射

### 4. Skill Loader
- AGENTS-first 发现 SKILL
- 显式触发优先 + 语义触发兜底
- 冲突优先级：`system built-in > AGENTS > SKILL`
- 编译输出：`SkillBundle{SystemPromptFragments, EnabledTools, WorkflowHints}`

### 5. Observability
- 事件 schema v1，关联字段：`run_id / iteration / call_id / trace_id / span_id`
- OTel spans：`agent.run` 根 span + model/tool/mcp/skill 子 span
- JSON stdout logger（支持 trace/span/run 关联）
- 诊断写入采用 single-writer（`observability/event.RuntimeRecorder`）+ 幂等去重（`runtime/diagnostics`）

### 6. Integration + Benchmark
- fake model/tool/mcp 组件
- E2E 测试：多轮 tool loop、mixed local/MCP、streaming 因果顺序
- benchmark：迭代延迟、工具扇出、MCP 重连开销

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

### CI
- 仓库内置 CI 工作流：`.github/workflows/ci.yml`
- 默认执行：
  - `scripts/check-quality-gate.sh`
  - `scripts/check-runtime-boundaries.sh`
  - `scripts/check-docs-consistency.ps1`
  - benchmark smoke（`go test ./integration -run ^$ -bench Benchmark -benchtime=50ms`）
  - `golangci-lint`

## 脚本清单（当前保留）

- `scripts/check-quality-gate.sh`：Linux CI 质量门禁（`go test` + `go test -race`）。
- `scripts/check-quality-gate.ps1`：Windows 本地质量门禁等价脚本。
- `scripts/check-runtime-boundaries.sh`：runtime 模块边界静态检查。
- `scripts/check-docs-consistency.ps1`：README/docs 引用与关键章节一致性检查。
- `scripts/openspec-archive-seq.ps1`：OpenSpec 归档序号规范化与归档索引维护。

## 目录结构

- `core/types`: 公共接口与 DTO
- `core/runner`: 主循环状态机
- `tool/local`: 本地工具注册与调度
- `mcp/stdio`, `mcp/http`: MCP 适配层
- `model/openai`: OpenAI 官方 SDK 适配
- `skill/loader`: AGENTS/SKILL 发现与编译
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
