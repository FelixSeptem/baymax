## Context

M1 已让三 provider 在 `Generate` 路径可用，但 `Stream` 路径仍只有 OpenAI 完整落地。  
同时错误映射仍以基础分类为主，难以在跨 provider 场景中稳定对比和运营。

M2 的核心是“统一语义，不追求 provider 内部细节完全一致”：
- 统一对外流式语义
- 统一错误分类语义
- 保持 fail-fast 和可观测字段一致

## Goals / Non-Goals

**Goals:**
- 实现 Anthropic/Gemini 官方 SDK 流式适配。
- 对齐三 provider 的公共流式事件语义与顺序约束。
- 细化错误分类映射并在文档中给出规则说明。
- 更新现有文档中的阶段状态、能力边界与限制。

**Non-Goals:**
- 不引入 provider 自动降级策略（M3）。
- 不对外暴露 tool-call 参数增量片段。
- 不扩容 examples 批次。

## Decisions

### 1) 继续坚持“完整 tool-call only”
- 决策：外部事件中只发完整 tool call。
- 理由：显著降低跨 provider 事件合并复杂度与误执行风险。
- 说明：内部允许处理参数增量片段用于组装，但不对外透出。

### 2) 允许新增最小必要事件枚举
- 决策：`ModelEvent.Type` 允许新增枚举值以承载跨 provider 统一语义。
- 理由：避免滥用 `meta` 导致事件语义隐式化。
- 约束：仅新增“契约必要”的枚举，不做过度扩展。

### 3) 错误分类按“基础类 + reason”双层表达
- 决策：保留现有 `types.ErrorClass` 基础类，并在 `Details/provider_reason` 细化 reason。
- 理由：兼容现有调用方，同时提升运营诊断精度。

## Streaming Semantic Contract

统一外部语义集合（示例）：
- `response.output_text.delta`
- `tool_call`（complete only）
- `response.completed`
- `response.error`（若需要新增）

统一行为：
- 发生不可恢复错误时 fail-fast 终止。
- 超时统一映射 `ErrPolicyTimeout`。
- 所有事件保持 run/iteration/trace/span 关联能力。

## Risks / Trade-offs

- [Risk] 不同 SDK 的流式事件模型差异较大。  
  [Mitigation] 采用“公共最小语义集”，provider 特性放 `meta`。
- [Risk] 事件枚举新增造成消费端兼容压力。  
  [Mitigation] 保持向后兼容；新增枚举前补契约测试与文档。
- [Risk] 错误细化后行为回归。  
  [Mitigation] 增加跨 provider 错误分类契约测试矩阵。

## Migration Plan

1. 扩展 `core/types` 事件枚举与文档注释。
2. 实现 `model/anthropic` streaming + 完整 tool-call 组装。
3. 实现 `model/gemini` streaming + 完整 tool-call 组装。
4. 对齐 OpenAI 事件映射到统一语义集合（必要时微调）。
5. 增加 integration streaming 契约测试矩阵。
6. 更新 README/docs 并完成质量门禁。
