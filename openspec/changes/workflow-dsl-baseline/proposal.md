## Why

当前运行时具备单 run 循环与多代理示例，但缺少可声明、可校验、可恢复的工作流编排契约。业务侧需要手写流程控制（分支、依赖、重试、超时），导致流程语义不一致，也难以进行统一回放与治理。

引入 Workflow DSL baseline，可以先收敛“确定性执行语义”和“最小恢复点能力”，为后续 Teams 与 A2A 的组合编排提供稳定底座。

## What Changes

- 新增 Workflow DSL 基线能力：定义 YAML/JSON 工作流最小语法（step/depends_on/condition/retry/timeout）。
- 新增 workflow 计划校验与执行基线：DAG 校验、确定性调度、分支与重试语义。
- 新增最小 checkpoint/resume 契约，用于中断恢复与诊断回放对齐。
- 扩展 timeline/diagnostics 字段，建立 `workflow_id`、`step_id` 与 `run_id/session_id` 的关联语义。
- 保持 library-first，workflow 作为独立编排模块，不直接改写 `core/runner` 主状态机语义。

## Capabilities

### New Capabilities
- `workflow-deterministic-dsl`: 提供可声明、可校验、可回放的工作流 DSL 与确定性执行语义。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 workflow 配置与 run 级流程诊断字段契约。
- `action-timeline-events`: 增加 workflow 执行链路元数据与 reason 语义。
- `runtime-module-boundaries`: 增加 workflow 引擎与 runner 之间的边界约束。

## Impact

- 影响代码：
  - 新增 `workflow` 编排模块（路径以实施稿为准）
  - `core/types`（workflow 相关 DTO/元数据）
  - `runtime/config`、`runtime/diagnostics`、`observability/event`
- 影响测试：
  - DSL schema 与 DAG 校验测试
  - checkpoint/resume 回归测试
  - Run/Stream 语义等价测试
- 影响文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/runtime-module-boundaries.md`
  - `docs/diagnostics-replay.md`
  - `docs/multi-agent-identifier-model.md`
