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
      echo "[a64-harnessability-scorecard] invalid baseline line (expected KEY=VALUE): ${line}" >&2
      exit 1
    fi
    local key="${line%%=*}"
    local value="${line#*=}"
    if [[ ! "${key}" =~ ^[A-Z0-9_]+$ ]]; then
      echo "[a64-harnessability-scorecard] invalid baseline key: ${key}" >&2
      exit 1
    fi
    if [[ -z "${!key:-}" ]]; then
      export "${key}=${value}"
    fi
  done < "${baseline_file}"
}

require_non_negative_int() {
  local name="$1"
  local raw="$2"
  if ! [[ "${raw}" =~ ^[0-9]+$ ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be a non-negative integer, got: ${raw}" >&2
    exit 1
  fi
}

require_positive_number() {
  local name="$1"
  local raw="$2"
  if ! [[ "${raw}" =~ ^-?[0-9]+([.][0-9]+)?$ ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be numeric, got: ${raw}" >&2
    exit 1
  fi
  local ok
  ok="$(awk -v v="${raw}" 'BEGIN { print (v>0) ? "1" : "0" }')"
  if [[ "${ok}" != "1" ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be > 0, got: ${raw}" >&2
    exit 1
  fi
}

require_percent() {
  local name="$1"
  local raw="$2"
  if ! [[ "${raw}" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be numeric percent, got: ${raw}" >&2
    exit 1
  fi
  local ok
  ok="$(awk -v v="${raw}" 'BEGIN { print (v>=0 && v<=100) ? "1" : "0" }')"
  if [[ "${ok}" != "1" ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be within [0,100], got: ${raw}" >&2
    exit 1
  fi
}

require_signed_percent() {
  local name="$1"
  local raw="$2"
  if ! [[ "${raw}" =~ ^-?[0-9]+([.][0-9]+)?$ ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be numeric, got: ${raw}" >&2
    exit 1
  fi
  local ok
  ok="$(awk -v v="${raw}" 'BEGIN { print (v>=-100 && v<=100) ? "1" : "0" }')"
  if [[ "${ok}" != "1" ]]; then
    echo "[a64-harnessability-scorecard] ${name} must be within [-100,100], got: ${raw}" >&2
    exit 1
  fi
}

parse_bool() {
  local name="$1"
  local raw="$2"
  local value
  value="$(echo "${raw}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "${value}" in
    true | false)
      echo "${value}"
      ;;
    *)
      echo "[a64-harnessability-scorecard] ${name} must be true|false, got: ${raw}" >&2
      exit 1
      ;;
  esac
}

parse_tier() {
  local raw="$1"
  local value
  value="$(echo "${raw}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "${value}" in
    lightweight | standard | enhanced)
      echo "${value}"
      ;;
    *)
      echo "[a64-harnessability-scorecard] BAYMAX_A64_HARNESS_COMPLEXITY_TIER must be lightweight|standard|enhanced, got: ${raw}" >&2
      exit 1
      ;;
  esac
}

to_upper() {
  echo "$1" | tr '[:lower:]' '[:upper:]'
}

round2() {
  awk -v v="$1" 'BEGIN { printf "%.2f", v }'
}

calc_overhead_pct() {
  local measured="$1"
  local baseline="$2"
  awk -v m="${measured}" -v b="${baseline}" 'BEGIN { printf "%.2f", ((m-b)/b)*100 }'
}

num_lt() {
  awk -v a="$1" -v b="$2" 'BEGIN { print (a < b) ? "1" : "0" }'
}

num_gt() {
  awk -v a="$1" -v b="$2" 'BEGIN { print (a > b) ? "1" : "0" }'
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

default_baseline_file="${repo_root}/scripts/a64-harnessability-scorecard-baseline.env"
baseline_file="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_BASELINE_FILE" "${default_baseline_file}")"
if [[ -n "${baseline_file}" ]]; then
  if [[ ! -f "${baseline_file}" ]]; then
    echo "[a64-harnessability-scorecard] baseline file not found: ${baseline_file}" >&2
    exit 1
  fi
  load_env_defaults_file "${baseline_file}"
fi

enabled="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_ENABLED" "true")"
enabled="$(echo "${enabled}" | tr '[:upper:]' '[:lower:]' | xargs)"
if [[ "${enabled}" != "true" ]]; then
  echo "[a64-harnessability-scorecard] skipped by BAYMAX_A64_HARNESS_SCORECARD_ENABLED=${enabled}"
  exit 0
fi

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${repo_root}/.gocache"
fi

declare -a failures

# Contract coverage: invoke impacted gate selection with full mode.
impacted_report_path="$(get_env_or_default "BAYMAX_A64_SCORECARD_IMPACTED_REPORT_PATH" "${repo_root}/.artifacts/a64/impacted-full-report.json")"
mkdir -p "$(dirname "${impacted_report_path}")"
BAYMAX_A64_GATE_SELECTION_MODE=full BAYMAX_A64_IMPACTED_REPORT_PATH="${impacted_report_path}" bash scripts/check-a64-impacted-gate-selection.sh
if [[ ! -f "${impacted_report_path}" ]]; then
  echo "[a64-harnessability-scorecard] impacted report missing: ${impacted_report_path}" >&2
  exit 1
fi
impacted_count="$(grep -o '"S[0-9]\+"' "${impacted_report_path}" | sort -u | wc -l | tr -d ' ' || true)"
if [[ -z "${impacted_count}" ]]; then
  impacted_count=0
fi
contract_coverage_pct="$(round2 "$(awk -v c="${impacted_count}" 'BEGIN { printf "%.8f", (c/10)*100 }')")"
min_contract_coverage_pct="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_MIN_CONTRACT_COVERAGE_PCT" "100")"
require_percent "BAYMAX_A64_HARNESS_SCORECARD_MIN_CONTRACT_COVERAGE_PCT" "${min_contract_coverage_pct}"
contract_coverage_within=true
if [[ "$(num_lt "${contract_coverage_pct}" "${min_contract_coverage_pct}")" == "1" ]]; then
  contract_coverage_within=false
  failures+=("contract_coverage_pct=${contract_coverage_pct} < min=${min_contract_coverage_pct}")
fi

# Drift statistics.
drift_fixture_count=0
if [[ -d "tool/diagnosticsreplay/testdata" ]]; then
  drift_fixture_count="$(find "tool/diagnosticsreplay/testdata" -maxdepth 1 -type f | grep -Ei '(inferential|drift)' | wc -l | tr -d ' ' || true)"
  if [[ -z "${drift_fixture_count}" ]]; then
    drift_fixture_count=0
  fi
fi
min_drift_fixture_count="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_MIN_DRIFT_FIXTURE_COUNT" "2")"
unclassified_drift_count="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_UNCLASSIFIED_DRIFT_COUNT" "0")"
max_unclassified_drift_count="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_MAX_UNCLASSIFIED_DRIFT_COUNT" "0")"
require_non_negative_int "BAYMAX_A64_HARNESS_SCORECARD_MIN_DRIFT_FIXTURE_COUNT" "${min_drift_fixture_count}"
require_non_negative_int "BAYMAX_A64_HARNESS_SCORECARD_UNCLASSIFIED_DRIFT_COUNT" "${unclassified_drift_count}"
require_non_negative_int "BAYMAX_A64_HARNESS_SCORECARD_MAX_UNCLASSIFIED_DRIFT_COUNT" "${max_unclassified_drift_count}"
drift_within=true
if (( drift_fixture_count < min_drift_fixture_count )) || (( unclassified_drift_count > max_unclassified_drift_count )); then
  drift_within=false
  failures+=("drift_stats fixture_count=${drift_fixture_count} (min=${min_drift_fixture_count}) unclassified=${unclassified_drift_count} (max=${max_unclassified_drift_count})")
fi

# Gate coverage in quality-gate shell + PowerShell.
quality_gate_shell="$(cat scripts/check-quality-gate.sh)"
quality_gate_ps="$(cat scripts/check-quality-gate.ps1)"
required_gate_total=4
covered_gate_count=0
declare -a gate_details

gate_names=(
  "a64 impacted gate selection"
  "a64 semantic stability gate"
  "a64 performance regression gate"
  "a64 harnessability scorecard"
)
gate_shell_tokens=(
  "check-a64-impacted-gate-selection.sh"
  "check-a64-semantic-stability-contract.sh"
  "check-a64-performance-regression.sh"
  "check-a64-harnessability-scorecard.sh"
)
gate_ps_tokens=(
  "check-a64-impacted-gate-selection.ps1"
  "check-a64-semantic-stability-contract.ps1"
  "check-a64-performance-regression.ps1"
  "check-a64-harnessability-scorecard.ps1"
)

for i in "${!gate_names[@]}"; do
  shell_found=false
  ps_found=false
  if grep -Fq "${gate_shell_tokens[$i]}" <<< "${quality_gate_shell}"; then
    shell_found=true
  fi
  if grep -Fq "${gate_ps_tokens[$i]}" <<< "${quality_gate_ps}"; then
    ps_found=true
  fi
  covered=false
  if [[ "${shell_found}" == "true" && "${ps_found}" == "true" ]]; then
    covered=true
    covered_gate_count=$((covered_gate_count + 1))
  fi
  gate_details+=("{\"gate\":\"$(json_escape "${gate_names[$i]}")\",\"shell_token\":\"$(json_escape "${gate_shell_tokens[$i]}")\",\"powershell_token\":\"$(json_escape "${gate_ps_tokens[$i]}")\",\"shell_found\":${shell_found},\"powershell_found\":${ps_found},\"covered\":${covered}}")
done

gate_coverage_pct="$(round2 "$(awk -v c="${covered_gate_count}" -v t="${required_gate_total}" 'BEGIN { printf "%.8f", (c/t)*100 }')")"
min_gate_coverage_pct="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_MIN_GATE_COVERAGE_PCT" "100")"
require_percent "BAYMAX_A64_HARNESS_SCORECARD_MIN_GATE_COVERAGE_PCT" "${min_gate_coverage_pct}"
gate_coverage_within=true
if [[ "$(num_lt "${gate_coverage_pct}" "${min_gate_coverage_pct}")" == "1" ]]; then
  gate_coverage_within=false
  failures+=("gate_coverage_pct=${gate_coverage_pct} < min=${min_gate_coverage_pct}")
fi

# Docs consistency markers.
docs_issue_count=0
declare -a docs_details

check_doc_marker() {
  local file_path="$1"
  local marker="$2"
  if [[ ! -f "${file_path}" ]]; then
    docs_issue_count=$((docs_issue_count + 1))
    docs_details+=("{\"path\":\"$(json_escape "${file_path}")\",\"marker\":\"<file>\",\"found\":false}")
    return
  fi
  local found=false
  if grep -Fq "${marker}" "${file_path}"; then
    found=true
  else
    docs_issue_count=$((docs_issue_count + 1))
  fi
  docs_details+=("{\"path\":\"$(json_escape "${file_path}")\",\"marker\":\"$(json_escape "${marker}")\",\"found\":${found}}")
}

check_doc_marker "docs/development-roadmap.md" "harnessability scorecard"
check_doc_marker "docs/development-roadmap.md" "harness ROI/depth"
check_doc_marker "docs/development-roadmap.md" "computational-first, inferential-second"
check_doc_marker "docs/development-roadmap.md" "门禁耗时预算治理"
check_doc_marker "docs/mainline-contract-test-index.md" "check-a64-harnessability-scorecard.sh"
check_doc_marker "docs/mainline-contract-test-index.md" "check-a64-harnessability-scorecard.ps1"
check_doc_marker "docs/mainline-contract-test-index.md" "a64-harnessability-scorecard-baseline.env"
check_doc_marker "docs/mainline-contract-test-index.md" "a64-gate-latency-baseline.env"

governance_index_path="openspec/changes/introduce-engineering-and-performance-optimization-contract-a64/a64-governance-index.md"
if [[ ! -f "${governance_index_path}" ]]; then
  archived_governance_dir="$(find "openspec/changes/archive" -maxdepth 1 -type d -name "*introduce-engineering-and-performance-optimization-contract-a64" | sort | tail -n 1 || true)"
  if [[ -n "${archived_governance_dir}" ]]; then
    governance_index_path="${archived_governance_dir}/a64-governance-index.md"
  fi
fi

check_doc_marker "${governance_index_path}" "Harnessability Scorecard"
check_doc_marker "${governance_index_path}" "门禁耗时基线"

max_docs_issue_count="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_MAX_DOCS_ISSUE_COUNT" "0")"
require_non_negative_int "BAYMAX_A64_HARNESS_SCORECARD_MAX_DOCS_ISSUE_COUNT" "${max_docs_issue_count}"
docs_within=true
if (( docs_issue_count > max_docs_issue_count )); then
  docs_within=false
  failures+=("docs_consistency_issue_count=${docs_issue_count} > max=${max_docs_issue_count}")
fi

# ROI + adaptive depth.
tier="$(parse_tier "$(get_env_or_default "BAYMAX_A64_HARNESS_COMPLEXITY_TIER" "standard")")"
tier_upper="$(to_upper "${tier}")"

baseline_token_var="BAYMAX_A64_HARNESS_BASELINE_TOKEN_${tier_upper}"
baseline_latency_var="BAYMAX_A64_HARNESS_BASELINE_LATENCY_MS_${tier_upper}"
baseline_quality_var="BAYMAX_A64_HARNESS_BASELINE_QUALITY_${tier_upper}"
max_token_overhead_var="BAYMAX_A64_HARNESS_MAX_TOKEN_OVERHEAD_PCT_${tier_upper}"
max_latency_overhead_var="BAYMAX_A64_HARNESS_MAX_LATENCY_OVERHEAD_PCT_${tier_upper}"
min_quality_delta_var="BAYMAX_A64_HARNESS_MIN_QUALITY_DELTA_PCT_${tier_upper}"

baseline_token="${!baseline_token_var:-}"
baseline_latency_ms="${!baseline_latency_var:-}"
baseline_quality="${!baseline_quality_var:-}"
max_token_overhead_pct="${!max_token_overhead_var:-}"
max_latency_overhead_pct="${!max_latency_overhead_var:-}"
min_quality_delta_pct="${!min_quality_delta_var:-}"

require_positive_number "${baseline_token_var}" "${baseline_token}"
require_positive_number "${baseline_latency_var}" "${baseline_latency_ms}"
require_positive_number "${baseline_quality_var}" "${baseline_quality}"
require_percent "${max_token_overhead_var}" "${max_token_overhead_pct}"
require_percent "${max_latency_overhead_var}" "${max_latency_overhead_pct}"
require_signed_percent "${min_quality_delta_var}" "${min_quality_delta_pct}"

measured_token="$(get_env_or_default "BAYMAX_A64_HARNESS_MEASURED_TOKEN" "${baseline_token}")"
measured_latency_ms="$(get_env_or_default "BAYMAX_A64_HARNESS_MEASURED_LATENCY_MS" "${baseline_latency_ms}")"
measured_quality="$(get_env_or_default "BAYMAX_A64_HARNESS_MEASURED_QUALITY_SCORE" "${baseline_quality}")"

require_positive_number "BAYMAX_A64_HARNESS_MEASURED_TOKEN" "${measured_token}"
require_positive_number "BAYMAX_A64_HARNESS_MEASURED_LATENCY_MS" "${measured_latency_ms}"
require_positive_number "BAYMAX_A64_HARNESS_MEASURED_QUALITY_SCORE" "${measured_quality}"

token_overhead_pct="$(calc_overhead_pct "${measured_token}" "${baseline_token}")"
latency_overhead_pct="$(calc_overhead_pct "${measured_latency_ms}" "${baseline_latency_ms}")"
quality_delta_pct="$(calc_overhead_pct "${measured_quality}" "${baseline_quality}")"

roi_within=true
if [[ "$(num_gt "${token_overhead_pct}" "${max_token_overhead_pct}")" == "1" ]] ||
  [[ "$(num_gt "${latency_overhead_pct}" "${max_latency_overhead_pct}")" == "1" ]] ||
  [[ "$(num_lt "${quality_delta_pct}" "${min_quality_delta_pct}")" == "1" ]]; then
  roi_within=false
fi

recommended_tier="${tier}"
if [[ "${roi_within}" != "true" ]]; then
  case "${tier}" in
    enhanced)
      recommended_tier="standard"
      ;;
    standard)
      recommended_tier="lightweight"
      ;;
    *)
      recommended_tier="lightweight"
      ;;
  esac
  failures+=("roi thresholds breached tier=${tier} token_overhead=${token_overhead_pct}% (max=${max_token_overhead_pct}%) latency_overhead=${latency_overhead_pct}% (max=${max_latency_overhead_pct}%) quality_delta=${quality_delta_pct}% (min=${min_quality_delta_pct}%)")
