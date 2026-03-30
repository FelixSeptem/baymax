## 1. Rollout Policy and Config Surface

- [x] 1.1 在 `runtime/config` 增加 `security.sandbox.rollout.*` 与容量治理配置结构（phase/traffic/health-budget/capacity/freeze）。
- [x] 1.2 实现 rollout phase 合法迁移校验与 startup fail-fast 断言。
- [x] 1.3 实现 rollout 配置热更新非法回滚测试，确保原子快照不被污染。
- [x] 1.4 补齐配置文档索引与默认值说明（`env > file > default` 语义保持一致）。

## 2. Readiness and Admission Integration

- [x] 2.1 在 `runtime/config/readiness` 接入 rollout/freeze/capacity canonical findings 与 strict/non-strict 映射。
- [x] 2.2 在 admission guard 接入 frozen fail-fast 语义并验证 deny side-effect free。
- [x] 2.3 在 admission guard 接入 capacity action（`allow|throttle|deny`）与 degraded policy 映射。
- [x] 2.4 增加 Run/Stream rollout admission parity 测试覆盖。

## 3. Runtime Governance Evaluation

- [x] 3.1 在 runtime 治理路径实现健康预算评估（launch-failure/timeout/violation/p95/admission-deny）。
- [x] 3.2 实现 budget breach 自动冻结与 freeze reason taxonomy 映射。
- [x] 3.3 实现 cooldown + manual unfreeze token 受控解冻语义。
- [x] 3.4 增加容量预算下 queue/inflight 计算与 deterministic capacity action 断言。

## 4. Observability and Replay Contract

- [x] 4.1 在 action timeline 增加 rollout-governance canonical reasons（phase/freeze/health/capacity）。
- [x] 4.2 在 `runtime/diagnostics` 与 `RuntimeRecorder` 增加 rollout/capacity/freeze additive 字段并保持 single-writer idempotency。
- [x] 4.3 在 `tool/diagnosticsreplay` 增加 `a52.v1` fixture parser 与 drift 分类（phase/health/capacity/freeze）。
- [x] 4.4 增加 A51 + A52 混合 fixture 回放兼容测试，防止 parser regression。

## 5. Quality Gate and Performance Baseline

- [x] 5.1 新增 `check-sandbox-rollout-governance-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
- [x] 5.2 在 diagnostics query baseline 增加 sandbox rollout-enriched dataset 与阈值断言。
- [x] 5.3 在 CI 工作流暴露 rollout-governance 独立 required-check 候选。
- [x] 5.4 补齐 shell/PowerShell gate parity 测试（失败传播一致性）。

## 6. Docs and Validation

- [x] 6.1 更新 `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md`。
- [x] 6.2 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
