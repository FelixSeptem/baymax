## 1. Explainability Core and Bounded Secondary Semantics

- [x] 1.1 在 runtime arbitration 路径增加 secondary reasons 组装逻辑（去重、排序、上限控制）。
- [x] 1.2 固化 explainability 规则版本输出（`runtime_arbitration_rule_version`）与 canonical 规则来源标签。
- [x] 1.3 增加 remediation hint taxonomy 映射并补齐非法 code fail-fast 校验。

## 2. Readiness and Admission Explainability Integration

- [x] 2.1 在 `runtime/config/readiness*` 输出 explainability 字段（primary + secondary + hint）。
- [x] 2.2 在 admission guard 路径对齐 explainability 语义，避免 per-path remap drift。
- [x] 2.3 补齐 Run/Stream explainability parity 回归测试。

## 3. Diagnostics and Recorder Additive Fields

- [x] 3.1 在 `runtime/diagnostics` 增加 explainability additive 字段：`runtime_secondary_reason_codes`、`runtime_secondary_reason_count`、`runtime_arbitration_rule_version`、`runtime_remediation_hint_code`、`runtime_remediation_hint_domain`。
- [x] 3.2 在 `observability/event.RuntimeRecorder` 接入 explainability 聚合写入并保持 single-writer 语义。
- [x] 3.3 补齐 replay idempotency 测试，确保 explainability 重复事件不膨胀逻辑计数。

## 4. Replay Tooling and Gate Suites

- [x] 4.1 在 `tool/diagnosticsreplay` 增加 explainability fixture 校验与 drift 分类（secondary_order/secondary_count/hint_taxonomy/rule_version）。
- [x] 4.2 在 `integration` 增加 explainability contract suites（Run/Stream parity、replay parity、taxonomy drift guard）。
- [x] 4.3 更新 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，将 explainability suites 纳入阻断步骤并保持 shell/PowerShell parity。

## 5. Documentation and Acceptance

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md`，补齐 explainability 字段说明与兼容窗口。
- [x] 5.2 更新 `docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md`，同步 A49 状态与 gate 映射。
- [x] 5.3 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