fi

# Computational-first, inferential-second hierarchy + structured evidence.
computational_total=3
computational_present=0
for script_path in \
  "scripts/check-a64-impacted-gate-selection.sh" \
  "scripts/check-a64-semantic-stability-contract.sh" \
  "scripts/check-a64-performance-regression.sh"; do
  if [[ -f "${script_path}" ]]; then
    computational_present=$((computational_present + 1))
  fi
done
computational_coverage_pct="$(round2 "$(awk -v c="${computational_present}" -v t="${computational_total}" 'BEGIN { printf "%.8f", (c/t)*100 }')")"
inferential_blocking_requested="$(parse_bool "BAYMAX_A64_INFERENTIAL_BLOCKING_REQUESTED" "$(get_env_or_default "BAYMAX_A64_INFERENTIAL_BLOCKING_REQUESTED" "false")")"
computational_first_compliant=true
if [[ "$(num_lt "${computational_coverage_pct}" "100")" == "1" || "${inferential_blocking_requested}" == "true" ]]; then
  computational_first_compliant=false
  failures+=("computational-first hierarchy violated: computational_coverage_pct=${computational_coverage_pct}, inferential_blocking_requested=${inferential_blocking_requested}")
fi

input_snapshot_path="$(get_env_or_default "BAYMAX_A64_INFERENTIAL_INPUT_SNAPSHOT" "tool/diagnosticsreplay/testdata/a61_inferential_advisory_distributed_success_input.json")"
prompt_version="$(get_env_or_default "BAYMAX_A64_INFERENTIAL_PROMPT_VERSION" "a64-harnessability-v1")"
scoring_summary="$(get_env_or_default "BAYMAX_A64_INFERENTIAL_SCORING_SUMMARY" "tier=${tier}; quality_delta_pct=${quality_delta_pct}")"
uncertainty_pct="$(get_env_or_default "BAYMAX_A64_INFERENTIAL_UNCERTAINTY_PCT" "15")"
max_uncertainty_pct="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_MAX_INFERENTIAL_UNCERTAINTY_PCT" "35")"
require_percent "BAYMAX_A64_INFERENTIAL_UNCERTAINTY_PCT" "${uncertainty_pct}"
require_percent "BAYMAX_A64_HARNESS_SCORECARD_MAX_INFERENTIAL_UNCERTAINTY_PCT" "${max_uncertainty_pct}"
uncertainty_within=true
if [[ "$(num_gt "${uncertainty_pct}" "${max_uncertainty_pct}")" == "1" ]]; then
  uncertainty_within=false
