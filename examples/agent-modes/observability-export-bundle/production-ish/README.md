# observability-export-bundle (production-ish)

## Purpose
用真实语义链路演示 `observability-export-bundle` 的生产治理闭环：在最小链路上增加质量门控与 replay 绑定。

## Variant Delta (vs minimal)
- 生产输入有更高 export 体量与 dropped 事件，行为差异来自真实数据质量而非 marker 文本。
- 在 replay 链接后增加治理判定：`allow / allow_with_sampling / deny`。
- 追加 replay-bound 签名，确保 observability 交付可审计复放。

## Run
go run ./examples/agent-modes/observability-export-bundle/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `observability.export_bundle_replay`.
- Classification: `observability.export_bundle`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,observability/event,runtime/diagnostics`.
- Semantic flow:
  - minimal 的 3 步 export/bundle/replay 链路；
  - 追加 `governance_observability_gate_enforced` 与 `governance_observability_replay_bound` 两步治理链路。
- Related contracts: `observability-export-and-diagnostics-bundle-contract`.
- Required gates: `check-observability-export-and-bundle-contract.*`.
- Replay fixtures: `observability.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=observability.export_bundle_replay`
- `verification.semantic.classification=observability.export_bundle`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,observability/event,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=observability_export_collected,observability_bundle_emitted,observability_replay_linked,governance_observability_gate_enforced,governance_observability_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay` 字段，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,检查 `dropped`、`bundle_size_kb` 与 gate 决策是否一致。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
