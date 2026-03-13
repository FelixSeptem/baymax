#!/usr/bin/env bash
set -euo pipefail

enabled="${BAYMAX_CA4_BENCH_ENABLED:-true}"
if [[ "${enabled}" != "true" ]]; then
  echo "[ca4-bench] skipped by BAYMAX_CA4_BENCH_ENABLED=${enabled}"
  exit 0
fi

if [[ -f "scripts/ca4-benchmark-baseline.env" ]]; then
  # shellcheck disable=SC1091
  source "scripts/ca4-benchmark-baseline.env"
fi

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

benchtime="${BAYMAX_CA4_BENCH_BENCHTIME:-150ms}"
count="${BAYMAX_CA4_BENCH_COUNT:-3}"
max_deg_pct="${BAYMAX_CA4_BENCH_MAX_DEGRADATION_PCT:-5}"
max_p95_deg_pct="${BAYMAX_CA4_BENCH_MAX_P95_DEGRADATION_PCT:-8}"
baseline_ns="${BAYMAX_CA4_BENCH_BASELINE_NS_OP:-}"
baseline_p95_ns="${BAYMAX_CA4_BENCH_BASELINE_P95_NS_OP:-}"

if [[ -z "${baseline_ns}" || -z "${baseline_p95_ns}" ]]; then
  echo "[ca4-bench] missing baseline values; set BAYMAX_CA4_BENCH_BASELINE_NS_OP and BAYMAX_CA4_BENCH_BASELINE_P95_NS_OP"
  exit 1
fi

echo "[ca4-bench] running benchmark (benchtime=${benchtime}, count=${count})"
output="$(go test ./integration -run '^$' -bench '^BenchmarkCA4PressureEvaluation$' -benchmem -benchtime="${benchtime}" -count="${count}" 2>&1)"
echo "${output}"

line="$(echo "${output}" | grep 'BenchmarkCA4PressureEvaluation' | tail -n 1 || true)"
if [[ -z "${line}" ]]; then
  echo "[ca4-bench] benchmark output not found"
  exit 1
fi

candidate_ns="$(echo "${line}" | awk '{for(i=1;i<=NF;i++){if($(i+1)=="ns/op"){print $i; exit}}}')"
candidate_p95_ns="$(echo "${line}" | awk '{for(i=1;i<=NF;i++){if($(i+1)=="p95-ns/op"){print $i; exit}}}')"
if [[ -z "${candidate_ns}" || -z "${candidate_p95_ns}" ]]; then
  echo "[ca4-bench] failed to parse ns/op or p95-ns/op from benchmark line: ${line}"
  exit 1
fi

deg_pct="$(awk -v c="${candidate_ns}" -v b="${baseline_ns}" 'BEGIN { printf "%.4f", ((c-b)/b)*100 }')"
p95_deg_pct="$(awk -v c="${candidate_p95_ns}" -v b="${baseline_p95_ns}" 'BEGIN { printf "%.4f", ((c-b)/b)*100 }')"

echo "[ca4-bench] baseline ns/op=${baseline_ns}, candidate ns/op=${candidate_ns}, degradation=${deg_pct}%"
echo "[ca4-bench] baseline p95-ns/op=${baseline_p95_ns}, candidate p95-ns/op=${candidate_p95_ns}, degradation=${p95_deg_pct}%"

ns_fail="$(awk -v d="${deg_pct}" -v m="${max_deg_pct}" 'BEGIN { print (d>m) ? "1" : "0" }')"
p95_fail="$(awk -v d="${p95_deg_pct}" -v m="${max_p95_deg_pct}" 'BEGIN { print (d>m) ? "1" : "0" }')"
if [[ "${ns_fail}" == "1" || "${p95_fail}" == "1" ]]; then
  echo "[ca4-bench] regression threshold exceeded (ns>${max_deg_pct}% or p95>${max_p95_deg_pct}%)"
  exit 1
fi

echo "[ca4-bench] passed"
