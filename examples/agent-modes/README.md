# Agent Modes Example Pack

## Purpose
Real runtime semantic examples for 28 agent modes with auditable minimal/production-ish variant evidence.

## Run
- Single variant: `go run ./examples/agent-modes/<pattern>/<minimal|production-ish>`
- Batch smoke: `pwsh -File scripts/check-agent-mode-examples-smoke.ps1`

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local `GOCACHE` for deterministic smoke execution.
- No external network service is required for these examples.

## Real Runtime Path
- Shared runtime path baseline: `core/runner,tool/local,runtime/config`.
- Pattern-specific domain paths are recorded in `examples/agent-modes/MATRIX.md`.

## Expected Output/Verification
- Output must include `verification.mainline_runtime_path=ok`.
- Output must include semantic evidence fields under `verification.semantic.*`.
- Output must include `result.final_answer=` and `result.signature=`.
- Production-ish output must include `verification.semantic.governance=enforced`.

## Failure/Rollback Notes
- If smoke fails, run the target variant directly and inspect missing `verification.semantic.*` markers.
- For semantic drift, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- For README drift, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- If rollback is required, revert the affected mode directory and regenerate docs from mode specs.

## Contract Mapping
- Matrix: `examples/agent-modes/MATRIX.md`
- Migration playbook: `examples/agent-modes/PLAYBOOK.md`
- Pattern count: `28`
