#!/usr/bin/env bash
set -euo pipefail

enabled="${BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED:-true}"
if [[ "${enabled}" != "true" ]]; then
  echo "[diagnostics-query-bench] skipped by BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED=${enabled}"
  exit 0
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

load_baseline_defaults() {
  local baseline_file="$1"
  if [[ ! -f "${baseline_file}" ]]; then
    return 0
  fi
  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    if [[ -z "${line}" || "${line}" == \#* ]]; then
      continue
    fi
    if [[ "${line}" != *=* ]]; then
      echo "[diagnostics-query-bench] invalid baseline line (expected KEY=VALUE): ${line}"
      exit 1
    fi
    key="${line%%=*}"
    value="${line#*=}"
    if [[ ! "${key}" =~ ^[A-Z0-9_]+$ ]]; then
      echo "[diagnostics-query-bench] invalid baseline key: ${key}"
      exit 1
    fi
    if [[ -z "${!key:-}" ]]; then
      export "${key}=${value}"
    fi
  done < "${baseline_file}"
}

load_baseline_defaults "${script_dir}/diagnostics-query-benchmark-baseline.env"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${repo_root}/.gocache"
fi

benchtime="${BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME:-200ms}"
count="${BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT:-5}"
max_ns_deg_pct="${BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT:-12}"
max_p95_deg_pct="${BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT:-15}"
max_allocs_deg_pct="${BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT:-12}"

is_number() {
  local value="$1"
  [[ "${value}" =~ ^[0-9]+([.][0-9]+)?$ ]]
}

require_number() {
  local key="$1"
  local value="$2"
  if [[ -z "${value}" ]]; then
    echo "[diagnostics-query-bench] invalid numeric value for ${key}: <empty>"
    exit 1
  fi
  if ! is_number "${value}"; then
    echo "[diagnostics-query-bench] invalid numeric value for ${key}: ${value:-<empty>}"
    exit 1
  fi
}

require_positive_number() {
  local key="$1"
  local value="$2"
  require_number "${key}" "${value}"
  local is_positive
  is_positive="$(awk -v v="${value}" 'BEGIN { print (v>0) ? "1" : "0" }')"
  if [[ "${is_positive}" != "1" ]]; then
    echo "[diagnostics-query-bench] ${key} must be > 0, got ${value}"
    exit 1
  fi
}

if [[ -z "${benchtime}" ]]; then
  echo "[diagnostics-query-bench] invalid BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=<empty>"
  exit 1
fi
if [[ ! "${count}" =~ ^[1-9][0-9]*$ ]]; then
  echo "[diagnostics-query-bench] invalid BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT=${count}; expected positive integer"
  exit 1
fi
require_positive_number "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT" "${max_ns_deg_pct}"
require_positive_number "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT" "${max_p95_deg_pct}"
require_positive_number "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT" "${max_allocs_deg_pct}"

declare -A benchmark_alias=(
  [BenchmarkDiagnosticsQueryRuns]="QUERY_RUNS"
  [BenchmarkDiagnosticsQueryRunsSandboxEnriched]="QUERY_RUNS_SANDBOX_ENRICHED"
  [BenchmarkDiagnosticsQueryMailbox]="QUERY_MAILBOX"
  [BenchmarkDiagnosticsMailboxAggregates]="MAILBOX_AGGREGATES"
  [BenchmarkRuntimeRecorderRunFinished]="RUN_FINISHED"
  [BenchmarkRuntimeRecorderRunFinishedSandboxEnriched]="RUN_FINISHED_SANDBOX_ENRICHED"
)

benchmarks=(
  "BenchmarkDiagnosticsQueryRuns"
  "BenchmarkDiagnosticsQueryRunsSandboxEnriched"
  "BenchmarkDiagnosticsQueryMailbox"
  "BenchmarkDiagnosticsMailboxAggregates"
  "BenchmarkRuntimeRecorderRunFinished"
  "BenchmarkRuntimeRecorderRunFinishedSandboxEnriched"
)

for bench in "${benchmarks[@]}"; do
  path_key="${benchmark_alias[${bench}]}"
  for metric in NS_OP P95_NS_OP ALLOCS_OP; do
    baseline_key="BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_${path_key}_${metric}"
    baseline_value="${!baseline_key:-}"
    require_positive_number "${baseline_key}" "${baseline_value}"
  done
done

extract_metric() {
  local line="$1"
  local metric="$2"
  echo "${line}" | awk -v m="${metric}" '{for(i=1;i<=NF;i++){if($i==m && i>1){print $(i-1); exit}}}'
}

calc_deg_pct() {
  local candidate="$1"
  local baseline="$2"
  awk -v c="${candidate}" -v b="${baseline}" 'BEGIN { printf "%.4f", ((c-b)/b)*100 }'
}

is_threshold_fail() {
  local degradation="$1"
  local threshold="$2"
  awk -v d="${degradation}" -v t="${threshold}" 'BEGIN { print (d>t) ? "1" : "0" }'
}

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

echo "[diagnostics-query-bench] running benchmarks (benchtime=${benchtime}, count=${count})"
output="$(go test ./integration -run '^$' -bench '^Benchmark(DiagnosticsQueryRuns|DiagnosticsQueryRunsSandboxEnriched|DiagnosticsQueryMailbox|DiagnosticsMailboxAggregates|RuntimeRecorderRunFinished|RuntimeRecorderRunFinishedSandboxEnriched)$' -benchmem -benchtime="${benchtime}" -count="${count}" 2>&1)"
echo "${output}"

failed=0
for bench in "${benchmarks[@]}"; do
  path_key="${benchmark_alias[${bench}]}"
  mapfile -t lines < <(echo "${output}" | awk -v bench="${bench}" '$1 ~ ("^" bench "-[0-9]+$") { print $0 }')
  if [[ "${#lines[@]}" -eq 0 ]]; then
    echo "[diagnostics-query-bench] parse-failure benchmark=${bench} reason=missing_output_line"
    exit 1
  fi

  ns_samples=()
  p95_samples=()
  allocs_samples=()
  for line in "${lines[@]}"; do
    sample_ns="$(extract_metric "${line}" "ns/op")"
    sample_p95="$(extract_metric "${line}" "p95-ns/op")"
    sample_allocs="$(extract_metric "${line}" "allocs/op")"
    if [[ -z "${sample_ns}" || -z "${sample_p95}" || -z "${sample_allocs}" ]]; then
      echo "[diagnostics-query-bench] parse-failure benchmark=${bench} reason=missing_required_metric line=${line}"
      exit 1
    fi
    require_positive_number "${bench}.sample.ns/op" "${sample_ns}"
    require_positive_number "${bench}.sample.p95-ns/op" "${sample_p95}"
    require_positive_number "${bench}.sample.allocs/op" "${sample_allocs}"
    ns_samples+=("${sample_ns}")
    p95_samples+=("${sample_p95}")
    allocs_samples+=("${sample_allocs}")
  done

  candidate_ns="$(median_of_values "${ns_samples[@]}" || true)"
  candidate_p95="$(median_of_values "${p95_samples[@]}" || true)"
  candidate_allocs="$(median_of_values "${allocs_samples[@]}" || true)"
  if [[ -z "${candidate_ns}" || -z "${candidate_p95}" || -z "${candidate_allocs}" ]]; then
    echo "[diagnostics-query-bench] parse-failure benchmark=${bench} reason=failed_to_compute_median"
    exit 1
  fi

  require_positive_number "${bench}.candidate.ns/op" "${candidate_ns}"
  require_positive_number "${bench}.candidate.p95-ns/op" "${candidate_p95}"
  require_positive_number "${bench}.candidate.allocs/op" "${candidate_allocs}"

  baseline_ns_key="BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_${path_key}_NS_OP"
  baseline_p95_key="BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_${path_key}_P95_NS_OP"
  baseline_allocs_key="BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_${path_key}_ALLOCS_OP"
  baseline_ns="${!baseline_ns_key}"
  baseline_p95="${!baseline_p95_key}"
  baseline_allocs="${!baseline_allocs_key}"

  ns_deg_pct="$(calc_deg_pct "${candidate_ns}" "${baseline_ns}")"
  p95_deg_pct="$(calc_deg_pct "${candidate_p95}" "${baseline_p95}")"
  allocs_deg_pct="$(calc_deg_pct "${candidate_allocs}" "${baseline_allocs}")"

  echo "[diagnostics-query-bench] ${bench} ns/op baseline=${baseline_ns} candidate=${candidate_ns} degradation=${ns_deg_pct}% (max=${max_ns_deg_pct}%)"
  echo "[diagnostics-query-bench] ${bench} p95-ns/op baseline=${baseline_p95} candidate=${candidate_p95} degradation=${p95_deg_pct}% (max=${max_p95_deg_pct}%)"
  echo "[diagnostics-query-bench] ${bench} allocs/op baseline=${baseline_allocs} candidate=${candidate_allocs} degradation=${allocs_deg_pct}% (max=${max_allocs_deg_pct}%)"

  ns_fail="$(is_threshold_fail "${ns_deg_pct}" "${max_ns_deg_pct}")"
  p95_fail="$(is_threshold_fail "${p95_deg_pct}" "${max_p95_deg_pct}")"
  allocs_fail="$(is_threshold_fail "${allocs_deg_pct}" "${max_allocs_deg_pct}")"
  if [[ "${ns_fail}" == "1" || "${p95_fail}" == "1" || "${allocs_fail}" == "1" ]]; then
    echo "[diagnostics-query-bench] regression-threshold-exceeded benchmark=${bench}"
    failed=1
  fi
done

if [[ "${failed}" == "1" ]]; then
  echo "[diagnostics-query-bench] failed"
  exit 1
fi

echo "[diagnostics-query-bench] passed"
