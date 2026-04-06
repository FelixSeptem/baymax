#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${REPO_ROOT}/.gocache"
fi
if [[ "${GODEBUG:-}" != *"goindex="* ]]; then
  if [[ -z "${GODEBUG:-}" ]]; then
    export GODEBUG="goindex=0"
  else
    export GODEBUG="${GODEBUG},goindex=0"
  fi
fi

echo "[memory-scope-search-gate] memory governance implementation suites"
go test ./memory ./context/provider ./context/assembler ./core/runner ./runtime/diagnostics ./observability/event \
  -run 'Test(MemoryProviderPassesGovernanceConfigToFacade|AssemblerContextStage2MemoryGovernanceDiagnosticsFields|MemoryRunDiagnosticsAccumulatorSnapshot|RunFinishedPayloadIncludesMemoryAdditiveFields|StoreRunMemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesMemoryGovernanceAdditiveFields|RuntimeRecorderMemoryGovernanceParserCompatibilityAdditiveNullableDefault)' \
  -count=1

echo "[memory-scope-search-gate] replay and integration fixture suites"
go test ./tool/diagnosticsreplay ./integration \
  -run 'Test(ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' \
  -count=1

echo "[memory-scope-search-gate] done"
