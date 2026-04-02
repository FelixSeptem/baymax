## 1. Runtime Budget Config Schema and Validation

- [x] 1.1 在 `runtime/config` 新增 `runtime.admission.budget.cost.*` 字段与默认值。
- [x] 1.2 在 `runtime/config` 新增 `runtime.admission.budget.latency.*` 字段与默认值。
- [x] 1.3 在 `runtime/config` 新增 `runtime.admission.degrade_policy.*` 字段与默认值。
- [x] 1.4 实现预算阈值与降级策略校验（非法值 fail-fast，热更新回滚）。
- [x] 1.5 增加配置单测（`env > file > default`、非法阈值、非法策略、热更新回滚）。

## 2. Unified Budget Snapshot Construction

- [x] 2.1 实现统一 `budget_snapshot` 结构，覆盖 token/tool/sandbox/memory 成本与时延分解。
- [x] 2.2 固化预算估算输入来源与 deterministic 聚合顺序。
- [x] 2.3 增加快照构建单测（等效输入 determinism、边界值、空输入）。

## 3. Admission Decision Mapping

- [x] 3.1 实现 `allow|degrade|deny` 两阶段预算判定逻辑（硬阈值优先）。
- [x] 3.2 将预算判定接入 readiness admission guard，不重定义 A58 policy 字段。
- [x] 3.3 固化 deny side-effect-free 语义（无 scheduler/mailbox/task 变更）。
- [x] 3.4 增加 Run/Stream 等价测试（同输入同 `budget_decision`）。

## 4. Degrade Policy Execution

- [x] 4.1 实现 `runtime.admission.degrade_policy.*` 的 canonical 动作选择逻辑。
- [x] 4.2 在 admission 输出中增加 `degrade_action` 透传与可观测映射。
- [x] 4.3 增加降级策略单测（动作顺序、冲突策略、非法策略）。

## 5. Diagnostics and RuntimeRecorder Additive Fields

- [x] 5.1 在 `runtime/diagnostics` 增加 `budget_snapshot`、`budget_decision`、`degrade_action` 字段。
- [x] 5.2 在 `observability/event.RuntimeRecorder` 接入 budget-admission 字段映射并保持单写幂等。
- [x] 5.3 增加 QueryRuns 兼容测试（additive + nullable + default）。

## 6. Replay Fixture and Drift Taxonomy

- [x] 6.1 在 `tool/diagnosticsreplay` 新增 `budget_admission.v1` fixture schema、loader 与 normalization。
- [x] 6.2 新增 drift 分类断言：`budget_threshold_drift`、`admission_decision_drift`、`degrade_policy_drift`。
- [x] 6.3 增加 mixed-fixture 回放兼容测试（历史 fixtures + `budget_admission.v1`）。

## 7. Contract Tests and Integration Matrix

- [x] 7.1 新增预算阈值判定单测（cost/latency 双阈值）。
- [x] 7.2 新增 admission 集成测试（token/tool/sandbox/memory 混合成本下判定一致）。
- [x] 7.3 新增压测用例（P95/P99 触发阈值下 degrade 与 fail-fast 稳定性）。

## 8. Gate and CI Wiring

- [x] 8.1 新增 `scripts/check-runtime-budget-admission-contract.sh/.ps1`。
- [x] 8.2 将 budget-admission contract gate 接入 `scripts/check-quality-gate.*`。
- [x] 8.3 在 CI 暴露独立 required-check 候选（`runtime-budget-admission-gate`）。
- [x] 8.4 在 gate 中实现并验证 `budget_control_plane_absent` 与 `budget_field_reuse_required` 断言。
- [x] 8.5 验证 shell/PowerShell gate 失败传播语义一致。

## 9. Documentation Sync

- [x] 9.1 更新 `docs/runtime-config-diagnostics.md`（budget/admission 字段、默认值与阈值示例）。
- [x] 9.2 更新 `docs/mainline-contract-test-index.md`（`budget_admission.v1` 与 gate 索引）。
- [x] 9.3 更新 `docs/development-roadmap.md` 与 `README.md`（A60 状态和入口说明）。
- [x] 9.4 更新 `runtime/config/README.md`、`runtime/diagnostics/README.md` 的预算 admission 合同说明。
- [x] 9.5 在 roadmap 的跨提案联动段落补充 A60 同域收口规则（预算 admission 增量仅在 A60 内吸收）。

## 10. Validation

- [x] 10.1 执行 `go test ./runtime/config ./runtime/config/readiness -count=1`。
- [x] 10.2 执行 `go test ./runtime/diagnostics ./observability/event -count=1`。
- [x] 10.3 执行 `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContract.*Budget' -count=1`。
- [x] 10.4 执行 `go test -race ./...`。
- [x] 10.5 执行 `golangci-lint run --config .golangci.yml`。
- [x] 10.6 执行 `pwsh -File scripts/check-runtime-budget-admission-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
