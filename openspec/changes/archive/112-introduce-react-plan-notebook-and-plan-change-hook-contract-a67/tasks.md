## 1. Runtime Config Contract (A67)

- [x] 1.1 在 `runtime/config` 增加 `runtime.react.plan_notebook.*` 与 `runtime.react.plan_change_hook.*` 字段、默认值与 env 映射。
- [x] 1.2 增加 A67 启动校验（枚举、边界、组合合法性）并保持 fail-fast。
- [x] 1.3 增加热更新非法配置原子回滚测试（file/env 组合）。

## 2. ReAct Plan Notebook Core

- [x] 2.1 定义 notebook 数据模型（`plan_id/version/status/history/action/reason`）与 canonical action taxonomy（`create|revise|complete|recover`）。
- [x] 2.2 在 `core/runner` ReAct loop 接入 notebook 生命周期推进，保持 step-boundary deterministic。
- [x] 2.3 增加 notebook 终态冻结与版本单调递增保护逻辑。

## 3. Plan-Change Hook Execution

- [x] 3.1 在计划变更边界接入 `before_plan_change` / `after_plan_change` 钩子并定义 payload。
- [x] 3.2 实现 `fail_fast|degrade` 失败策略与超时处理，不改变 A56 终止 taxonomy 主语义。
- [x] 3.3 增加 hook 顺序、错误冒泡与上下文透传的单测与集成测试。

## 4. Recovery and Idempotency

- [x] 4.1 在现有 session/recovery 接缝接入 notebook 恢复，不新增平行事实源。
- [x] 4.2 增加重复 recover/revise 重放幂等测试，确保计数与终态不膨胀。
- [x] 4.3 增加 Run/Stream + memory/file 后端下的恢复语义一致性测试。

## 5. Diagnostics and Recorder Mapping

- [x] 5.1 在 `runtime/diagnostics` 增加 A67 additive 字段：`react_plan_id`、`react_plan_version`、`react_plan_change_total`、`react_plan_last_action`、`react_plan_change_reason`、`react_plan_recover_count`、`react_plan_hook_status`。
- [x] 5.2 在 `observability/event.RuntimeRecorder` 接入 A67 字段映射并保持单写幂等。
- [x] 5.3 增加 QueryRuns 序列化兼容测试（旧字段解析不回归）。

## 6. Diagnostics Replay and Drift Taxonomy

- [x] 6.1 新增 `react_plan_notebook.v1` fixture（覆盖 create/revise/complete/recover 与 hook fail_fast/degrade）。
- [x] 6.2 在 `tool/diagnosticsreplay` 增加 A67 drift 分类：`react_plan_version_drift`、`react_plan_change_reason_drift`、`react_plan_hook_semantic_drift`、`react_plan_recover_drift`。
- [x] 6.3 增加 mixed-fixture 兼容测试（历史 fixtures + A67 fixture）。

## 7. Contract and Integration Tests

- [x] 7.1 增加 A67 核心合同集成用例：计划生命周期与计划变更 hook。
- [x] 7.2 增加 ReAct Run/Stream parity 用例（含 plan revision/recover 场景）。
- [x] 7.3 增加 A58/A57 边界回归测试，确保 A67 不绕过 precedence 与安全链路。

## 8. Gate and CI Wiring

- [x] 8.1 新增 `scripts/check-react-plan-notebook-contract.sh` 与 `scripts/check-react-plan-notebook-contract.ps1`。
- [x] 8.2 将 A67 gate 接入 `scripts/check-quality-gate.sh/.ps1`，保持 shell/PowerShell fail-fast 语义等价。
- [x] 8.3 在 gate 中实现 impacted-contract suites 校验（按 A67 模块触发对应主干 suites）。
- [x] 8.4 在 CI 中暴露独立 required-check 候选（`react-plan-notebook-gate`）。

## 9. Documentation Sync

- [x] 9.1 更新 `docs/runtime-config-diagnostics.md`（A67 配置字段、默认值、失败语义、诊断字段）。
- [x] 9.2 更新 `docs/mainline-contract-test-index.md`（A67 fixture + gate 映射）。
- [x] 9.3 更新 `docs/development-roadmap.md`（A67 状态与验收口径）。
- [x] 9.4 更新 `README.md`（里程碑快照与能力状态对齐）。

## 10. Validation and Exit

- [x] 10.1 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 10.2 执行合同门禁：`check-react-plan-notebook-contract.*`、`check-quality-gate.*`、`check-docs-consistency.*`。
  - 已执行：`pwsh -File scripts/check-react-plan-notebook-contract.ps1`（通过）。
  - 已执行：`pwsh -File scripts/check-docs-consistency.ps1`（通过）。
  - 已执行：`pwsh -File scripts/check-quality-gate.ps1`（通过，`BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=900`、`BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=600`、`BAYMAX_QUALITY_GATE_PARALLELISM=4`，总耗时约 646s）。
- [x] 10.3 记录未执行项与风险说明，确保提案可审查、可回滚、可归档。
  - 未执行项：无。
  - 风险说明：修复门禁耗时卡点时，收敛了全仓库扫描范围（跳过缓存/临时目录并改为 tracked file 优先）；已通过 quality gate 与 A67 专项门禁回归。
