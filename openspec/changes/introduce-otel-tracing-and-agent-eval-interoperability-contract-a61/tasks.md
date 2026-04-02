## 1. OTel Tracing Config Schema and Validation

- [ ] 1.1 在 `runtime/config` 新增 `runtime.observability.tracing.otel.*` 配置结构、默认值与归一化逻辑。
- [ ] 1.2 新增 tracing 配置校验（protocol/endpoint/sample_ratio/timeout/resource attrs）并遵守 fail-fast。
- [ ] 1.3 打通 tracing 配置热更新失败原子回滚。
- [ ] 1.4 补充配置单测（`env > file > default`、非法值、热更新回滚、fallback 解析）。

## 2. Canonical OTel SemConv Mapping

- [ ] 2.1 在 `observability/trace` 冻结 `otel_semconv.v1` 的 span 拓扑映射（run/model/tool/mcp/memory/hitl）。
- [ ] 2.2 冻结 canonical attribute 键与版本化映射表（含 schema version 透传）。
- [ ] 2.3 补充 Run/Stream tracing 语义等价测试（允许非语义顺序差异）。

## 3. Collector Interoperability and Export Runtime

- [ ] 3.1 扩展 tracing export runtime，保证 OTLP collector 互操作接线可配置且 deterministic。
- [ ] 3.2 对齐 tracing 导出失败分类与 on-error 策略（fail-fast/degrade）语义。
- [ ] 3.3 新增至少两类 OTel backend 兼容冒烟（本地 exporter + 远端 collector）。
- [ ] 3.4 补充 collector 不可达/超时/鉴权失败场景的集成测试。

## 4. Diagnostics Additive Fields and Recorder Wiring

- [ ] 4.1 在 `runtime/diagnostics` 增加 `trace_export_status`、`trace_schema_version` 字段。
- [ ] 4.2 在 `runtime/diagnostics` 增加 `eval_suite_id`、`eval_summary`、`eval_execution_mode`、`eval_job_id`、`eval_shard_total`、`eval_resume_count` 字段。
- [ ] 4.3 在 `observability/event.RuntimeRecorder` 接入 tracing/eval 字段映射并保持单写幂等。
- [ ] 4.4 增加 QueryRuns 兼容测试（additive + nullable + default + bounded-cardinality）。

## 5. Agent Eval Baseline Contract

- [ ] 5.1 在 `runtime/config` 新增 `runtime.eval.agent.*` 配置域（任务成功、工具正确性、拒绝/拦截准确率、cost-latency 约束）。
- [ ] 5.2 实现 eval 基线指标汇总与 `eval_summary` 生成逻辑。
- [ ] 5.3 增加 eval 单测（指标计算、阈值边界、异常输入处理）。

## 6. Embedded Distributed Eval Execution Governance

- [ ] 6.1 在 `runtime/config` 新增 `runtime.eval.execution.*` 配置域（`mode=local|distributed`、`shard`、`retry`、`resume`、`aggregation`）。
- [ ] 6.2 实现 distributed 执行治理（分片、失败重试、断点续跑、结果幂等聚合）。
- [ ] 6.3 增加 local/distributed 聚合等价测试与 resume 幂等测试。
- [ ] 6.4 增加边界断言测试，确保 distributed 执行不引入托管控制面依赖。

## 7. Replay Fixtures and Drift Taxonomy

- [ ] 7.1 在 `tool/diagnosticsreplay` 新增 `otel_semconv.v1` fixture schema、loader 与 normalization。
- [ ] 7.2 新增 `agent_eval.v1` 与 `agent_eval_distributed.v1` fixture schema、loader 与 normalization。
- [ ] 7.3 新增 drift 分类断言：`otel_attr_mapping_drift`、`span_topology_drift`、`eval_metric_drift`、`eval_aggregation_drift`、`eval_shard_resume_drift`。
- [ ] 7.4 增加 mixed-fixture backward compatibility 回放测试（历史 fixtures + A61 fixtures）。

## 8. Contract Gate and CI Wiring

- [ ] 8.1 新增 `scripts/check-agent-eval-and-tracing-interop-contract.sh/.ps1`。
- [ ] 8.2 将 A61 gate 接入 `scripts/check-quality-gate.sh/.ps1` 并保持 shell/PowerShell parity。
- [ ] 8.3 在 CI 暴露独立 required-check 候选 `agent-eval-tracing-interop-gate`。
- [ ] 8.4 在 gate 中实现并验证 `control_plane_absent` 断言。

## 9. Documentation Sync

- [ ] 9.1 更新 `docs/runtime-config-diagnostics.md`（OTel tracing + eval + execution 配置与字段说明）。
- [ ] 9.2 更新 `docs/mainline-contract-test-index.md`（A61 fixtures + gate 索引）。
- [ ] 9.3 更新 `docs/development-roadmap.md`（A61 状态与验收推进同步）。
- [ ] 9.4 更新 `README.md` 与 `observability/README.md`（OTel 接入入口与注意事项）。

## 10. Validation

- [ ] 10.1 执行 `go test ./observability/trace ./observability/event -count=1`。
- [ ] 10.2 执行 `go test ./runtime/config ./runtime/diagnostics -count=1`。
- [ ] 10.3 执行 `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContract.*(Otel|Eval|A61)' -count=1`。
- [ ] 10.4 执行 `go test -race ./...`。
- [ ] 10.5 执行 `golangci-lint run --config .golangci.yml`。
- [ ] 10.6 执行 `pwsh -File scripts/check-agent-eval-and-tracing-interop-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
