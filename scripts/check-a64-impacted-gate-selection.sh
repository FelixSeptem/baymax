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

parse_mode() {
  local explicit
  explicit="$(get_env_or_default "BAYMAX_A64_GATE_SELECTION_MODE" "")"
  explicit="$(echo "${explicit}" | tr '[:upper:]' '[:lower:]' | xargs)"
  if [[ -n "${explicit}" ]]; then
    if [[ "${explicit}" != "fast" && "${explicit}" != "full" ]]; then
      echo "[a64-impacted-gate-selection] BAYMAX_A64_GATE_SELECTION_MODE must be fast|full, got: ${explicit}" >&2
      exit 1
    fi
    echo "${explicit}"
    return
  fi
  local scope
  scope="$(get_env_or_default "BAYMAX_QUALITY_GATE_SCOPE" "full")"
  scope="$(echo "${scope}" | tr '[:upper:]' '[:lower:]' | xargs)"
  if [[ "${scope}" == "general" ]]; then
    echo "fast"
    return
  fi
  echo "full"
}

read_changed_files() {
  local from_env
  from_env="$(get_env_or_default "BAYMAX_A64_CHANGED_FILES" "")"
  if [[ -n "${from_env}" ]]; then
    echo "${from_env}" | tr ',;' '\n' | sed 's/\r$//' | awk 'NF > 0' | sort -u
    return
  fi
  {
    git diff --name-only HEAD
    git ls-files --others --exclude-standard
  } | sed 's/\r$//' | awk 'NF > 0' | sort -u
}

all_s_items=(S1 S2 S3 S4 S5 S6 S7 S8 S9 S10)

declare -A shell_map
declare -A powershell_map
shell_map[S1]=$'go test ./context/assembler ./context/provider ./context/journal -count=1\nbash scripts/check-diagnostics-replay-contract.sh'
powershell_map[S1]=$'go test ./context/assembler ./context/provider ./context/journal -count=1\npwsh -File scripts/check-diagnostics-replay-contract.ps1'
shell_map[S2]=$'bash scripts/check-diagnostics-replay-contract.sh\nbash scripts/check-diagnostics-query-performance-regression.sh'
powershell_map[S2]=$'pwsh -File scripts/check-diagnostics-replay-contract.ps1\npwsh -File scripts/check-diagnostics-query-performance-regression.ps1'
shell_map[S3]=$'bash scripts/check-multi-agent-shared-contract.sh\ngo test ./orchestration/scheduler ./orchestration/composer -count=1'
powershell_map[S3]=$'pwsh -File scripts/check-multi-agent-shared-contract.ps1\ngo test ./orchestration/scheduler ./orchestration/composer -count=1'
shell_map[S4]=$'go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1\nbash scripts/check-multi-agent-shared-contract.sh'
powershell_map[S4]=$'go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1\npwsh -File scripts/check-multi-agent-shared-contract.ps1'
shell_map[S5]='go test ./skill/loader ./runtime/config -count=1'
powershell_map[S5]='go test ./skill/loader ./runtime/config -count=1'
shell_map[S6]=$'bash scripts/check-memory-contract-conformance.sh\nbash scripts/check-memory-scope-and-search-contract.sh'
powershell_map[S6]=$'pwsh -File scripts/check-memory-contract-conformance.ps1\npwsh -File scripts/check-memory-scope-and-search-contract.ps1'
shell_map[S7]=$'bash scripts/check-security-policy-contract.sh\nbash scripts/check-security-event-contract.sh\nbash scripts/check-security-delivery-contract.sh\nbash scripts/check-security-sandbox-contract.sh'
powershell_map[S7]=$'pwsh -File scripts/check-security-policy-contract.ps1\npwsh -File scripts/check-security-event-contract.ps1\npwsh -File scripts/check-security-delivery-contract.ps1\npwsh -File scripts/check-security-sandbox-contract.ps1'
shell_map[S8]='bash scripts/check-react-contract.sh'
powershell_map[S8]='pwsh -File scripts/check-react-contract.ps1'
shell_map[S9]=$'bash scripts/check-policy-precedence-contract.sh\nbash scripts/check-runtime-budget-admission-contract.sh\nbash scripts/check-sandbox-rollout-governance-contract.sh'
powershell_map[S9]=$'pwsh -File scripts/check-policy-precedence-contract.ps1\npwsh -File scripts/check-runtime-budget-admission-contract.ps1\npwsh -File scripts/check-sandbox-rollout-governance-contract.ps1'
shell_map[S10]=$'bash scripts/check-observability-export-and-bundle-contract.sh\nbash scripts/check-diagnostics-replay-contract.sh'
powershell_map[S10]=$'pwsh -File scripts/check-observability-export-and-bundle-contract.ps1\npwsh -File scripts/check-diagnostics-replay-contract.ps1'

