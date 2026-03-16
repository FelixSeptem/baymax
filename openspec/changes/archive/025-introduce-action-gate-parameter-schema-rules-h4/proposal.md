## Why

当前 Action Gate H2 仅支持 `tool name + keyword` 粗粒度规则，无法稳定识别“参数值本身”带来的高风险动作（如 `force=true`、危险路径、越权标志）。随着 H3 已落地，下一步需要在不引入外部策略引擎的前提下补齐参数级风险判定，提升安全与可控性。

## What Changes

- 在 `action_gate` 配置中引入参数规则引擎，支持按规则声明 `path + operator + expected`。
- 支持复合条件组合（AND/OR），可表达多条件联动命中。
- 支持操作符集合（首期覆盖精确匹配、集合匹配、字符串匹配、正则、数值比较、存在性判断）。
- 规则可独立配置动作决策（`allow/require_confirm/deny`），未配置时继承全局 `action_gate.policy`。
- 明确优先级：参数规则命中优先于 `decision_by_tool/decision_by_keyword`。
- Run/Stream 路径保持语义一致，补齐复合条件与优先级契约测试。
- 新增最小诊断字段与 timeline reason code（参数规则命中相关）。
- 增量扩展示例，提供参数规则最小演示。
- 同步更新 README、runtime 配置文档、验收文档、roadmap、主干契约测试索引。

## Capabilities

### New Capabilities
- `action-gate-parameter-rules`: Action Gate 参数级规则与复合条件判定能力（本地配置驱动，非外部策略引擎）。

### Modified Capabilities
- `action-gate-hitl`: 扩展 H2 行为到参数级规则判定与规则动作继承语义。
- `action-timeline-events`: 增加参数规则命中相关 reason code 语义。
- `runtime-config-and-diagnostics-api`: 增加参数规则配置字段与最小诊断字段。
- `tutorial-examples-expansion`: 增量示例覆盖参数规则命中与确认路径。

## Impact

- 代码范围：`core/runner`、`core/types`、`runtime/config`、`runtime/diagnostics`、`observability/event`、`examples/*`。
- 测试范围：Action Gate 参数规则操作符、复合条件、优先级、默认继承、Run/Stream 等价契约测试。
- 文档范围：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`。
- 非目标：不做参数 schema 自动推断；不接入 OPA/外部策略引擎；不改变现有 H3 生命周期语义。
