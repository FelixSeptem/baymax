# observability-export-bundle (minimal)

## Purpose
用真实语义链路演示 `observability-export-bundle` 的最小闭环：事件收集、bundle 发射、replay 链接。

## Run
go run ./examples/agent-modes/observability-export-bundle/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `observability.export_bundle_replay`.
- Classification: `observability.export_bundle`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,observability/event,runtime/diagnostics`.
- Semantic flow:
  - `observability_export_collected`: 收集 trace/metric/log 导出数据并统计 dropped。
  - `observability_bundle_emitted`: 产出 bundle id/hash/size/compression。
  - `observability_replay_linked`: 建立 replay link 并校验完整性。
- Related contracts: `observability-export-and-diagnostics-bundle-contract`.
- Required gates: `check-observability-export-and-bundle-contract.*`.
- Replay fixtures: `observability.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=observability.export_bundle_replay`
- `verification.semantic.classification=observability.export_bundle`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,observability/event,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=observability_export_collected,observability_bundle_emitted,observability_replay_linked`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `events/dropped/bundle/hash/replay_link/integrity` 等真实字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If bundle or replay output is unexpected,先核对 `dropped_count` 与 `integrity` 一致性。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
