## 1. Runtime Config Schema and Validation

- [ ] 1.1 在 `runtime/config` 新增 `security.sandbox.egress.*` 字段、默认值与 `env > file > default` 解析。
- [ ] 1.2 在 `runtime/config` 新增 `adapter.allowlist.*` 字段、默认值与 schema 校验。
- [ ] 1.3 实现启动 fail-fast 与热更新原子回滚（非法 enum、非法 allowlist entry、冲突规则）。
- [ ] 1.4 补齐配置单测（优先级、非法更新回滚、默认 deny-first）。

## 2. Egress Enforcement in Sandbox Path

- [ ] 2.1 在 sandbox 执行路径增加 egress policy 解析与 action 决策（deny/allow/allow_and_record）。
- [ ] 2.2 增加 per-tool 规则覆盖与默认规则优先级收敛。
- [ ] 2.3 固化 egress violation taxonomy 与 deterministic reason code 映射。
- [ ] 2.4 补齐 Run/Stream 等价测试（deny、allow、allow_and_record）。

## 3. Adapter Allowlist Activation Boundary

- [ ] 3.1 扩展 `adapter/manifest`：增加 allowlist identity 字段（adapter_id/publisher/version/signature_status）。
- [ ] 3.2 在 adapter 激活链路增加 allowlist fail-fast 校验，未授权 adapter 禁止加载。
- [ ] 3.3 补齐 allowlist 激活测试（missing entry、signature invalid、allowed path）。

## 4. Readiness and Admission Integration

- [ ] 4.1 在 `runtime/config/readiness` 增加 `sandbox.egress.*` 与 `adapter.allowlist.*` canonical findings。
- [ ] 4.2 固化 strict/non-strict 映射与 deterministic primary reason 断言。
- [ ] 4.3 在 admission guard 接入 A57 findings 的 deny/allow 映射，保持 deny side-effect-free。
- [ ] 4.4 补齐 readiness/admission 集成测试（Run/Stream parity + explainability 透传）。

## 5. Diagnostics and RuntimeRecorder Additive Fields

- [ ] 5.1 在 `runtime/diagnostics` 增加 A57 additive 字段（egress action/source/violations + allowlist decision/code）。
- [ ] 5.2 在 `observability/event.RuntimeRecorder` 接入 A57 事件映射，保持 single-writer idempotency。
- [ ] 5.3 校验 bounded-cardinality 与 replay idempotency 语义。

## 6. Replay Fixture and Drift Taxonomy

- [ ] 6.1 在 `tool/diagnosticsreplay` 新增 `sandbox_egress.v1` fixture schema、loader 与 normalization。
- [ ] 6.2 新增 drift 分类断言（egress action/policy source/violation taxonomy + allowlist decision/taxonomy）。
- [ ] 6.3 增加 mixed-fixture 兼容测试（A52 sandbox.v1 + memory.v1 + react.v1 + sandbox_egress.v1）。

## 7. Conformance and Migration Pack

- [ ] 7.1 在 `integration/adapterconformance` 增加 egress + allowlist matrix suites。
- [ ] 7.2 更新 `docs/adapter-migration-mapping.md` 与模板索引，补齐 A57 迁移映射与 case id 绑定。
- [ ] 7.3 补齐 conformance regression 测试（backend/profile/manifest/taxonomy drift）。

## 8. Quality Gate and CI Wiring

- [ ] 8.1 新增 `scripts/check-sandbox-egress-allowlist-contract.sh/.ps1`。
- [ ] 8.2 将 A57 contract checks 接入 `check-quality-gate.*` 阻断路径。
- [ ] 8.3 在 CI 暴露独立 required-check 候选（`sandbox-egress-allowlist-gate`）。
- [ ] 8.4 校验 shell/PowerShell parity（失败传播、退出码、阻断语义一致）。

## 9. Documentation Sync

- [ ] 9.1 更新 `docs/runtime-config-diagnostics.md`（A57 配置域、findings、additive 字段、taxonomy）。
- [ ] 9.2 更新 `docs/mainline-contract-test-index.md`（A57 contract/replay/gate 索引）。
- [ ] 9.3 更新 `docs/development-roadmap.md` 与 `README.md`（A57 状态快照与优先级）。
- [ ] 9.4 更新 `adapter/README.md`、`runtime/security/README.md` 中与 allowlist/egress 相关说明。

## 10. Validation

- [ ] 10.1 执行 `go test ./runtime/config ./runtime/config/readiness ./runtime/security -count=1`。
- [ ] 10.2 执行 `go test ./adapter/manifest ./integration/adapterconformance -count=1`。
- [ ] 10.3 执行 `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractSandboxEgressAllowlistFixture' -count=1`。
- [ ] 10.4 执行 `go test -race ./...`。
- [ ] 10.5 执行 `golangci-lint run --config .golangci.yml`。
- [ ] 10.6 执行 `pwsh -File scripts/check-sandbox-egress-allowlist-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
