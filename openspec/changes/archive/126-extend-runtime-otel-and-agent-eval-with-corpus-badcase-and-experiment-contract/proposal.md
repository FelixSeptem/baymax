## Why

现有 OTel 与 agent eval 合同已经能够记录运行关联、生成最小指标并在 local/distributed 模式下确定性聚合，但仍缺少可版本化的评测语料、Badcase 复现和实验对比语义。结果是质量问题可以被观测，却难以稳定复盘、比较和形成可审阅的改进建议。现在补齐这一层，可以在不引入托管评测控制面的前提下，把现有 diagnostics/replay/trace 基线闭合为可回归的质量反馈闭环。

## What Changes

- 增加 versioned evaluation corpus 合同，固定 scenario、input、tool、policy、runtime snapshot 引用及兼容性语义。
- 增加 Badcase 合同，记录分类、关联的 Run/Step/Trace/Artifact/Checkpoint、复现输入和确定性复现状态。
- 增加 experiment comparison 合同，支持跨 corpus 版本、运行批次和 local/distributed 结果的确定性、幂等比较。
- 增加 metric/rubric declaration 与 drift 分类，确保指标定义变更可检测且不会静默改变历史结果含义。
- 增加人工审批 feedback recommendation 投影；建议只读、可审计，不自动修改 prompt、tool、policy 或 runtime configuration。
- 扩展 diagnostics replay、OTel/RuntimeRecorder additive 字段、示例 fixture 和 quality gate，保持现有 v1 数据兼容。
- 保持 Run/Stream 语义等价、`env > file > default` 配置优先级和 fail-fast/原子回滚边界。
- 不引入新的观测数据面、远程调度器、托管评测控制面、评测 UI、外部事件/存储服务或平台化多租户能力。

## Capabilities

### New Capabilities

- `evaluation-corpus-badcase-and-experiment-contract`: 版本化评测语料、Badcase 复现、指标/rubric 声明、实验比较和人工反馈建议的 transport/storage-neutral 合同。

### Modified Capabilities

- `runtime-otel-tracing-and-agent-eval-interoperability-contract`: 增加 corpus、Badcase、experiment 和 feedback 的 additive correlation、版本漂移、聚合幂等与 local/distributed parity 要求。
- `diagnostics-replay-tooling`: 增加上述对象的 replay fixture、确定性 drift 分类和历史 fixture 兼容要求。
- `observability-export-and-diagnostics-bundle-contract`: 增加可选、可脱敏的评测关联字段导出约束，并继续通过 `RuntimeRecorder` 单写入口。

## Impact

- `runtime/config`：增加评测语料、比较和反馈相关的可选配置/默认值/严格校验，维持 `env > file > default` 与非法热更新原子回滚。
- `observability/event`、`observability/trace`：扩展 additive、nullable、bounded 的 corpus/Badcase/experiment correlation 字段。
- `tool/diagnosticsreplay`、`integration/`：新增成功、版本不兼容、Badcase 不可复现、metric/rubric drift、聚合冲突、审批缺失和 Run/Stream parity 用例。
- `scripts/`：扩展 agent eval/tracing contract gate 及 quality gate、docs consistency 检查。
- `examples/agent-modes/tracing-eval-smoke` 与相关 README/MATRIX：文档先行并补充从 protocol correlation 到 evaluation result 的可执行 smoke 断言。
- 既有 OTel、eval、diagnostics、snapshot、policy、memory、budget 合同保持 additive + nullable + default；不改变现有调用方默认行为。

## Example Impact Assessment

修改示例

扩展 `tracing-eval-smoke` 的文档、fixture 和运行断言，展示 corpus 版本、Badcase 关联、实验比较以及人工反馈建议的只读投影。