match_s_item_for_path() {
  local path="$1"
  case "${path}" in
    scripts/check-a64-*|openspec/changes/introduce-engineering-and-performance-optimization-contract-a64/*|docs/development-roadmap.md|docs/mainline-contract-test-index.md|docs/runtime-config-diagnostics.md)
      printf '%s\n' "${all_s_items[@]}"
      return
      ;;
    context/assembler/*|context/provider/*|context/journal/*)
      echo "S1"
      ;;
    runtime/diagnostics/*|observability/event/runtime_recorder*)
      echo "S2"
      ;;
    orchestration/scheduler/*|orchestration/mailbox/*|orchestration/composer/*)
      echo "S3"
      ;;
    mcp/http/*|mcp/stdio/*|mcp/retry/*|mcp/diag/*)
      echo "S4"
      ;;
    skill/loader/*)
      echo "S5"
      ;;
    memory/*)
      echo "S6"
      ;;
    core/runner/*|tool/local/*|orchestration/teams/*|orchestration/workflow/*)
      echo "S7"
      ;;
    model/openai/*|model/anthropic/*|model/gemini/*)
      echo "S8"
      ;;
    runtime/config/*)
      echo "S9"
      ;;
    observability/event/dispatcher*|observability/event/logger*|observability/event/runtime_exporter*)
      echo "S10"
      ;;
    *)
      ;;
  esac
}

mode="$(parse_mode)"
mapfile -t changed_files < <(read_changed_files)

declare -A impacted_set
if [[ "${mode}" == "full" ]]; then
  for item in "${all_s_items[@]}"; do
    impacted_set["${item}"]=1
  done
else
  for path in "${changed_files[@]}"; do
    while IFS= read -r item; do
      [[ -n "${item}" ]] || continue
      impacted_set["${item}"]=1
    done < <(match_s_item_for_path "${path}")
  done
  if [[ "${#changed_files[@]}" -gt 0 && "${#impacted_set[@]}" -eq 0 ]]; then
    echo "[a64-impacted-gate-selection] fast mode selected but no impacted S-items resolved; mapping must be updated" >&2
    exit 1
  fi
fi

impacted=()
for item in "${all_s_items[@]}"; do
  if [[ -n "${impacted_set[${item}]+x}" ]]; then
    impacted+=("${item}")
  fi
done

for item in "${impacted[@]}"; do
  if [[ -z "${shell_map[${item}]:-}" || -z "${powershell_map[${item}]:-}" ]]; then
    echo "[a64-impacted-gate-selection] incomplete suite mapping for ${item} (shell/powershell must both be non-empty)" >&2
    exit 1
  fi
done

echo "[a64-impacted-gate-selection] mode=${mode} changed_file_total=${#changed_files[@]} impacted=$(IFS=,; echo "${impacted[*]}")"
for item in "${impacted[@]}"; do
  echo "[a64-impacted-gate-selection] ${item} shell suites:"
  while IFS= read -r line; do
    [[ -n "${line}" ]] || continue
    echo "  - ${line}"
  done <<< "${shell_map[${item}]}"
  echo "[a64-impacted-gate-selection] ${item} powershell suites:"
  while IFS= read -r line; do
    [[ -n "${line}" ]] || continue
    echo "  - ${line}"
  done <<< "${powershell_map[${item}]}"
done

report_path="$(get_env_or_default "BAYMAX_A64_IMPACTED_REPORT_PATH" "")"
if [[ -n "${report_path}" ]]; then
  mkdir -p "$(dirname "${report_path}")"
  {
    echo "{"
    echo "  \"mode\": \"${mode}\","
    echo "  \"changed_file_total\": ${#changed_files[@]},"
    echo -n "  \"impacted_s_items\": ["
    for i in "${!impacted[@]}"; do
      if [[ "${i}" -gt 0 ]]; then
        echo -n ", "
      fi
      echo -n "\"${impacted[$i]}\""
    done
    echo "]"
    echo "}"
  } > "${report_path}"
  echo "[a64-impacted-gate-selection] report written to ${report_path}"
fi

echo "[a64-impacted-gate-selection] passed"
