# multi-agents-collab-recovery (production-ish)

## Purpose
Real runtime semantic example for `multi-agents-collab-recovery` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/multi-agents-collab-recovery/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `collab.mailbox_taskboard_recovery`.
- Classification: `multi_agents.collaboration_recovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/collab,orchestration/mailbox,orchestration/scheduler`.
- Related contracts: `multi-agent-collaboration-primitives; long-running-recovery-boundary`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=collab.mailbox_taskboard_recovery`
- `verification.semantic.classification=multi_agents.collaboration_recovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/collab,orchestration/mailbox,orchestration/scheduler`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=collab_mailbox_orchestrated,collab_task_board_reconciled,collab_recovery_continued,governance_collab_gate_enforced,governance_collab_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
