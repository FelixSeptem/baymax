#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

export GOCACHE="${GOCACHE:-${REPO_ROOT}/.tmp/go-cache-agent-mode-smoke-stability}"
mkdir -p "${GOCACHE}"

BASELINE_PATH="${BAYMAX_AGENT_MODE_STABILITY_BASELINE_PATH:-examples/agent-modes/STABILITY_BASELINE.json}"
REPORT_PATH="${BAYMAX_AGENT_MODE_STABILITY_REPORT_PATH:-.tmp/agent-mode-smoke-stability-last-run.json}"
TIMEOUT_SEC="${BAYMAX_AGENT_MODE_STABILITY_TIMEOUT_SEC:-120}"
RETRY_MAX="${BAYMAX_AGENT_MODE_STABILITY_RETRY_MAX:-1}"

required_patterns=(
  "rag-hybrid-retrieval"
  "structured-output-schema-contract"
  "skill-driven-discovery-hybrid"
  "mcp-governed-stdio-http"
  "hitl-governed-checkpoint"
  "context-governed-reference-first"
  "sandbox-governed-toolchain"
  "realtime-interrupt-resume"
  "multi-agents-collab-recovery"
  "workflow-branch-retry-failfast"
  "mapreduce-large-batch"
  "state-session-snapshot-recovery"
  "policy-budget-admission"
  "tracing-eval-smoke"
  "react-plan-notebook-loop"
  "hooks-middleware-extension-pipeline"
  "observability-export-bundle"
  "adapter-onboarding-manifest-capability"
  "security-policy-event-delivery"
  "config-hot-reload-rollback"
  "workflow-routing-strategy-switch"
  "multi-agents-hierarchical-planner-validator"
  "mainline-mailbox-async-delayed-reconcile"
  "mainline-task-board-query-control"
  "mainline-scheduler-qos-backoff-dlq"
  "mainline-readiness-admission-degradation"
  "custom-adapter-mcp-model-tool-memory-pack"
  "custom-adapter-health-readiness-circuit"
)

contains_pattern() {
  local target="$1"
  shift
  local item=""
  for item in "$@"; do
    if [[ "${item}" == "${target}" ]]; then
      return 0
    fi
  done
  return 1
}

now_ms() {
  if date +%s%3N >/dev/null 2>&1; then
    date +%s%3N
    return
  fi
  echo "$(( $(date +%s) * 1000 ))"
}

extract_json_number() {
  local key="$1"
  local value
  value="$(grep -oE "\"${key}\"[[:space:]]*:[[:space:]]*[0-9]+([.][0-9]+)?" "${BASELINE_PATH}" | head -n1 | sed -E 's/.*:[[:space:]]*//')"
  echo "${value}"
}

if [[ ! -f "${BASELINE_PATH}" ]]; then
  echo "[agent-mode-smoke-stability-governance][missing-checklist] missing baseline: ${BASELINE_PATH}" >&2
  exit 1
fi

max_p95_ms="$(extract_json_number "max_p95_ms")"
max_flaky_rate="$(extract_json_number "max_flaky_rate")"
max_retry_rate="$(extract_json_number "max_retry_rate")"

if [[ -z "${max_p95_ms}" || -z "${max_flaky_rate}" || -z "${max_retry_rate}" ]]; then
  echo "[agent-mode-smoke-stability-governance][missing-checklist] baseline missing thresholds: max_p95_ms/max_flaky_rate/max_retry_rate" >&2
  exit 1
fi

selected_patterns=()
if [[ -n "${BAYMAX_AGENT_MODE_STABILITY_PATTERNS:-}" ]]; then
  IFS=',' read -r -a requested <<<"${BAYMAX_AGENT_MODE_STABILITY_PATTERNS}"
  for raw in "${requested[@]}"; do
    pattern="$(echo "${raw}" | xargs)"
    [[ -n "${pattern}" ]] || continue
    if ! contains_pattern "${pattern}" "${required_patterns[@]}"; then
      echo "[agent-mode-smoke-stability-governance] unsupported pattern in BAYMAX_AGENT_MODE_STABILITY_PATTERNS: ${pattern}" >&2
      exit 1
    fi
    selected_patterns+=("${pattern}")
  done
else
  selected_patterns=("${required_patterns[@]}")
fi

selected_variants=()
if [[ -n "${BAYMAX_AGENT_MODE_STABILITY_VARIANTS:-}" ]]; then
  IFS=',' read -r -a variants <<<"${BAYMAX_AGENT_MODE_STABILITY_VARIANTS}"
  for raw in "${variants[@]}"; do
    variant="$(echo "${raw}" | xargs)"
    [[ -n "${variant}" ]] || continue
    if [[ "${variant}" != "minimal" && "${variant}" != "production-ish" ]]; then
      echo "[agent-mode-smoke-stability-governance] unsupported variant: ${variant}" >&2
      exit 1
    fi
    selected_variants+=("${variant}")
  done
else
  selected_variants=("minimal")
fi

