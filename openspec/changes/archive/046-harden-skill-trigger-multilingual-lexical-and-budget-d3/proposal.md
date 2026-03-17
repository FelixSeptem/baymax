## Why

当前 skill 触发默认依赖 lexical weighted-keyword，但词法分词更偏英文 token，中文与中英混合输入在不启用 embedding 时命中稳定性不足。D2 已提供可选 embedding 增强，但默认路径仍应在多语言场景可用，且需要控制候选规模以减少提示词噪音与行为漂移。

## What Changes

- 在 `skill/loader` 的 lexical 路径增加 `mixed_cjk_en` 分词与匹配能力，覆盖中文与中英混合输入。
- 新增语义候选预算控制 `max_semantic_candidates`，采用 `top-k` 截断策略，默认值为 `3`。
- 在 skill 触发诊断字段中新增 `tokenizer_mode` 与 `candidate_pruned_count`，用于观测分词模式与裁剪行为。
- 保持默认策略为 `lexical_weighted_keywords`，`lexical_plus_embedding` 语义不变，仅在排序后应用统一预算裁剪。
- 强制保持 Run/Stream 在等价输入与配置下的 skill 触发语义等价。
- 配置入口仅通过 JSON/YAML（及其 env 映射），不引入 CLI 新参数。

## Capabilities

### New Capabilities
无。

### Modified Capabilities
- `skill-trigger-scoring`: 扩展 lexical 分词能力为中英混合，新增 top-k 候选预算与对应诊断字段，并保持 Run/Stream 语义等价门禁。
- `runtime-config-and-diagnostics-api`: 新增 skill trigger scoring 的分词模式与候选预算配置、默认值与校验规则，保证热更新语义一致。

## Impact

- 影响代码：
  - `skill/loader/*`（分词、候选排序与裁剪、事件字段）
  - `runtime/config/*`（配置结构、默认值、校验、加载映射）
  - `observability/event` 与 `runtime/diagnostics`（新增字段透传）
- 影响测试：
  - `skill/loader` 词法与预算策略单测
  - `runtime/config` 配置默认值与非法配置校验测试
  - Run/Stream 等价契约回归测试
- 对外影响：
  - 无 breaking API；新增配置与诊断字段均为增量扩展。
