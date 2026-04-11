# tracing-eval-smoke (production-ish)

## Purpose
用真实语义链路演示 `tracing-eval-smoke` 的生产治理闭环：在最小链路基础上增加治理门控与 replay 绑定。

## Variant Delta (vs minimal)
- 生产输入使用更高延迟/错误率指标，触发 `critical` 信号，行为差异来自真实指标而非 marker 文本。
- 在 feedback 动作后增加治理 gate，输出 `allow/allow_with_sampling/deny`。
- 追加 replay 绑定，确保 trace+eval 决策可审计复放。

## Run
go run ./examples/agent-modes/tracing-eval-smoke/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `trace.eval_feedback_loop`.
- Classification: `tracing.eval_interop`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,observability/trace,runtime/diagnostics`.
- Semantic flow:
  - minimal 的 3 步 tracing/eval 闭环；
  - 追加 `governance_tracing_gate_enforced` 与 `governance_tracing_replay_bound` 两步治理链路。
- Related contracts: `otel-tracing-and-agent-eval-interoperability-contract`.
- Required gates: `check-agent-eval-and-tracing-interop-contract.*`.
- Replay fixtures: `otel_semconv.v1; agent_eval.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=trace.eval_feedback_loop`
- `verification.semantic.classification=tracing.eval_interop`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,observability/trace,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=tracing_span_emitted,eval_signal_recorded,trace_eval_loop_closed,governance_tracing_gate_enforced,governance_tracing_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay` 字段，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance decision is unexpected,检查 `eval_signal` 与 `feedback_action` 是否匹配 gate 结果。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
