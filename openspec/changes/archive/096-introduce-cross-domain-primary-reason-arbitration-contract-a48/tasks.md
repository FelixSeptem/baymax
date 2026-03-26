## 1. Arbitration Core and Deterministic Rules

- [x] 1.1 在 runtime 侧新增 cross-domain arbitration helper，固化 precedence 顺序（timeout > readiness blocked > required unavailable > degraded/optional）。
- [x] 1.2 实现同级冲突 tie-break（canonical code 字典序）与 conflict 计数逻辑。
- [x] 1.3 补齐单测覆盖 precedence、tie-break、无候选/单候选/多候选边界场景。

## 2. Readiness and Admission Integration

- [x] 2.1 在 `runtime/config/readiness*` 接入 arbitration helper，输出统一 primary domain/code/source。
- [x] 2.2 在 admission guard 路径对齐 arbitration 输出，消除 per-path reclassification drift。
- [x] 2.3 补齐 Run/Stream 等价测试，验证 primary reason 解释层一致性。

## 3. Diagnostics and Recorder Additive Fields

- [x] 3.1 在 `runtime/diagnostics` 增加 `runtime_primary_domain`、`runtime_primary_code`、`runtime_primary_source`、`runtime_primary_conflict_total`。
- [x] 3.2 在 `observability/event.RuntimeRecorder` 接入 arbitration 聚合写入并保证 single-writer 语义。
- [x] 3.3 补齐 replay idempotency 测试，确保重复事件不膨胀 primary conflict 计数。

## 4. Replay Tooling and Contract Suites

- [x] 4.1 在 `tool/diagnosticsreplay` 增加 arbitration fixture 支持与 drift 分类（precedence/tie-break/taxonomy）。
- [x] 4.2 在 `integration` 增加 arbitration suites（Run/Stream parity、replay parity、taxonomy drift guard）。
- [x] 4.3 更新 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，将 arbitration suites 纳入阻断步骤并保持 parity。

## 5. Documentation and Acceptance

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md`，补齐 arbitration 字段与解释链路说明。
- [x] 5.2 更新 `docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md`，同步 A48 状态与 gate 映射。
- [x] 5.3 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
