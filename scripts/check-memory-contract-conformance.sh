#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi
if [[ "${GODEBUG:-}" != *"goindex="* ]]; then
  if [[ -z "${GODEBUG:-}" ]]; then
    export GODEBUG="goindex=0"
  else
    export GODEBUG="${GODEBUG},goindex=0"
  fi
fi

mode="${BAYMAX_MEMORY_CONTRACT_MODE:-smoke}"
mode="$(echo "${mode}" | tr '[:upper:]' '[:lower:]')"
if [[ "${mode}" != "smoke" && "${mode}" != "full" ]]; then
  echo "[memory-contract] unsupported BAYMAX_MEMORY_CONTRACT_MODE=${mode}; expected smoke|full" >&2
  exit 1
fi

run_step() {
  local label="$1"
  shift
  echo "[memory-contract] ${label}"
  "$@"
}

run_step "adapter manifest memory contract suites" \
  go test ./adapter/manifest -run 'Test(ParseMemoryManifest|ActivateMemoryManifest)' -count=1

run_step "memory conformance matrix suites" \
  go test ./integration/adapterconformance -run '^TestMemoryAdapterConformance' -count=1

run_step "runtime memory config/readiness suites" \
  go test ./runtime/config -run 'Test(RuntimeMemoryConfig|ManagerRuntimeMemoryInvalidReloadRollsBack|ManagerReadinessPreflightMemory)' -count=1

run_step "memory replay fixture suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' -count=1

if [[ "${mode}" == "full" ]]; then
  run_step "full memory adapter conformance package" \
    go test ./integration/adapterconformance -count=1
  run_step "full diagnostics replay package" \
    go test ./tool/diagnosticsreplay -count=1
fi

echo "[memory-contract] done (mode=${mode})"
