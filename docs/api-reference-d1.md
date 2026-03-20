# D1 API 参考覆盖（core/runtime/context/skill）

更新时间：2026-03-20

## 目标与范围

本文件定义当前 API 参考覆盖范围（中文优先，接受英文示例）。

覆盖包（外部集成主路径）：
- `core/types`
- `core/runner`
- `runtime/config`
- `runtime/diagnostics`
- `context/assembler`
- `skill/loader`

## 覆盖清单（Audit）

| 包 | 优先导出 API | 最小用途 |
| --- | --- | --- |
| `core/runner` | `New`, `Run`, `Stream`, `WithRuntimeManager` | 构建 agent loop 与执行入口 |
| `core/types` | `RunRequest`, `RunResult`, `ModelClient`, `EventHandler` | 统一请求/响应与扩展契约 |
| `runtime/config` | `NewManager`, `EffectiveConfig`, `RecentRuns`, `TimelineTrends` | 配置加载、诊断查询 |
| `context/assembler` | `New`, `Assemble`, `WithSemanticReranker` | pre-model 上下文拼装与扩展点 |
| `skill/loader` | `NewWithRuntimeManager`, `Discover`, `Compile` | 技能发现与编译 |

## 最小示例（Minimal Examples）

### 1) Runner + Runtime Manager

```go
mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
    FilePath:  "runtime.yaml",
    EnvPrefix: "BAYMAX",
})
if err != nil {
    panic(err)
}
defer mgr.Close()

engine := runner.New(modelClient, runner.WithRuntimeManager(mgr))
res, err := engine.Run(ctx, types.RunRequest{Input: "hello"}, nil)
```

### 2) Context Assembler 扩展注册

```go
asm := assembler.New(
    func() runtimeconfig.ContextAssemblerConfig { return cfg.ContextAssembler },
    assembler.WithSemanticReranker("openai", customReranker),
)
```

### 3) Skill Loader 基础链路

```go
ldr := loader.NewWithRuntimeManager(nil, mgr)
specs, _ := ldr.Discover(ctx, repoRoot)
bundle, _ := ldr.Compile(ctx, specs, types.SkillInput{UserInput: userInput})
```

## 维护约定

- 以上清单中的 API 若有语义变化，应在同一 PR 更新本文档或附带明确 follow-up。
- README 的“文档”入口必须包含本文件链接。

## Adapter Onboarding Navigation（A21）

外部适配接入入口：
- 模板索引：`docs/external-adapter-template-index.md`
- 迁移映射：`docs/adapter-migration-mapping.md`

适配类别（优先级）：
1. MCP adapter template
2. Model provider adapter template
3. Tool adapter template
