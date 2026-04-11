# mcp-governed-stdio-http (production-ish)

## Purpose
Real runtime semantic example for `mcp-governed-stdio-http` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Adds `governance_transport_gate_enforced`: classify transport decision into `allow`, `allow_with_record`, or `block`.
- Adds `governance_transport_replay_bound`: bind replay signature from profile version + failover reason + gate decision.
- Preserves minimal transport chain and then appends governance enforcement.
- Requires verification.semantic.governance=enforced.
- Requires verification.semantic.expected_markers and result.signature to differ from minimal.

## Run
go run ./examples/agent-modes/mcp-governed-stdio-http/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `transport.profile_failover_governance`.
- Classification: `mcp.transport_governance`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,mcp/stdio,mcp/http,mcp/profile`.
- Related contracts: `mcp-runtime-reliability-profiles`.
- Required gates: `check-quality-gate.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=transport.profile_failover_governance`
- `verification.semantic.classification=mcp.transport_governance`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,mcp/stdio,mcp/http,mcp/profile`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=transport_profile_selected,transport_failover_decided,transport_reason_trace_emitted,governance_transport_gate_enforced,governance_transport_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If governance decision/replay binding is unexpected, inspect `governance_transport_*` branches in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


