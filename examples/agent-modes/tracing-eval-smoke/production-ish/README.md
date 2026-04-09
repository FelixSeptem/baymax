# tracing-eval-smoke (production-ish)

## Purpose
Real runtime semantic example for `tracing-eval-smoke` with `production-ish` evidence profile.

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

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
