## 1. Runtime Snapshot Config Schema and Validation

- [x] 1.1 在 `runtime/config` 新增 `runtime.state.snapshot.*` 字段与默认值。
- [x] 1.2 在 `runtime/config` 新增 `runtime.session.state.*` 字段与默认值。
- [x] 1.3 实现 strict/compatible、兼容窗口、partial restore 等配置校验（非法值 fail-fast）。
- [x] 1.4 增加配置优先级测试（`env > file > default`）。
- [x] 1.5 增加热更新非法配置回滚测试（原子回滚）。

## 2. Unified Snapshot Manifest and Segment Model

- [x] 2.1 定义统一 manifest 结构（schema version/source/timestamp/segments/digest）。
- [x] 2.2 定义模块分段 payload（runner/session、scheduler/mailbox、composer recovery、memory）。
- [x] 2.3 实现 manifest 序列化/反序列化与校验摘要计算。
- [x] 2.4 增加 schema 校验测试（缺字段、错版本、摘要不一致）。

## 3. Export/Import API and Idempotency

- [x] 3.1 实现统一 snapshot 导出入口（按分段组装，不重写事实源）。
- [x] 3.2 实现统一 snapshot 导入入口（strict|compatible 模式）。
- [x] 3.3 实现导入幂等键与重复导入去膨胀语义。
- [x] 3.4 增加导入导出回环测试（export -> import -> export 稳定）。

## 4. Recovery Integration (Composer/Scheduler)

- [x] 4.1 将 composer recovery 接入 unified snapshot 导入接缝。
- [x] 4.2 将 scheduler/store 恢复路径接入 unified snapshot 分段消费。
- [x] 4.3 实现恢复冲突策略映射（strict reject / compatible bounded restore）。
- [x] 4.4 增加恢复边界测试（冲突 fail-fast、兼容恢复动作可观测）。

## 5. Memory Lifecycle Alignment (A59 Reuse)

- [x] 5.1 实现 memory 分段导入导出与 A59 lifecycle 字段对齐。
- [x] 5.2 保证 memory restore 复用既有 SPI/filesystem 语义，不引入平行事实源。
- [x] 5.3 增加 memory restore 幂等测试（重复导入不膨胀）。
- [x] 5.4 增加 restore 前后检索质量稳定性回归测试。

## 6. Diagnostics and RuntimeRecorder Additive Fields

- [x] 6.1 在 `runtime/diagnostics` 增加 A66 additive 字段：`state_snapshot_version`、`state_restore_action`、`state_restore_conflict_code`、`state_restore_source`。
- [x] 6.2 在 `observability/event.RuntimeRecorder` 接入 A66 字段映射并保持单写幂等。
- [x] 6.3 增加 QueryRuns parser compatibility 测试（additive + nullable + default）。
- [x] 6.4 增加冲突码与恢复动作 taxonomy drift guard。

## 7. Replay Fixture and Drift Taxonomy

- [x] 7.1 在 `tool/diagnosticsreplay` 新增 `state_session_snapshot.v1` fixture schema 与 loader。
- [x] 7.2 实现 drift 分类：`snapshot_schema_drift`、`state_restore_semantic_drift`、`snapshot_compat_window_drift`、`partial_restore_policy_drift`。
- [x] 7.3 增加 mixed-fixture 回放兼容测试（历史 fixtures + A66 fixture）。
- [x] 7.4 增加 deterministic normalization 断言。

## 8. Contract Tests and Integration Matrix

- [x] 8.1 新增统一 snapshot config 校验单测（strict/compatible/窗口边界）。
- [x] 8.2 新增 composer/scheduler 恢复集成测试（Run/Stream 等价）。
- [x] 8.3 新增 memory/file backend parity 集成测试（恢复后语义一致）。
- [x] 8.4 新增导入重复提交幂等与无副作用测试。

## 9. Gate and CI Wiring

- [x] 9.1 新增 `scripts/check-state-snapshot-contract.sh/.ps1`。
- [x] 9.2 将 state snapshot gate 接入 `scripts/check-quality-gate.sh/.ps1`。
- [x] 9.3 在 gate 中实现 `state_control_plane_absent` 断言（禁止托管状态控制面）。
- [x] 9.4 在 gate 中实现 `state_source_of_truth_reuse_required` 断言（不得重写 A59 memory 事实源）。
- [x] 9.5 在 CI 暴露 required-check 候选（`state-snapshot-contract-gate`）。

## 10. Documentation and Validation

- [x] 10.1 更新 `docs/runtime-config-diagnostics.md`（A66 配置与诊断字段）。
- [x] 10.2 更新 `docs/mainline-contract-test-index.md`（A66 fixture + gate 映射）。
- [x] 10.3 更新 `docs/development-roadmap.md`（A66 状态与验收口径）。
- [x] 10.4 更新 `README.md`（统一 snapshot 能力入口说明）。
- [x] 10.5 执行 `go test ./...` 与 `go test -race ./...`。
- [x] 10.6 执行 `golangci-lint run --config .golangci.yml`。
- [x] 10.7 执行 `scripts/check-state-snapshot-contract.sh/.ps1` 与 `scripts/check-quality-gate.sh/.ps1`。
