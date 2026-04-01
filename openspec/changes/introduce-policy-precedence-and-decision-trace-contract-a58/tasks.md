## 1. Runtime Policy Config Schema and Validation

- [ ] 1.1 在 `runtime/config` 新增 `runtime.policy.precedence.*` 字段、默认值与 `env > file > default` 解析。
- [ ] 1.2 在 `runtime/config` 新增 `runtime.policy.tie_breaker.*` 与 `runtime.policy.explainability.*` 字段及校验。
- [ ] 1.3 实现启动 fail-fast 与热更新原子回滚（非法 stage、非法 tie-break、冲突矩阵）。
- [ ] 1.4 补齐配置单测（优先级解析、非法更新回滚、默认 precedence matrix）。

## 2. Policy Stack Evaluator and Deterministic Tie-Break

- [ ] 2.1 实现统一 policy stack evaluator（汇总 `action_gate|security_s2|sandbox_action|sandbox_egress|adapter_allowlist|readiness_admission` 候选）。
- [ ] 2.2 冻结 canonical precedence matrix 与 deterministic winner 计算逻辑。
- [ ] 2.3 实现同层冲突 tie-break（lexical code + stable source order）。
- [ ] 2.4 补齐 evaluator 单测（多候选冲突、空候选、同层冲突、版本切换）。

## 3. Runtime Integration in Runner and Security Paths

- [ ] 3.1 在 `core/runner` 与 `runtime/security` 接入 precedence evaluator 输出，不重写既有策略逻辑。
- [ ] 3.2 固化 deny path side-effect-free（禁止触发调度/发布副作用）。
- [ ] 3.3 补齐 Run/Stream 等价测试（同输入同 winner stage / deny source）。

## 4. Readiness Preflight and Admission Alignment

- [ ] 4.1 在 `runtime/config/readiness` 输出 policy stack 候选聚合与 winner-stage 元数据。
- [ ] 4.2 在 admission guard 统一消费 precedence 输出并透传 explainability 字段。
- [ ] 4.3 补齐 strict/non-strict 与 precedence 叠加场景测试（degraded 升级、blocked 优先级）。
- [ ] 4.4 补齐 preflight/admission integration tests（Run/Stream parity + no side effects）。

## 5. Diagnostics and RuntimeRecorder Additive Fields

- [ ] 5.1 在 `runtime/diagnostics` 增加 A58 additive 字段：`policy_decision_path`、`deny_source`、`winner_stage`、`tie_break_reason`。
- [ ] 5.2 在 `observability/event.RuntimeRecorder` 接入 A58 事件映射并保持 single-writer idempotency。
- [ ] 5.3 校验 bounded-cardinality 与 replay idempotency 不回退。

## 6. Replay Fixture and Drift Taxonomy

- [ ] 6.1 在 `tool/diagnosticsreplay` 新增 `policy_stack.v1` fixture schema、loader 与 normalization。
- [ ] 6.2 新增 drift 分类断言：`precedence_conflict`、`tie_break_drift`、`deny_source_mismatch`。
- [ ] 6.3 增加 mixed-fixture 兼容测试（`a50.v1` + `react.v1` + `sandbox_egress.v1` + `policy_stack.v1`）。

## 7. Contract Tests and Integration Matrix

- [ ] 7.1 新增 policy precedence contract tests（跨层冲突矩阵）。
- [ ] 7.2 新增 admission parity tests（Run/Stream + multi-entry consistency）。
- [ ] 7.3 新增 negative matrix tests（配置冲突/缺失 fail-fast + 回滚）。

## 8. Quality Gate and CI Wiring

- [ ] 8.1 新增 `scripts/check-policy-precedence-contract.sh/.ps1`。
- [ ] 8.2 将 A58 contract checks 接入 `scripts/check-quality-gate.*` 阻断路径。
- [ ] 8.3 在 CI 暴露独立 required-check 候选（`policy-precedence-gate`）。
- [ ] 8.4 校验 shell/PowerShell parity（失败传播、退出码、阻断语义一致）。

## 9. Documentation Sync

- [ ] 9.1 更新 `docs/runtime-config-diagnostics.md`（`runtime.policy.*` 配置域、decision trace 字段、taxonomy）。
- [ ] 9.2 更新 `docs/mainline-contract-test-index.md`（A58 contract/replay/gate 索引）。
- [ ] 9.3 更新 `docs/development-roadmap.md` 与 `README.md`（A58 状态与优先级）。
- [ ] 9.4 更新 `runtime/config/README.md`、`runtime/security/README.md` 中 policy precedence 说明。

## 10. Validation

- [ ] 10.1 执行 `go test ./runtime/config ./runtime/config/readiness ./runtime/security -count=1`。
- [ ] 10.2 执行 `go test ./runtime/diagnostics ./observability/event -count=1`。
- [ ] 10.3 执行 `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractPolicyPrecedenceFixture' -count=1`。
- [ ] 10.4 执行 `go test -race ./...`。
- [ ] 10.5 执行 `golangci-lint run --config .golangci.yml`。
- [ ] 10.6 执行 `pwsh -File scripts/check-policy-precedence-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
