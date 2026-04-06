#!/usr/bin/env bash
set -euo pipefail

canonical_prefix="BAYMAX_CONTEXT_PRODUCTION_HARDENING_BENCH"
benchmark_regex='^(BenchmarkContextProductionHardeningPressureEvaluation)$'

log() {
  echo "[context-production-hardening-bench] $*"
}

read_setting() {
  local canonical_name="$1"
  local default_value="$2"
  local value="${!canonical_name:-}"
  if [[ -z "${value}" ]]; then
    value="${default_value}"
  fi
  printf "%s" "${value}"
}

load_baseline_file() {
  local file="$1"
  if [[ -f "${file}" ]]; then
    # shellcheck disable=SC1090
    source "${file}"
    return 0
  fi
  return 1
}

enabled="$(read_setting "${canonical_prefix}_ENABLED" "true")"
if [[ "${enabled}" != "true" ]]; then
  log "skipped by ${canonical_prefix}_ENABLED=${enabled}"
  exit 0
fi

load_baseline_file "scripts/context-production-hardening-benchmark-baseline.env" || true

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

benchtime="$(read_setting "${canonical_prefix}_BENCHTIME" "150ms")"
count="$(read_setting "${canonical_prefix}_COUNT" "3")"
max_deg_pct="$(read_setting "${canonical_prefix}_MAX_DEGRADATION_PCT" "5")"
max_p95_deg_pct="$(read_setting "${canonical_prefix}_MAX_P95_DEGRADATION_PCT" "8")"
baseline_ns="$(read_setting "${canonical_prefix}_BASELINE_NS_OP" "")"
baseline_p95_ns="$(read_setting "${canonical_prefix}_BASELINE_P95_NS_OP" "")"

if [[ -z "${baseline_ns}" || -z "${baseline_p95_ns}" ]]; then
  log "missing baseline values; set ${canonical_prefix}_BASELINE_NS_OP and ${canonical_prefix}_BASELINE_P95_NS_OP"
  exit 1
fi

log "running benchmark (benchtime=${benchtime}, count=${count})"
output="$(go test ./integration -run '^$' -bench "${benchmark_regex}" -benchmem -benchtime="${benchtime}" -count="${count}" 2>&1)"
echo "${output}"

mapfile -t lines < <(echo "${output}" | grep -E 'BenchmarkContextProductionHardeningPressureEvaluation' || true)
if [[ "${#lines[@]}" -eq 0 ]]; then
  log "benchmark output not found"
  exit 1
fi

median_of_values() {
  if [[ "$#" -eq 0 ]]; then
    return 1
  fi
  printf "%s\n" "$@" | sort -n | awk '
    { values[NR] = $1 }
    END {
      if (NR == 0) {
        exit 1
      }
      mid = int((NR + 1) / 2)
      if (NR % 2 == 1) {
        printf "%s", values[mid]
        exit 0
      }
      printf "%.4f", (values[mid] + values[mid + 1]) / 2
    }
  '
}

ns_samples=()
p95_samples=()
for line in "${lines[@]}"; do
  sample_ns="$(echo "${line}" | awk '{for(i=1;i<=NF;i++){if($(i+1)=="ns/op"){print $i; exit}}}')"
  sample_p95_ns="$(echo "${line}" | awk '{for(i=1;i<=NF;i++){if($(i+1)=="p95-ns/op"){print $i; exit}}}')"
  if [[ -z "${sample_ns}" || -z "${sample_p95_ns}" ]]; then
    log "failed to parse ns/op or p95-ns/op from benchmark line: ${line}"
    exit 1
  fi
  ns_samples+=("${sample_ns}")
  p95_samples+=("${sample_p95_ns}")
done

candidate_ns="$(median_of_values "${ns_samples[@]}" || true)"
candidate_p95_ns="$(median_of_values "${p95_samples[@]}" || true)"
if [[ -z "${candidate_ns}" || -z "${candidate_p95_ns}" ]]; then
  log "failed to compute benchmark medians"
  exit 1
fi

deg_pct="$(awk -v c="${candidate_ns}" -v b="${baseline_ns}" 'BEGIN { printf "%.4f", ((c-b)/b)*100 }')"
p95_deg_pct="$(awk -v c="${candidate_p95_ns}" -v b="${baseline_p95_ns}" 'BEGIN { printf "%.4f", ((c-b)/b)*100 }')"

log "baseline ns/op=${baseline_ns}, candidate ns/op=${candidate_ns}, degradation=${deg_pct}%"
log "baseline p95-ns/op=${baseline_p95_ns}, candidate p95-ns/op=${candidate_p95_ns}, degradation=${p95_deg_pct}%"

ns_fail="$(awk -v d="${deg_pct}" -v m="${max_deg_pct}" 'BEGIN { print (d>m) ? "1" : "0" }')"
p95_fail="$(awk -v d="${p95_deg_pct}" -v m="${max_p95_deg_pct}" 'BEGIN { print (d>m) ? "1" : "0" }')"
if [[ "${ns_fail}" == "1" || "${p95_fail}" == "1" ]]; then
  log "regression threshold exceeded (ns>${max_deg_pct}% or p95>${max_p95_deg_pct}%)"
  exit 1
fi

log "passed"
