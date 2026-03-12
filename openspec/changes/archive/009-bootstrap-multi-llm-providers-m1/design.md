## Context

当前模型适配以 `model/openai` 为主，MCP 与 runtime 能力已具备稳定基线，但 provider 维度仍是单点。  
本提案按 R3 M1 目标先交付“多 provider 最小可用”：
- 只做非流式（Run/Generate）路径
- 优先使用官方 SDK
- 错误分类先归拢到基础集合
- 在代码与文档保留 M2 TODO，防止后续遗忘

## Goals / Non-Goals

**Goals:**
- 新增 Anthropic/Gemini 最小 provider 适配并接入统一模型接口。
- 保证三 provider 在最小调用路径上的语义一致性。
- 提供跨 provider 契约测试与基础错误分类映射回归。
- 通过文档标注 M1/M2 边界和 TODO。

**Non-Goals:**
- 不实现 Anthropic/Gemini streaming。
- 不扩展 examples 批次。
- 不改动 MCP/runtime 主体架构。

## Decisions

### 1) Provider 实现优先官方 SDK
- 决策：Anthropic/Gemini 适配优先走官方 SDK。
- 理由：降低后续升级迁移成本，符合既有工程原则。
- 备注：若 SDK 在极少字段上存在空缺，仅允许局部补充协议层封装，不改变“官方优先”原则。

### 2) M1 仅暴露完整非流式调用
- 决策：M1 不引入 streaming 占位接口。
- 理由：减少 API 噪音与未实现承诺，保持接口面简洁。

### 3) 错误分类先归拢到基础集合 + TODO 占位
- 决策：先映射到现有基础 `types.ErrorClass` 语义（认证/限流/超时/请求错误/未知）。
- 理由：满足一致性和可运维需要，避免在 M1 过度细分。
- 约束：在代码与文档加入 TODO，明确 M2 将细化 provider 特有错误分类。

### 4) 配置与密钥管理维持现状
- 决策：M1 使用构造参数 + 环境变量方式管理 key/model，不引入新的配置子系统。
- 理由：保持改动范围聚焦，避免重复建设。

### 5) 测试分层：本地契约为主，线上 smoke 可选
- 决策：以 fake/provider stub + 契约测试覆盖核心语义；真实 provider 线上 smoke 保持可选非阻塞。
- 理由：保证 CI 稳定与可重复性。

## Architecture Sketch

```
core/runner
    |
    v
core/types.ModelClient
    |
    +--> model/openai       (existing)
    +--> model/anthropic    (new, non-stream)
    +--> model/gemini       (new, non-stream)
```

语义约束：
- 三 provider 在非流式输出结构与基础错误分类上保持一致。
- provider-specific 映射逻辑封装在各自 `model/<provider>` 包内。

## Risks / Trade-offs

- [Risk] 不同 SDK 字段语义存在差异，可能导致输出细节不完全一致。  
  [Mitigation] 通过契约测试统一“必要字段”的一致性定义。
- [Risk] M1 只做非流式，短期内仍有能力落差。  
  [Mitigation] 在任务与文档中显式写入 M2 TODO，纳入后续提案。
- [Risk] 错误分类过粗影响深度排障。  
  [Mitigation] 保留原始 provider 错误信息并在 M2 细化映射。

## Migration Plan

1. 新增 `model/anthropic`、`model/gemini` 最小实现与单测。
2. 增加跨 provider 契约测试（成功路径 + 基础错误分类）。
3. 更新 README 和 docs 的 provider 状态与边界说明。
4. 跑通 `go test ./...`、`golangci-lint`、文档一致性检查。

## TODO (for M2)

- 细化 Anthropic/Gemini provider 特有错误分类映射。
- 对齐多 provider streaming 与 tool-call 流式语义。
