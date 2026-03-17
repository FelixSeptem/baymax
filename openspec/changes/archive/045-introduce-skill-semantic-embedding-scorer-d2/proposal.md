## Why

当前 skill 触发仅依赖 lexical weighted-keyword，复杂语义输入下存在误触发与漏触发风险。随着多技能场景增多，需要在保持默认行为稳定的前提下，引入可选的 embedding 语义评分增强能力，并提供可观测与可回退机制。

## What Changes

- 在 `skill/loader` 增加 embedding scorer 扩展接口，支持宿主注册实现。
- 扩展 `skill.trigger_scoring.strategy`，新增 `lexical_plus_embedding` 策略（默认仍为 `lexical_weighted_keywords`）。
- 在 `lexical_plus_embedding` 下使用线性加权融合分数（`final = lexical_weight * lexical + embedding_weight * embedding`）。
- embedding 路径不可用/超时/错误时，按 best-effort 回退 lexical，不中断 skill 选择流程。
- 扩展运行时配置 `skill.trigger_scoring.embedding.*`（仅 JSON/YAML 配置路径），并纳入 startup/hot-reload fail-fast 校验。
- 新增 skill 触发诊断字段（策略、分数、回退原因）以支持调优与排障。
- 强制补齐 Run/Stream 语义等价契约测试（等价输入与配置下 skill 选择结果一致）。

## Capabilities

### New Capabilities

- 无

### Modified Capabilities

- `skill-trigger-scoring`: 从 lexical-only 扩展为 lexical+embedding 可选增强，定义线性加权、回退语义与等价性契约。
- `runtime-config-and-diagnostics-api`: 增加 skill embedding scoring 配置与诊断可观测字段，并定义对应校验与热更新行为。

## Impact

- 受影响代码：
  - `skill/loader/*`（评分策略、扩展接口、选择流程、事件字段）
  - `runtime/config/*`（配置结构、默认值、校验、加载映射、热更新回滚）
  - `runtime/diagnostics/*` 与 `observability/event/*`（skill 诊断字段映射）
  - `core/runner/*`（Run/Stream 语义等价契约测试）
- 受影响文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/v1-acceptance.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 默认策略保持不变（lexical-only），为增量能力扩展，避免现有行为漂移。
