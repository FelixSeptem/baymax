#!/usr/bin/env bash
set -euo pipefail

enabled="${BAYMAX_A64_PERF_REGRESSION_ENABLED:-true}"
if [[ "${enabled}" != "true" ]]; then
  echo "[a64-performance-regression] skipped by BAYMAX_A64_PERF_REGRESSION_ENABLED=${enabled}"
  exit 0
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

log() {
  echo "[a64-performance-regression] $*"
}

substeps=(
  "context production hardening benchmark regression|scripts/check-context-production-hardening-benchmark-regression.sh"
  "diagnostics query benchmark regression|scripts/check-diagnostics-query-performance-regression.sh"
  "multi-agent performance benchmark regression|scripts/check-multi-agent-performance-regression.sh"
)

for item in "${substeps[@]}"; do
  name="${item%%|*}"
  script_path="${item#*|}"
  if [[ -z "${name}" || -z "${script_path}" ]]; then
    echo "[a64-performance-regression] invalid substep definition: ${item}" >&2
    exit 1
  fi
  if [[ ! -f "${script_path}" ]]; then
    echo "[a64-performance-regression] required substep script missing: ${script_path}" >&2
    exit 1
  fi
  log "substep: ${name}"
  bash "${script_path}"
done

log "passed"
