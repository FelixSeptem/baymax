## Context

当前 `model/openai` 的 `Stream` 实现通过兼容路径复用 `Generate` 结果，无法准确反映官方 SDK 的原生流式事件语义。这导致：
- 流式事件粒度不足，无法清晰表达模型输出增量与完整 tool call 关系。
- 错误路径在流式阶段的终止语义不够明确，可能给上层造成行为歧义。
- 回归验证偏弱，难以稳定防止事件顺序与错误分类回退。

本变更限定在 OpenAI 流式路径和质量门禁，不触及 MCP 组件重构，以控制范围与交付风险。

## Goals / Non-Goals

**Goals:**
- 通过 OpenAI 官方 Go SDK Responses streaming 实现原生流式适配。
- 定义并固定 SDK event -> `types.ModelEvent` 的映射约定，允许新增事件类型。
- 在 runner 流式路径中落实 fail-fast：流式报错立即中止并返回分类错误。
- 仅在上层暴露完整 tool call（不暴露参数增量片段）。
- 增加 adapter + integration + golden 测试，覆盖顺序、终态、错误分类与语义一致性。
- 引入 `golangci-lint` 与建议配置，形成统一质量门禁。

**Non-Goals:**
- 不改造 MCP HTTP/stdio 的重试与事件共享逻辑。
- 不引入多 provider 统一流式抽象。
- 不扩展到 R2 的配置热更新、告警平台和运维控制面。
- 不包含版本 tag 发布动作。

## Decisions

### Decision 1: 使用 OpenAI 官方 SDK 原生 streaming 事件作为单一事实源
- Choice: 在 `model/openai` 中直接消费 `openai-go` Responses streaming API 的原生事件流。
- Rationale: 避免 compatibility 层语义丢失，降低后续 SDK 升级迁移成本。
- Alternatives considered:
  - 继续兼容实现：改动小但语义不完整，无法满足 R1 目标。
  - 自建中间 streaming 协议层：灵活但复杂度更高，当前收益不足。

### Decision 2: `types.ModelEvent` 采用向后兼容扩展
- Choice: 保留现有字段和事件消费方式，新增必要 `Type` 枚举来表达原生流式语义。
- Rationale: 兼顾现有调用方兼容与事件表达能力。
- Alternatives considered:
  - 仅用现有类型复用：表达力不足，后续测试与文档难以清晰约束。
  - 破坏式重构事件模型：成本高，超出本次范围。

### Decision 3: Tool call 只在“完整可用”时对外发射
- Choice: 对外只暴露完整 tool call 事件，不暴露参数增量片段。
- Rationale: 保持上层协议简单稳定，减少消费方处理复杂度。
- Alternatives considered:
  - 暴露参数增量：信息更细，但会放大事件处理复杂度并引入更多边界条件。

### Decision 4: Stream 错误采用 fail-fast
- Choice: 流式任一不可恢复错误出现时立即停止流并返回错误。
- Rationale: 明确行为边界，减少“部分成功 + 隐式失败”歧义。
- Alternatives considered:
  - 尽量继续输出已收集片段：容错更强，但错误语义与上层一致性更复杂。

### Decision 5: 双层测试策略 + golden 固化
- Choice: 在 `model/openai` 做映射单测，在 `integration` 做端到端顺序/终态验证，并引入 golden 事件序列。
- Rationale: 单测保证映射正确，集成测试保证 runtime 行为稳定，golden 防止无意变更。
- Alternatives considered:
  - 只做 integration：定位问题困难。
  - 只做单测：无法覆盖 runner 侧事件与终态收敛。

### Decision 6: 接入 `golangci-lint` 作为质量门禁
- Choice: 新增 `.golangci.yml` 初始配置，覆盖静态分析、错误处理、格式化与 import 规范。
- Rationale: 与 R1 发布准备目标一致，降低回归与风格漂移风险。
- Alternatives considered:
  - 仅依赖 gofmt/go test：对潜在缺陷发现不足。

## Risks / Trade-offs

- [SDK 事件协议随版本演进发生细节变化] → 将映射逻辑集中在 adapter 单点并用 golden 固化，可快速识别与修复。
- [新增事件枚举导致消费方处理不全] → 保持向后兼容，文档明确“未识别事件应忽略或按默认处理”。
- [fail-fast 导致部分输出不可用] → 这是有意策略；通过清晰错误分类与日志关联帮助快速定位。
- [lint 首次接入可能带来告警激增] → 采用分阶段治理，先建立基线并在后续迭代逐步收紧。
