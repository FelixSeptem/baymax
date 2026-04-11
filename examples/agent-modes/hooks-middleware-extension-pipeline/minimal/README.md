# hooks-middleware-extension-pipeline (minimal)

## Purpose
用真实语义链路演示 `hooks-middleware-extension-pipeline` 的最小闭环：onion 顺序校验、错误 bubble、extension 透传。

## Run
go run ./examples/agent-modes/hooks-middleware-extension-pipeline/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `middleware.onion_bubble_passthrough`.
- Classification: `hooks.middleware_pipeline`.
- Runtime path evidence: `core/runner,tool/local,runtime/config`.
- Semantic flow:
  - `middleware_onion_order_verified`: 校验 middleware 进入/退出顺序（onion model）。
  - `middleware_error_bubbled`: 将处理器错误向外层 bubble 并标注 severity/retryable。
  - `middleware_extension_passthrough`: 校验 extension 字段在管道中的透传完整性。
- Related contracts: `agent-lifecycle-hooks-and-tool-middleware-contract`.
- Required gates: `check-hooks-middleware-contract.*`.
- Replay fixtures: `hooks_middleware.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=middleware.onion_bubble_passthrough`
- `verification.semantic.classification=hooks.middleware_pipeline`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=middleware_onion_order_verified,middleware_error_bubbled,middleware_extension_passthrough`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `onion/depth/bubble/severity/retryable/extension/passthrough` 等真实字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If bubble or passthrough output is unexpected,先核对 `bubble_severity` 与 extension 字段数量是否匹配。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
