## Why

当前 D3 已实现 `mixed_cjk_en` 与固定 `top-k` 预算，但固定候选数在“强头部”与“高歧义”两类输入上都不够理想：前者容易引入噪音候选，后者又可能召回不足。需要引入可解释、可配置且确定性的自适应预算策略，在保持默认可用性的同时提升触发稳定性。

## What Changes

- 在 skill trigger scoring 中引入预算模式 `fixed|adaptive`，默认改为 `adaptive`。
- 增加自适应预算参数：`min_k`、`max_k`、`min_score_margin`（默认 `0.08`），并定义确定性预算决策规则。
- 保留并兼容现有 `max_semantic_candidates` 与 explicit 命中旁路语义。
- 扩展 skill 观测字段：`budget_mode`、`selected_semantic_count`、`score_margin_top1_top2`、`budget_decision_reason`。
- 保持 Run/Stream 在等价输入与配置下的 skill 触发语义等价门禁。
- 配置入口仅通过 JSON/YAML（及 env 映射）接入，不新增 CLI 参数。

## Capabilities

### New Capabilities
无。

### Modified Capabilities
- `skill-trigger-scoring`: 将语义预算从固定 top-k 升级为 `fixed|adaptive` 双模式（默认 adaptive），并补齐自适应预算可观测字段与等价性契约。
- `runtime-config-and-diagnostics-api`: 扩展 skill trigger scoring 预算模式与参数配置、默认值与 fail-fast 校验规则，并扩展诊断字段契约。

## Impact

- 影响代码：
  - `skill/loader/*`（预算决策、裁剪逻辑、payload 字段）
  - `runtime/config/*`（预算配置结构、默认值、校验、热更新映射）
  - `observability/event` 与 `runtime/diagnostics`（新增字段透传）
- 影响测试：
  - loader 自适应预算与 fixed 预算行为测试
  - runtime config 默认值/env 覆盖/非法配置回滚测试
  - Run/Stream 等价契约测试
- 对外影响：
  - 无 breaking API；新增配置与诊断字段为增量扩展。
