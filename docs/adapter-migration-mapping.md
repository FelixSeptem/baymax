# Adapter Migration Mapping (A21)

更新时间：2026-03-19

## 目标

提供统一迁移映射，按以下双维度组织：
- capability-domain（能力域）
- code-snippet（代码片段）

每条映射至少包含：
- previous pattern
- recommended pattern
- compatibility notes

## Capability-Domain Mapping

| 能力域 | previous pattern | recommended pattern | compatibility notes |
| --- | --- | --- | --- |
| MCP adapter | 在业务代码中直接散落网络调用与重试逻辑 | 收敛到 `mcp/http` 或 `mcp/stdio` 客户端，并由 profile/runtime policy 管理 | additive 字段可增量引入；旧字段缺失走 default；非法配置 fail-fast |
| Model adapter | 直接在上层绑定 provider SDK 类型 | 在 `model/<provider>` 实现 `types.ModelClient` + 能力探测接口 | nullable 字段允许为空；新增能力字段保持 backward-safe |
| Tool adapter | 工具执行逻辑与业务主流程耦合 | 使用 `types.Tool` + `tool/local.Registry` 显式注册 | schema 变更需保持 additive 优先，破坏性变更需 fail-fast |

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
