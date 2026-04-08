#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

get_env_or_default() {
  local name="$1"
  local default_value="${2:-}"
  local value="${!name:-}"
  if [[ -z "${value}" ]]; then
    echo "${default_value}"
    return
  fi
  echo "${value}"
}

load_env_defaults_file() {
  local baseline_file="$1"
  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    if [[ -z "${line}" || "${line}" == \#* ]]; then
      continue
    fi
    if [[ "${line}" != *=* ]]; then
      echo "[a64-gate-latency-budget] invalid baseline line (expected KEY=VALUE): ${line}" >&2
      exit 1
    fi
    local key="${line%%=*}"
    local value="${line#*=}"
    if [[ ! "${key}" =~ ^[A-Z0-9_]+$ ]]; then
      echo "[a64-gate-latency-budget] invalid baseline key: ${key}" >&2
      exit 1
    fi
    if [[ -z "${!key:-}" ]]; then
      export "${key}=${value}"
    fi
  done < "${baseline_file}"
}

parse_positive_int() {
  local name="$1"
  local raw="$2"
  if ! [[ "${raw}" =~ ^[0-9]+$ ]] || [[ "${raw}" -le 0 ]]; then
    echo "[a64-gate-latency-budget] ${name} must be a positive integer, got: ${raw}" >&2
    exit 1
  fi
  echo "${raw}"
}

baseline_file="$(get_env_or_default "BAYMAX_A64_GATE_LATENCY_BASELINE_FILE" "${repo_root}/scripts/a64-gate-latency-baseline.env")"
if [[ -n "${baseline_file}" ]]; then
  if [[ ! -f "${baseline_file}" ]]; then
    echo "[a64-gate-latency-budget] baseline file not found: ${baseline_file}" >&2
    exit 1
  fi
  load_env_defaults_file "${baseline_file}"
fi

enabled="$(get_env_or_default "BAYMAX_A64_GATE_LATENCY_ENABLED" "true")"
enabled="$(echo "${enabled}" | tr '[:upper:]' '[:lower:]' | xargs)"
if [[ "${enabled}" != "true" ]]; then
  echo "[a64-gate-latency-budget] skipped by BAYMAX_A64_GATE_LATENCY_ENABLED=${enabled}"
  exit 0
fi

max_step_seconds="$(parse_positive_int "BAYMAX_A64_GATE_LATENCY_MAX_STEP_SECONDS" "$(get_env_or_default "BAYMAX_A64_GATE_LATENCY_MAX_STEP_SECONDS" "600")")"
max_total_seconds="$(parse_positive_int "BAYMAX_A64_GATE_LATENCY_MAX_TOTAL_SECONDS" "$(get_env_or_default "BAYMAX_A64_GATE_LATENCY_MAX_TOTAL_SECONDS" "1200")")"

steps=(
  "a64 impacted gate selection|scripts/check-a64-impacted-gate-selection.sh"
  "a64 semantic stability gate|scripts/check-a64-semantic-stability-contract.sh"
  "a64 performance regression gate|scripts/check-a64-performance-regression.sh"
)

declare -a records
total_start="$(date +%s)"
for item in "${steps[@]}"; do
  name="${item%%|*}"
  script_path="${item#*|}"
  if [[ -z "${name}" || -z "${script_path}" ]]; then
    echo "[a64-gate-latency-budget] invalid step definition: ${item}" >&2
    exit 1
  fi
  if [[ ! -f "${script_path}" ]]; then
    echo "[a64-gate-latency-budget] required script missing: ${script_path}" >&2
    exit 1
  fi
  echo "[a64-gate-latency-budget] step start: ${name}"
  step_start="$(date +%s)"
  bash "${script_path}"
  step_end="$(date +%s)"
  step_seconds="$((step_end - step_start))"
  records+=("${name}|${script_path}|${step_seconds}")
  echo "[a64-gate-latency-budget] step done: ${name} seconds=${step_seconds}"
  if (( step_seconds > max_step_seconds )); then
    echo "[a64-gate-latency-budget] step budget exceeded: ${name} elapsed=${step_seconds}s max=${max_step_seconds}s" >&2
    exit 1
  fi
done
total_end="$(date +%s)"
total_seconds="$((total_end - total_start))"
if (( total_seconds > max_total_seconds )); then
  echo "[a64-gate-latency-budget] total budget exceeded: elapsed=${total_seconds}s max=${max_total_seconds}s" >&2
  exit 1
fi

echo "[a64-gate-latency-budget] report:"
echo "{"
echo "  \"max_total_seconds\": ${max_total_seconds},"
echo "  \"max_step_seconds\": ${max_step_seconds},"
echo "  \"total_seconds\": ${total_seconds},"
echo "  \"steps\": ["
for i in "${!records[@]}"; do
  name="${records[$i]%%|*}"
  rest="${records[$i]#*|}"
  script_path="${rest%%|*}"
  seconds="${rest##*|}"
  comma=","
  if [[ "${i}" -eq "$(( ${#records[@]} - 1 ))" ]]; then
    comma=""
  fi
  echo "    {\"step\":\"${name}\",\"script\":\"${script_path}\",\"seconds\":${seconds}}${comma}"
done
echo "  ]"
echo "}"

report_path="$(get_env_or_default "BAYMAX_A64_GATE_LATENCY_REPORT_PATH" "")"
if [[ -n "${report_path}" ]]; then
  mkdir -p "$(dirname "${report_path}")"
  {
    echo "{"
    echo "  \"max_total_seconds\": ${max_total_seconds},"
    echo "  \"max_step_seconds\": ${max_step_seconds},"
    echo "  \"total_seconds\": ${total_seconds},"
    echo "  \"steps\": ["
    for i in "${!records[@]}"; do
      name="${records[$i]%%|*}"
      rest="${records[$i]#*|}"
      script_path="${rest%%|*}"
      seconds="${rest##*|}"
      comma=","
      if [[ "${i}" -eq "$(( ${#records[@]} - 1 ))" ]]; then
        comma=""
      fi
      echo "    {\"step\":\"${name}\",\"script\":\"${script_path}\",\"seconds\":${seconds}}${comma}"
    done
    echo "  ]"
    echo "}"
  } > "${report_path}"
  echo "[a64-gate-latency-budget] report written to ${report_path}"
fi

echo "[a64-gate-latency-budget] passed"