if (( ${#selected_patterns[@]} == 0 )); then
  echo "[agent-mode-smoke-stability-governance] no patterns selected" >&2
  exit 1
fi
if (( ${#selected_variants[@]} == 0 )); then
  echo "[agent-mode-smoke-stability-governance] no variants selected" >&2
  exit 1
fi

echo "[agent-mode-smoke-stability-governance] running stability checks for ${#selected_patterns[@]} patterns and ${#selected_variants[@]} variants"

start_ms="$(now_ms)"
durations_ms=()
total_cases=0
failed_cases=0
retry_total=0
flaky_cases=0

for pattern in "${selected_patterns[@]}"; do
  for variant in "${selected_variants[@]}"; do
    entry="./examples/agent-modes/${pattern}/${variant}"
    if [[ ! -d "${entry}" ]]; then
      echo "[agent-mode-smoke-stability-governance][missing-checklist] missing example directory: ${entry}" >&2
      exit 1
    fi

    total_cases=$((total_cases + 1))
    attempt=0
    success=0

    while (( attempt <= RETRY_MAX )); do
      attempt=$((attempt + 1))
      case_start_ms="$(now_ms)"
      rc=0
      set +e
      if command -v timeout >/dev/null 2>&1; then
        timeout --foreground "${TIMEOUT_SEC}s" go run "${entry}"
        rc=$?
      else
        go run "${entry}"
        rc=$?
      fi
      set -e
      case_end_ms="$(now_ms)"
      duration_ms="$((case_end_ms - case_start_ms))"

      if [[ ${rc} -eq 0 ]]; then
        success=1
        durations_ms+=("${duration_ms}")
        if (( attempt > 1 )); then
          flaky_cases=$((flaky_cases + 1))
          retry_total=$((retry_total + attempt - 1))
        fi
        echo "[agent-mode-smoke-stability-governance][metric] pattern=${pattern} variant=${variant} duration_ms=${duration_ms} attempts=${attempt}"
        break
      fi

      if (( attempt <= RETRY_MAX )); then
        retry_total=$((retry_total + 1))
        echo "[agent-mode-smoke-stability-governance][retry] pattern=${pattern} variant=${variant} attempt=${attempt} exit=${rc}"
        continue
      fi

      failed_cases=$((failed_cases + 1))
      if [[ ${rc} -eq 124 || ${rc} -eq 137 || ${rc} -eq 143 ]]; then
        echo "[agent-mode-smoke-stability-governance][timeout] pattern=${pattern} variant=${variant} timeout_sec=${TIMEOUT_SEC}" >&2
      fi
      echo "[agent-mode-smoke-stability-governance][failure] pattern=${pattern} variant=${variant} exit=${rc}" >&2
    done

    if (( success == 0 )); then
      break
    fi
  done
done

if (( total_cases == 0 )); then
  echo "[agent-mode-smoke-stability-governance][missing-checklist] no cases executed" >&2
  exit 1
fi

if (( ${#durations_ms[@]} == 0 )); then
  p50_ms=0
  p95_ms=0
else
  mapfile -t sorted_durations < <(printf '%s\n' "${durations_ms[@]}" | sort -n)
  count="${#sorted_durations[@]}"
  idx50=$(( (count - 1) * 50 / 100 ))
  idx95=$(( (count - 1) * 95 / 100 ))
  p50_ms="${sorted_durations[$idx50]}"
  p95_ms="${sorted_durations[$idx95]}"
fi

end_ms="$(now_ms)"
elapsed_ms="$((end_ms - start_ms))"

failure_rate="$(awk -v a="${failed_cases}" -v b="${total_cases}" 'BEGIN { if (b == 0) printf "0.000000"; else printf "%.6f", a / b }')"
retry_rate="$(awk -v a="${retry_total}" -v b="${total_cases}" 'BEGIN { if (b == 0) printf "0.000000"; else printf "%.6f", a / b }')"
flaky_rate="$(awk -v a="${flaky_cases}" -v b="${total_cases}" 'BEGIN { if (b == 0) printf "0.000000"; else printf "%.6f", a / b }')"

mkdir -p "$(dirname "${REPORT_PATH}")"
cat > "${REPORT_PATH}" <<EOF
{
  "timestamp_utc": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "total_cases": ${total_cases},
  "failed_cases": ${failed_cases},
  "retry_total": ${retry_total},
  "flaky_cases": ${flaky_cases},
  "p50_ms": ${p50_ms},
  "p95_ms": ${p95_ms},
  "elapsed_ms": ${elapsed_ms},
  "failure_rate": ${failure_rate},
  "retry_rate": ${retry_rate},
  "flaky_rate": ${flaky_rate}
}
EOF

echo "[agent-mode-smoke-stability-governance] summary total=${total_cases} failed=${failed_cases} retries=${retry_total} flaky=${flaky_cases} p50_ms=${p50_ms} p95_ms=${p95_ms} elapsed_ms=${elapsed_ms}"
echo "[agent-mode-smoke-stability-governance] report=${REPORT_PATH}"

breach=0
if (( p95_ms > max_p95_ms )); then
  echo "[agent-mode-smoke-stability-governance][example-smoke-latency-regression] current_p95_ms=${p95_ms} threshold_p95_ms=${max_p95_ms}" >&2
  breach=1
fi

if awk -v current="${flaky_rate}" -v threshold="${max_flaky_rate}" 'BEGIN { exit !(current > threshold) }'; then
  echo "[agent-mode-smoke-stability-governance][example-smoke-flaky-regression] current_flaky_rate=${flaky_rate} threshold_flaky_rate=${max_flaky_rate}" >&2
  breach=1
fi

if awk -v current="${retry_rate}" -v threshold="${max_retry_rate}" 'BEGIN { exit !(current > threshold) }'; then
  echo "[agent-mode-smoke-stability-governance][example-smoke-flaky-regression] current_retry_rate=${retry_rate} threshold_retry_rate=${max_retry_rate}" >&2
  breach=1
fi

if (( failed_cases > 0 )); then
  echo "[agent-mode-smoke-stability-governance][example-smoke-flaky-regression] failed_cases=${failed_cases}" >&2
  breach=1
fi

if (( breach == 1 )); then
  exit 1
fi

echo "[agent-mode-smoke-stability-governance] stability is within baseline thresholds"
