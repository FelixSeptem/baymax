# custom-adapter-mcp-model-tool-memory-pack (minimal)

## Purpose
Real runtime semantic example for `custom-adapter-mcp-model-tool-memory-pack` with `minimal` evidence profile.
This variant demonstrates adapter pack manifest resolution, capability fallback selection, and memory scope binding.

## Run
go run ./examples/agent-modes/custom-adapter-mcp-model-tool-memory-pack/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `adapterpack.manifest_capability_memory`.
- Classification: `adapter.custom_pack`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,adapter/scaffold,mcp/profile,memory`.
- Related contracts: `external-adapter-template-and-migration-mapping; external-adapter-conformance-harness; adapter-scaffold-generator`.
- Required gates: `check-adapter-conformance.*; check-adapter-scaffold-drift.*`.
- Replay fixtures: `adapter_contract_profile.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=adapterpack.manifest_capability_memory`
- `verification.semantic.classification=adapter.custom_pack`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,adapter/scaffold,mcp/profile,memory`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=adapter_pack_manifest_resolved,adapter_pack_capability_fallback,adapter_pack_memory_scope_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes adapter-pack fields: `manifest`, `transport`, `model`, `tools`, `capability_fallback`, `memory_scope`, `memory_namespace`, `pack_ready`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If fallback/memory behavior diverges, inspect marker handlers for `adapter_pack_capability_fallback` and `adapter_pack_memory_scope_bound`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
