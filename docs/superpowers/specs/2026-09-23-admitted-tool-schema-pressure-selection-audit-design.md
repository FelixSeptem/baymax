# Admitted Tool Schema Pressure and Selection Audit Design

## Goal

建立一个仅离线运行的、版本化的 `tool_schema_pressure_selection_audit.v1` 契约，用同一份已准入工具快照同时衡量 schema/context 压力和工具选择质量，并比较多个确定性候选策略，为未来是否值得实现按需 Tool Schema 投影提供证据。

## Boundaries

审计只接收宿主已经准入的 snapshot DTO，不调用 Tool、MCP、Provider、tokenizer、模型或网络。它不改变 `ModelRequest`、provider request projection、tool registry、allowlist、sandbox、ReAct、scheduler、runtime config、RuntimeRecorder 或 examples/agent-modes。

## Evaluation tracks

- Synthetic fixture 是强制 gate，声明 expected、allowed、forbidden、fallback 工具集合，并计算 precision、recall、F1、expected-hit、forbidden-hit 与 fallback coverage。
- Eval corpus/Badcase 是可选 advisory，只提供 bounded coverage、label completeness 和 aggregate trend；缺失或不足不会使 synthetic gate 失败。
- Pressure 指标包括工具数、canonical schema bytes、`utf8_bytes_div4_v1` estimate、per-tool bounded summary 与压力等级。
- 策略集合固定为 `full_admitted_set`、`capability_filtered_set`、`priority_top_k`、`source_partitioned_set` 和 bounded fixture-declared strategy。

只有 baseline pressure 超预算、baseline quality 退化、且至少一个策略在两个维度都确定性改善时才输出 `projection_candidate`。该结论是离线 advisory，不输出可直接接入 runtime 的 selector 或 tool subset。

## Evidence and rollback

所有结果必须可由 fixture/replay 重算；输出只保留 digest、长度、计数、有限标签、指标和 reason，不保留 raw schema、prompt、model output 或 tool result body。若 pressure、quality、strategy 或 Run/Stream parity drift，gate fail-fast，运行时保持现状，无迁移和回滚动作。
