## Why

当前 HITL 能力已覆盖 H2（Action Gate 执行前确认），但尚不支持“运行中请求用户澄清并恢复”的原生生命周期。  
在多 agent 场景下，`clarification-agent` 需要在信息不足时触发用户补充，再继续主流程；目前只能依赖外部编排，缺少统一语义与可观测口径。

## What Changes

- 引入 H3 最小原生语义：`await_user -> resumed -> canceled_by_user`（单进程）。
- 新增 `clarification_request` 结构化事件 payload，用于前端/调用方直接消费。
- 提供 library-first 的 HITL 回调接口（不提供 CLI），支持阻塞等待与恢复输入注入。
- 默认超时策略为 `cancel_by_user`（fail-fast 终止本次 run）。
- 收敛 Run/Stream 的 H3 语义一致性与错误分类行为。
- 增加最小 diagnostics 字段（`await_count`、`resume_count`、`cancel_by_user_count`）。
- 增量扩容多 agent 示例，演示 `clarification-agent` 请求澄清与恢复。

## Capabilities

### Modified Capabilities
- `action-gate-hitl`: 从 H2 扩展到 H3，纳入原生澄清暂停恢复语义（单进程）。
- `action-timeline-events`: 增加 H3 生命周期 reason code 与事件语义。
- `runtime-config-and-diagnostics-api`: 增补 H3 配置与最小运行诊断计数器。
- `tutorial-examples-expansion`: 增加多 agent clarification 示例要求。

## Impact

- 代码范围：`core/types`、`core/runner`、`runtime/config`、`runtime/diagnostics`、`observability/event`、`examples/07` 或 `examples/08`。
- 测试范围：Run/Stream H3 契约测试（await/resume/cancel/timeout）+ race 基线。
- 文档范围：`README.md`、`docs/development-roadmap.md`、`docs/runtime-config-diagnostics.md`、`docs/v1-acceptance.md`、`docs/examples-expansion-plan.md`。
- 兼容性边界：本期不做持久化恢复；仅保证单进程生命周期一致性。
