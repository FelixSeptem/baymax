## Context

当前 H2 Action Gate 仅支持 `tool name + keyword` 风险判定，命中粒度偏粗，无法稳定覆盖“参数值触发风险”的场景。项目已具备 H2/H3 基础能力（gate 决策、resolver、timeline、diagnostics、Run/Stream 契约测试），因此可以在既有框架上补一层参数规则引擎，而无需引入外部策略系统。

约束：
- 保持 library-first，不新增 CLI 依赖。
- 配置继续走 `runtime/config`（`env > file > default`，fail-fast）。
- Run/Stream 语义一致，不能破坏现有 H2/H3 行为。
- 变更域收敛：规则引擎仅处理输入参数判定，不做 schema 自动推断。

## Goals / Non-Goals

**Goals:**
- 为 Action Gate 增加参数级风险规则能力，支持 `path + operator + expected`。
- 支持复合条件（AND/OR）表达，满足多条件联动判定。
- 规则允许独立 action（`allow/require_confirm/deny`），缺省继承全局 policy。
- 明确优先级：参数规则 > `decision_by_tool`/`decision_by_keyword` > 现有默认路径。
- 新增最小可观测增量（timeline reason code + diagnostics counters）。
- 补齐复合条件与优先级相关契约测试，并保持 Run/Stream 等价。

**Non-Goals:**
- 不引入 OPA 或其他外部策略引擎。
- 不做参数 schema 自动推断或自动学习规则。
- 不变更 H3 clarification 生命周期语义。

## Decisions

### 1) 规则模型采用“可组合表达式树”
- 决策：在配置层定义 `rule` + `condition` 结构；`condition` 支持叶子条件与组合节点（AND/OR）。
- 原因：相比平铺条件，表达式树可覆盖复合条件，且后续扩展 NOT/权重时结构稳定。
- 备选：仅支持平铺 AND。缺点是表达能力不足，后续扩展会引发配置破坏性迁移。

### 2) 操作符首期一次到位
- 决策：首期支持常用操作符（eq/ne/contains/regex/in/not_in/gt/gte/lt/lte/exists）。
- 原因：用户已确认“同时支持操作符”，并且该集合足够覆盖参数风控主场景。
- 备选：先做最小 eq/contains。缺点是二次提案会引入额外迁移与测试成本。

### 3) 决策继承机制
- 决策：规则可显式 `action`；缺省继承 `action_gate.policy`。
- 原因：减少配置冗余，同时允许高风险规则单独收紧。
- 备选：规则 action 必填。缺点是配置体积大、可维护性差。

### 4) 优先级固定且可测试
- 决策：判定顺序固定为：
  1) 参数规则命中
  2) decision_by_tool / decision_by_keyword
  3) tool_names / keywords 默认 policy
  4) allow
- 原因：避免策略互相覆盖时语义歧义。
- 备选：按“首次命中即返回（配置顺序）”。缺点是稳定性和可维护性差。

### 5) 可观测最小增量
- 决策：timeline 新增 `gate.rule_match` reason code；diagnostics 新增最小字段（如 `gate_rule_hit_count`、`gate_rule_last_id`）。
- 原因：满足定位需求，同时控制变更面。
- 备选：完整规则命中明细落库。缺点是隐私与存储成本上升，且当前非必要。

### 6) 测试策略按“规则语义 + 主干契约”双层覆盖
- 决策：
  - 规则语义测试：操作符、组合、优先级、继承、非法配置 fail-fast。
  - 主干契约测试：Run/Stream 等价、timeout/deny 行为、观测字段稳定。
- 原因：复合条件引入的风险主要在判定语义，必须单元+契约双重约束。

## Risks / Trade-offs

- [Risk] 规则配置复杂度上升，误配置概率增加  
  -> Mitigation: 严格 schema 校验 + 启动/热更新 fail-fast + 文档示例。

- [Risk] 复合条件与多操作符带来执行开销  
  -> Mitigation: 单次 tool-call 判定保持 O(rules) 且短路执行；首期不引入昂贵表达式特性。

- [Risk] Run/Stream 路径出现行为漂移  
  -> Mitigation: 增加等价契约测试并纳入 mainline contract index。

## Migration Plan

1. 扩展 `runtime/config`：新增参数规则字段、默认值与校验。
2. 扩展 `core/types`：规则与条件 DTO、操作符枚举。
3. 在 `core/runner` 接入参数规则判定与优先级流程（Run/Stream 同步改造）。
4. 扩展 `observability/event` 与 `runtime/diagnostics` 最小增量字段。
5. 增量更新一个示例演示参数规则命中路径。
6. 补齐测试：语义单测 + Run/Stream 契约 + 配置 fail-fast。
7. 同步文档与 `docs/mainline-contract-test-index.md`。

## Open Questions

- 首期是否需要支持大小写敏感开关（当前建议统一大小写不敏感，仅 regex 按表达式本身控制）。
- `exists` 对 `null` 的语义：仅“路径存在”还是“存在且非空”（当前建议首期定义为“路径存在”）。
