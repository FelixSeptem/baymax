# tracing-eval-smoke (minimal)

## Purpose
用真实语义链路演示 `tracing-eval-smoke` 的最小闭环：trace span 发射、eval 信号计算、反馈动作闭环。

## Run
go run ./examples/agent-modes/tracing-eval-smoke/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `trace.eval_feedback_loop`.
- Classification: `tracing.eval_interop`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,observability/trace,runtime/diagnostics`.
- Semantic flow:
  - `tracing_span_emitted`: 产出 span 数、p95 延迟、错误率等 trace 指标。
  - `eval_signal_recorded`: 将 trace 指标映射为 eval score 与风险信号。
  - `trace_eval_loop_closed`: 根据 eval 信号选择反馈动作并闭环。
- Related contracts: `otel-tracing-and-agent-eval-interoperability-contract`.
- Required gates: `check-agent-eval-and-tracing-interop-contract.*`.
- Replay fixtures: `otel_semconv.v1; agent_eval.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=trace.eval_feedback_loop`
- `verification.semantic.classification=tracing.eval_interop`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,observability/trace,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=tracing_span_emitted,eval_signal_recorded,trace_eval_loop_closed`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `p95/error_permille/eval_score/eval_signal/action` 等真实闭环字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If eval output is unexpected,优先核对 `p95_latency_ms`、`error_permille` 与 `eval_score` 的对应关系。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
