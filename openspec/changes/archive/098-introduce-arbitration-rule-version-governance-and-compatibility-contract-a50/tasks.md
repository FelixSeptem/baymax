## 1. Arbitration Version Resolver and Config Surface

- [x] 1.1 在 `runtime/config` 增加 `runtime.arbitration.version.*` 配置结构、默认值与 `env > file > default` 解析。
- [x] 1.2 为 `default/compat_window/on_unsupported/on_mismatch` 增加启动 fail-fast 校验与热更新原子回滚测试。
- [x] 1.3 实现 arbitration version resolver（requested/default/effective/source/policy_action）并补齐单测矩阵。

## 2. Arbitration + Readiness/Admission Integration

- [x] 2.1 在 cross-domain arbitration 路径接入 version resolver，固化 unsupported/mismatch fail-fast 行为。
- [x] 2.2 在 `runtime/config/readiness*` 增加版本治理 finding 映射与 explainability 字段透传。
- [x] 2.3 在 readiness admission guard 路径保持版本治理 explainability 一致性并补齐 Run/Stream parity 测试。

## 3. Diagnostics and RuntimeRecorder Contract

- [x] 3.1 在 `runtime/diagnostics` 增加 A50 additive 字段：requested/effective/source/policy/unsupported_total/mismatch_total。
- [x] 3.2 在 `observability/event.RuntimeRecorder` 接入 A50 字段写入并保持 single-writer 语义。
- [x] 3.3 补齐 replay idempotency 测试，确保版本治理重复事件不膨胀逻辑聚合。

## 4. Replay Tooling and Fixture Drift Guard

- [x] 4.1 在 `tool/diagnosticsreplay` 新增 A50 fixture schema（`a50.v1`）与 loader/normalizer/assert 逻辑。
- [x] 4.2 新增 drift 分类断言：`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`。
- [x] 4.3 在 `integration` 增加 A50 contract suites（Run/Stream parity、memory/file parity、replay parity）。

## 5. Quality Gate and Documentation Sync

- [x] 5.1 更新 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，纳入 A50 suites 并保持 shell/PowerShell parity。
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md` 的 A50 状态与字段说明。
- [x] 5.3 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