fi

evidence_complete=true
if [[ -z "${input_snapshot_path}" || ! -f "${input_snapshot_path}" || -z "${prompt_version}" || -z "${scoring_summary}" ]]; then
  evidence_complete=false
  failures+=("inferential evidence incomplete: input_snapshot/prompt_version/scoring_summary must be provided and snapshot file must exist")
fi
if [[ "${inferential_blocking_requested}" == "true" && "${uncertainty_within}" != "true" ]]; then
  failures+=("inferential uncertainty ${uncertainty_pct}% exceeds max ${max_uncertainty_pct}% and cannot be used as blocking signal")
fi

score_pass=true
if [[ "${#failures[@]}" -gt 0 ]]; then
  score_pass=false
fi

timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

json_report_path="$(get_env_or_default "BAYMAX_A64_HARNESS_SCORECARD_REPORT_PATH" ".artifacts/a64/harnessability-scorecard.json")"
mkdir -p "$(dirname "${json_report_path}")"

{
  echo "{"
  echo "  \"generated_at\": \"$(json_escape "${timestamp}")\","
  echo "  \"complexity_tier\": \"$(json_escape "${tier}")\","
  echo "  \"metrics\": {"
  echo "    \"contract_coverage_pct\": ${contract_coverage_pct},"
  echo "    \"drift\": {\"fixture_count\": ${drift_fixture_count}, \"min_fixture_count\": ${min_drift_fixture_count}, \"unclassified_count\": ${unclassified_drift_count}, \"max_unclassified_count\": ${max_unclassified_drift_count}, \"within_threshold\": ${drift_within}},"
  echo "    \"gate_coverage_pct\": ${gate_coverage_pct},"
  echo "    \"docs_consistency\": {\"issue_count\": ${docs_issue_count}, \"max_issue_count\": ${max_docs_issue_count}, \"within_threshold\": ${docs_within}},"
  echo "    \"roi\": {"
  echo "      \"baseline\": {\"token\": ${baseline_token}, \"latency_ms\": ${baseline_latency_ms}, \"quality_score\": ${baseline_quality}},"
  echo "      \"measured\": {\"token\": ${measured_token}, \"latency_ms\": ${measured_latency_ms}, \"quality_score\": ${measured_quality}},"
  echo "      \"overhead_pct\": {\"token\": ${token_overhead_pct}, \"latency\": ${latency_overhead_pct}, \"quality\": ${quality_delta_pct}},"
  echo "      \"thresholds\": {\"max_token_overhead_pct\": ${max_token_overhead_pct}, \"max_latency_overhead_pct\": ${max_latency_overhead_pct}, \"min_quality_delta_pct\": ${min_quality_delta_pct}},"
  echo "      \"within_threshold\": ${roi_within},"
  echo "      \"downgrade_recommendation\": \"$(json_escape "${recommended_tier}")\""
  echo "    }"
  echo "  },"
  echo "  \"hierarchy\": {"
  echo "    \"objective_domains\": [\"contract\", \"replay\", \"schema\", \"taxonomy\"],"
  echo "    \"computational_suites\": ["
  echo "      \"scripts/check-a64-impacted-gate-selection.sh\","
  echo "      \"scripts/check-a64-semantic-stability-contract.sh\","
  echo "      \"scripts/check-a64-performance-regression.sh\""
  echo "    ],"
  echo "    \"computational_coverage_pct\": ${computational_coverage_pct},"
  echo "    \"inferential_blocking_requested\": ${inferential_blocking_requested},"
  echo "    \"computational_first_compliant\": ${computational_first_compliant}"
  echo "  },"
  echo "  \"inferential_evidence\": {"
  echo "    \"input_snapshot\": \"$(json_escape "${input_snapshot_path}")\","
  echo "    \"prompt_version\": \"$(json_escape "${prompt_version}")\","
  echo "    \"scoring_summary\": \"$(json_escape "${scoring_summary}")\","
  echo "    \"uncertainty_pct\": ${uncertainty_pct},"
  echo "    \"max_uncertainty_pct\": ${max_uncertainty_pct},"
  echo "    \"uncertainty_within_threshold\": ${uncertainty_within},"
  echo "    \"evidence_complete\": ${evidence_complete}"
  echo "  },"
  echo "  \"gate_coverage_details\": ["
  for i in "${!gate_details[@]}"; do
    comma=","
    if [[ "${i}" -eq "$(( ${#gate_details[@]} - 1 ))" ]]; then
      comma=""
    fi
    echo "    ${gate_details[$i]}${comma}"
  done
  echo "  ],"
  echo "  \"docs_check_details\": ["
  for i in "${!docs_details[@]}"; do
    comma=","
    if [[ "${i}" -eq "$(( ${#docs_details[@]} - 1 ))" ]]; then
      comma=""
    fi
    echo "    ${docs_details[$i]}${comma}"
  done
  echo "  ],"
  echo "  \"score\": {"
  echo "    \"pass\": ${score_pass},"
  echo "    \"failed_checks\": ["
  for i in "${!failures[@]}"; do
    comma=","
    if [[ "${i}" -eq "$(( ${#failures[@]} - 1 ))" ]]; then
      comma=""
    fi
    echo "      \"$(json_escape "${failures[$i]}")\"${comma}"
  done
  echo "    ]"
  echo "  }"
  echo "}"
} | tee "${json_report_path}"

echo "[a64-harnessability-scorecard] report written to ${json_report_path}"

if [[ "${score_pass}" != "true" ]]; then
  echo "[a64-harnessability-scorecard] failed: ${failures[*]}" >&2
  exit 1
fi

echo "[a64-harnessability-scorecard] passed"
