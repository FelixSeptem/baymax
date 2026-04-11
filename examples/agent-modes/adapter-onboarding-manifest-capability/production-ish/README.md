# adapter-onboarding-manifest-capability (production-ish)

## Purpose
用真实语义链路演示 `adapter-onboarding-manifest-capability` 的生产治理闭环：在最小链路上增加准入门控与 replay 绑定。

## Variant Delta (vs minimal)
- 生产 manifest 要求更高能力（如 `streaming`），可能产生真实 capability gap。
- 在 fallback 映射后增加治理决策：`allow / allow_with_fallback / deny`。
- 追加 replay 绑定，确保 adapter onboarding 决策可复放审计。

## Run
go run ./examples/agent-modes/adapter-onboarding-manifest-capability/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `adapter.manifest_capability_fallback`.
- Classification: `adapter.onboarding`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,adapter/manifest,adapter/capability`.
- Semantic flow:
  - minimal 的 3 步 onboarding 链路；
  - 追加 `governance_adapter_gate_enforced` 与 `governance_adapter_replay_bound` 两步治理链路。
- Related contracts: `adapter-manifest-and-runtime-compatibility; adapter-capability-negotiation-and-fallback; adapter-contract-profile-versioning-and-replay`.
- Required gates: `check-adapter-manifest-contract.*; check-adapter-capability-contract.*; check-adapter-contract-replay.*`.
- Replay fixtures: `adapter_contract_profile.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=adapter.manifest_capability_fallback`
- `verification.semantic.classification=adapter.onboarding`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,adapter/manifest,adapter/capability`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=adapter_manifest_loaded,adapter_capability_negotiated,adapter_fallback_mapped,governance_adapter_gate_enforced,governance_adapter_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay`，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,检查 `missing_caps` 与 gate 决策是否匹配。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
