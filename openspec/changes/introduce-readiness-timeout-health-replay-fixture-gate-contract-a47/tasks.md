## 1. Composite Fixture Model and Replay Engine

- [ ] 1.1 在 `tool/diagnosticsreplay` 定义 A47 组合 fixture schema（readiness/timeout/adapter-health）与版本字段。
- [ ] 1.2 实现 composite fixture loader 与规范化输出生成器（canonical semantic fields）。
- [ ] 1.3 实现 deterministic assertion pipeline 与错误分类（schema mismatch / semantic drift / ordering drift）。

## 2. Cross-Domain Assertion Coverage

- [ ] 2.1 增加 readiness 断言覆盖（strict/non-strict、degraded->blocked escalation、primary code 稳定性）。
- [ ] 2.2 增加 timeout-resolution 断言覆盖（profile/domain/request precedence、parent clamp/reject）。
- [ ] 2.3 增加 adapter-health 断言覆盖（status taxonomy、required/optional 路径、circuit state 可见性）。
- [ ] 2.4 增加 Run/Stream parity 断言与 replay idempotency 断言。

## 3. Integration and Quality Gate Wiring

- [ ] 3.1 在 `integration` 增加 readiness-timeout-health composite replay suites 与 golden fixtures。
- [ ] 3.2 更新 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，纳入 A47 阻断步骤并保持 shell/PowerShell parity。
- [ ] 3.3 增加 drift guard 用例（taxonomy/source/state 漂移）并验证 fail-fast 行为。

## 4. Documentation and Contract Index Alignment

- [ ] 4.1 更新 `docs/mainline-contract-test-index.md`，补齐 A47 fixture suite 到 gate 步骤映射。
- [ ] 4.2 更新 `docs/runtime-config-diagnostics.md` 与 `docs/diagnostics-replay.md`，补充 composite fixture 字段与对账说明。
- [ ] 4.3 更新 `docs/development-roadmap.md` 与 `README.md` 的 A47 状态快照与能力描述，清理重复或中间态条目。

## 5. Validation and Acceptance

- [ ] 5.1 执行并记录 `go test ./tool/diagnosticsreplay ./integration -count=1`。
- [ ] 5.2 执行并记录 `pwsh -File scripts/check-docs-consistency.ps1`。
- [ ] 5.3 执行并记录 `pwsh -File scripts/check-quality-gate.ps1`（strict）与 `BAYMAX_SECURITY_SCAN_MODE=warn` 对照结果。
