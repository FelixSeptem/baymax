#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${REPO_ROOT}/.gocache"
fi
if [[ -z "${GODEBUG:-}" ]]; then
  export GODEBUG="goindex=0"
elif [[ "${GODEBUG}" != *"goindex="* ]]; then
  export GODEBUG="${GODEBUG},goindex=0"
fi

echo "[go-split-strong-check] impacted memory contract suites"
if ! go test ./memory ./context/provider ./context/assembler ./core/runner ./runtime/diagnostics ./observability/event -run 'Test(MemoryProviderPassesGovernanceConfigToFacade|AssemblerContextStage2MemoryGovernanceDiagnosticsFields|MemoryRunDiagnosticsAccumulatorSnapshot|RunFinishedPayloadIncludesMemoryAdditiveFields|StoreRunMemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesMemoryGovernanceAdditiveFields|RuntimeRecorderMemoryGovernanceParserCompatibilityAdditiveNullableDefault)' -count=1; then
  echo "[go-split-strong-check][impacted-contract] impacted memory contract suites failed"
  exit 1
fi

echo "[go-split-strong-check] run stream parity suites"
if ! go test ./integration -run 'Test(TimeoutResolutionContractRunStreamAndMemoryFileParity|RuntimeReadinessAdmissionContractBlockedDenyRunStreamEquivalentAndNoSideEffects|RuntimeReadinessAdmissionContractAdapterCircuitOpenRunStreamParity)' -count=1; then
  echo "[go-split-strong-check][run-stream-parity] run stream parity suites failed"
  exit 1
fi

echo "[go-split-strong-check] replay idempotency and drift suites"
if ! go test ./tool/diagnosticsreplay ./integration -run 'Test(TimeoutResolutionContractReplayIdempotency|ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' -count=1; then
  echo "[go-split-strong-check][replay-idempotency] replay idempotency and drift suites failed"
  exit 1
fi

echo "[go-split-strong-check] passed"
