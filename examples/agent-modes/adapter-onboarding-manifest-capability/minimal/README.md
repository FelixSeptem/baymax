# adapter-onboarding-manifest-capability (minimal)

## Purpose
Real runtime semantic example for `adapter-onboarding-manifest-capability` with `minimal` evidence profile.

## Run
go run ./examples/agent-modes/adapter-onboarding-manifest-capability/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `adapter.manifest_capability_fallback`.
- Classification: `adapter.onboarding`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,adapter/manifest,adapter/capability`.
- Related contracts: `adapter-manifest-and-runtime-compatibility; adapter-capability-negotiation-and-fallback; adapter-contract-profile-versioning-and-replay`.
- Required gates: `check-adapter-manifest-contract.*; check-adapter-capability-contract.*; check-adapter-contract-replay.*`.
- Replay fixtures: `adapter_contract_profile.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=adapter.manifest_capability_fallback`
- `verification.semantic.classification=adapter.onboarding`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,adapter/manifest,adapter/capability`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=adapter_manifest_loaded,adapter_capability_negotiated,adapter_fallback_mapped`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
