## Why

A60 正在实施后，runtime 的预算准入与降级语义会更稳定，但“如何顺滑接入 OTel 采集”仍缺少统一 contract。当前仓库已有 `observability.export` 基线与轻量 `trace.Manager`，但跨后端 collector 的 span/attribute 映射、导出状态字段、回放夹具与质量门禁尚未一次性冻结，导致接入成本和回归成本偏高。  

A61 采用双目标收口：
- 目标一（主落点）：OTel 采集接入做成可配置、可观测、可回放、可阻断的标准能力；
- 目标二（原目标完整兼顾）：补齐 agent eval 互操作与 `local|distributed` 执行治理，固定质量回归口径并避免后续再开平行提案。

## What Changes

- 新增 A61 主合同：`runtime-otel-tracing-and-agent-eval-interoperability-contract`。
- 目标一收口：冻结 OTel 采集互操作语义，确保 runtime 到 collector 的接线“顺滑可复用”。
- tracing 语义冻结：统一 run/model/tool/mcp/memory/hitl span 拓扑与 canonical attribute 映射。
- tracing 配置冻结：新增 `runtime.observability.tracing.otel.*` 配置域并遵守 `env > file > default`、fail-fast、热更新原子回滚。
- tracing 诊断冻结：新增 QueryRuns additive 字段 `trace_export_status`、`trace_schema_version`。
- 目标二收口（eval）：新增 `runtime.eval.agent.*` 与 `runtime.eval.execution.*`（`mode=local|distributed`、`shard`、`retry`、`resume`、`aggregation`）。
- 目标二诊断冻结：新增 QueryRuns additive 字段 `eval_suite_id`、`eval_summary`、`eval_execution_mode`、`eval_job_id`、`eval_shard_total`、`eval_resume_count`。
- 回放与漂移冻结：新增 `otel_semconv.v1`、`agent_eval.v1`、`agent_eval_distributed.v1` 及 drift taxonomy：
  - `otel_attr_mapping_drift`
  - `span_topology_drift`
  - `eval_metric_drift`
  - `eval_aggregation_drift`
  - `eval_shard_resume_drift`
- 质量门禁新增：`check-agent-eval-and-tracing-interop-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- CI required-check 候选新增：`agent-eval-tracing-interop-gate`。
- 边界断言冻结：gate 必须包含 `control_plane_absent`（distributed execution 仅库内嵌入式治理，不新增托管控制面）。

## Capabilities

### New Capabilities
- `runtime-otel-tracing-and-agent-eval-interoperability-contract`: 冻结 OTel 采集互操作与 agent eval 执行治理的一体化 contract。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 tracing/eval 配置字段与 QueryRuns additive 字段。
- `observability-export-and-diagnostics-bundle-contract`: 增加 OTel tracing collector 互操作与 tracing 导出语义约束。
- `diagnostics-replay-tooling`: 增加 otel/eval fixture 校验与 drift 分类。
- `go-quality-gate`: 增加 tracing+eval contract gate 与 required-check 候选。

## Impact

- 代码：
  - `runtime/config`（tracing/eval 配置解析、校验、热更新回滚）
  - `observability/trace`、`observability/event`（span/attribute 映射、export status 聚合、RuntimeRecorder 单写映射）
  - `runtime/diagnostics`（additive 字段）
  - `tool/diagnosticsreplay`（fixture parser/normalization/drift 分类）
  - `integration/*`（OTel backend 兼容冒烟、eval local/distributed 聚合一致性）
  - `scripts/check-agent-eval-and-tracing-interop-contract.*` 与 `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`、`observability/README.md`
- 兼容性与边界：
  - 对外 API 不做 breaking 变更，新增字段遵循 `additive + nullable + default`。
  - Run/Stream tracing 与 eval 输出语义保持等价。
  - 不引入托管评测控制面、远程调度服务或平台化 UI/RBAC/多租户运维面板。
