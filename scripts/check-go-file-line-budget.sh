#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

POLICY_FILE="openspec/governance/go-file-line-budget-policy.env"
EXCEPTIONS_FILE="openspec/governance/go-file-line-budget-exceptions.csv"

if [[ ! -f "${POLICY_FILE}" ]]; then
  echo "[go-file-line-budget] missing policy file: ${POLICY_FILE}" >&2
  exit 1
fi
if [[ ! -f "${EXCEPTIONS_FILE}" ]]; then
  echo "[go-file-line-budget] missing exception file: ${EXCEPTIONS_FILE}" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${POLICY_FILE}"
set +a

WARN_THRESHOLD="${BAYMAX_GO_LINE_BUDGET_WARN:-800}"
HARD_THRESHOLD="${BAYMAX_GO_LINE_BUDGET_HARD:-1200}"
EXCLUDED_PREFIX="${BAYMAX_GO_LINE_BUDGET_EXCLUDED_PREFIX:-openspec/}"

if ! [[ "${WARN_THRESHOLD}" =~ ^[0-9]+$ ]] || ! [[ "${HARD_THRESHOLD}" =~ ^[0-9]+$ ]]; then
  echo "[go-file-line-budget] threshold must be integer: warn=${WARN_THRESHOLD}, hard=${HARD_THRESHOLD}" >&2
  exit 1
fi
if (( WARN_THRESHOLD <= 0 || HARD_THRESHOLD <= 0 )); then
  echo "[go-file-line-budget] threshold must be > 0: warn=${WARN_THRESHOLD}, hard=${HARD_THRESHOLD}" >&2
  exit 1
fi
if (( WARN_THRESHOLD >= HARD_THRESHOLD )); then
  echo "[go-file-line-budget] warn threshold must be < hard threshold: warn=${WARN_THRESHOLD}, hard=${HARD_THRESHOLD}" >&2
  exit 1
fi

declare -A EX_OWNER=()
declare -A EX_REASON=()
declare -A EX_EXPIRY=()
declare -A EX_BASELINE=()
declare -A EX_ALLOW_GROWTH=()

while IFS=, read -r path owner reason expiry baseline allow_growth; do
  if [[ "${path}" == "path" ]]; then
    continue
  fi
  path="${path//$'\r'/}"
  owner="${owner//$'\r'/}"
  reason="${reason//$'\r'/}"
  expiry="${expiry//$'\r'/}"
  baseline="${baseline//$'\r'/}"
  allow_growth="${allow_growth//$'\r'/}"
  if [[ -z "${path}" ]]; then
    continue
  fi
  if ! [[ "${baseline}" =~ ^[0-9]+$ ]]; then
    echo "[go-file-line-budget] invalid baseline_lines in exception row: ${path},${baseline}" >&2
    exit 1
  fi
  EX_OWNER["${path}"]="${owner}"
  EX_REASON["${path}"]="${reason}"
  EX_EXPIRY["${path}"]="${expiry}"
  EX_BASELINE["${path}"]="${baseline}"
  EX_ALLOW_GROWTH["${path}"]="${allow_growth,,}"
done < "${EXCEPTIONS_FILE}"

today="$(date +%F)"
checked=0
warn_hits=0
hard_hits=0
violations=0

while IFS= read -r file; do
  [[ -z "${file}" ]] && continue
  if [[ "${file}" == "${EXCLUDED_PREFIX}"* ]]; then
    continue
  fi
  if [[ "${file}" == *_test.go ]]; then
    continue
  fi
  lines="$(wc -l < "${file}")"
  lines="${lines//[[:space:]]/}"
  (( checked += 1 ))

  if (( lines > WARN_THRESHOLD )); then
    (( warn_hits += 1 ))
  fi

  if (( lines <= HARD_THRESHOLD )); then
    continue
  fi

  (( hard_hits += 1 ))
  if [[ -v EX_BASELINE["${file}"] ]]; then
    expiry="${EX_EXPIRY["${file}"]}"
    baseline="${EX_BASELINE["${file}"]}"
    allow_growth="${EX_ALLOW_GROWTH["${file}"]:-false}"
    if [[ -n "${expiry}" && "${expiry}" < "${today}" ]]; then
      echo "[go-file-line-budget][violation] expired exception: ${file} expiry=${expiry} today=${today}" >&2
      (( violations += 1 ))
      continue
    fi
    if [[ "${allow_growth}" != "true" ]] && (( lines > baseline )); then
      echo "[go-file-line-budget][violation] oversized debt expanded: ${file} lines=${lines} baseline=${baseline}" >&2
      (( violations += 1 ))
      continue
    fi
    echo "[go-file-line-budget][debt] ${file} lines=${lines} baseline=${baseline} owner=${EX_OWNER["${file}"]} expiry=${expiry}"
  else
    echo "[go-file-line-budget][violation] oversized file without exception: ${file} lines=${lines} hard=${HARD_THRESHOLD}" >&2
    (( violations += 1 ))
  fi
done < <(git ls-files '*.go')

for file in "${!EX_BASELINE[@]}"; do
  if [[ ! -f "${file}" ]]; then
    echo "[go-file-line-budget][violation] stale exception path missing: ${file}" >&2
    (( violations += 1 ))
    continue
  fi
  lines="$(wc -l < "${file}")"
  lines="${lines//[[:space:]]/}"
  if (( lines <= HARD_THRESHOLD )); then
    echo "[go-file-line-budget][violation] stale exception no longer needed: ${file} lines=${lines} hard=${HARD_THRESHOLD}" >&2
    (( violations += 1 ))
  fi
done

echo "[go-file-line-budget] checked=${checked} warn_threshold=${WARN_THRESHOLD} hard_threshold=${HARD_THRESHOLD} warn_hits=${warn_hits} hard_hits=${hard_hits}"
if (( violations > 0 )); then
  echo "[go-file-line-budget] failed: violations=${violations}" >&2
  exit 1
fi
echo "[go-file-line-budget] passed"
