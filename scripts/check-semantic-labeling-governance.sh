#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

MATRIX_FILE="openspec/governance/semantic-labeling-governed-path-matrix.yaml"
MAPPING_FILE="openspec/governance/semantic-labeling-legacy-mapping.yaml"
BASELINE_FILE="openspec/governance/semantic-labeling-regression-baseline.csv"

for required in "${MATRIX_FILE}" "${MAPPING_FILE}" "${BASELINE_FILE}"; do
  if [[ ! -f "${required}" ]]; then
    echo "[semantic-labeling-governance] missing required file: ${required}" >&2
    exit 1
  fi
done

extract_matrix_regex() {
  local check_id="$1"
  local raw=""
  raw="$(awk -v check_id="${check_id}" '
    $0 ~ "id:[[:space:]]*" check_id "$" { in_check=1; next }
    in_check && $1 == "regex:" {
      line=$0
      sub(/^[[:space:]]*regex:[[:space:]]*"/, "", line)
      sub(/"[[:space:]]*$/, "", line)
      print line
      exit
    }
    in_check && /^[[:space:]]*-[[:space:]]*id:[[:space:]]*/ { in_check=0 }
  ' "${MATRIX_FILE}")"
  if [[ -z "${raw}" ]]; then
    echo "[semantic-labeling-governance] failed to parse regex for check id: ${check_id}" >&2
    exit 1
  fi
  raw="${raw//\\\\/\\}"
  printf '%s' "${raw}"
}

mapfile -t governed_entries < <(awk '
  /^[[:space:]]*governed:[[:space:]]*$/ { in_governed=1; next }
  in_governed && /^[[:space:]]*checks:[[:space:]]*$/ { in_governed=0 }
  in_governed && /^[[:space:]]*-[[:space:]]*/ {
    line=$0
    sub(/^[[:space:]]*-[[:space:]]*/, "", line)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
    print line
  }
' "${MATRIX_FILE}")

if (( ${#governed_entries[@]} == 0 )); then
  echo "[semantic-labeling-governance] governed scope is empty in matrix" >&2
  exit 1
fi

declare -A scan_target_seen=()
declare -A governed_file_seen=()
scan_targets=()
governed_files=()

add_scan_target() {
  local target="$1"
  [[ -z "${target}" ]] && return 0
  if [[ -z "${scan_target_seen[${target}]:-}" ]]; then
    scan_target_seen["${target}"]=1
    scan_targets+=("${target}")
  fi
}

add_governed_file() {
  local file="$1"
  [[ -z "${file}" ]] && return 0
  if [[ -z "${governed_file_seen[${file}]:-}" ]]; then
    governed_file_seen["${file}"]=1
    governed_files+=("${file}")
  fi
}

for entry in "${governed_entries[@]}"; do
  if [[ "${entry}" == *"/**" ]]; then
    prefix="${entry%/**}"
    if [[ -d "${prefix}" ]]; then
      add_scan_target "${prefix}"
    fi
    while IFS= read -r file; do
      [[ -z "${file}" ]] && continue
      add_governed_file "${file}"
    done < <(git ls-files -- "${prefix}" || true)
    continue
  fi

  if [[ -e "${entry}" ]]; then
    add_scan_target "${entry}"
  fi
  while IFS= read -r file; do
    [[ -z "${file}" ]] && continue
    add_governed_file "${file}"
  done < <(git ls-files -- "${entry}" || true)
done

if (( ${#scan_targets[@]} == 0 )); then
  echo "[semantic-labeling-governance] no scan targets resolved from matrix" >&2
  exit 1
fi

if (( ${#governed_files[@]} == 0 )); then
  echo "[semantic-labeling-governance] no governed files resolved from matrix" >&2
  exit 1
fi

AXX_CONTENT_REGEX="$(extract_matrix_regex "legacy-axx-content")"
AXX_PATH_REGEX="$(extract_matrix_regex "legacy-axx-path")"
CA_REGEX="$(extract_matrix_regex "legacy-context-stage-wording")"

declare -A current_counts=()
declare -A baseline_counts=()
declare -A totals=()

normalize_key_path() {
  local path="$1"
  path="${path//\\//}"
  printf '%s' "${path}"
}

add_count() {
  local rule="$1"
  local path
  path="$(normalize_key_path "$2")"
  local key="${rule}|${path}"
  local current="${current_counts[${key}]:-0}"
  current_counts["${key}"]=$(( current + 1 ))
  totals["${rule}"]=$(( ${totals[${rule}]:-0} + 1 ))
}

collect_content_rule_counts() {
  local rule="$1"
  local regex="$2"
  mapfile -t hits < <(rg -n -- "${regex}" "${scan_targets[@]}" || true)
  for hit in "${hits[@]}"; do
    local file="${hit%%:*}"
    [[ -z "${file}" ]] && continue
    add_count "${rule}" "${file}"
  done
}

collect_path_rule_counts() {
  local rule="$1"
  local regex="$2"
  mapfile -t hits < <(printf '%s\n' "${governed_files[@]}" | rg -n -- "${regex}" || true)
  for hit in "${hits[@]}"; do
    local file="${hit#*:}"
    [[ -z "${file}" ]] && continue
    add_count "${rule}" "${file}"
  done
}

collect_content_rule_counts "legacy-axx-content" "${AXX_CONTENT_REGEX}"
collect_content_rule_counts "legacy-context-stage-wording-content" "${CA_REGEX}"
collect_path_rule_counts "legacy-axx-path" "${AXX_PATH_REGEX}"
collect_path_rule_counts "legacy-context-stage-wording-path" "${CA_REGEX}"

while IFS=, read -r rule path baseline; do
  if [[ "${rule}" == "rule" ]]; then
    continue
  fi
  rule="${rule//$'\r'/}"
  path="${path//$'\r'/}"
  baseline="${baseline//$'\r'/}"
  rule="${rule//\"/}"
  path="${path//\"/}"
  baseline="${baseline//\"/}"
  path="$(normalize_key_path "${path}")"
  [[ -z "${rule}" || -z "${path}" ]] && continue
  if ! [[ "${baseline}" =~ ^[0-9]+$ ]]; then
    echo "[semantic-labeling-governance] invalid baseline count in ${BASELINE_FILE}: ${rule},${path},${baseline}" >&2
    exit 1
  fi
  baseline_counts["${rule}|${path}"]="${baseline}"
done < "${BASELINE_FILE}"

violations=0
for key in "${!current_counts[@]}"; do
  current="${current_counts[${key}]}"
  if [[ -z "${baseline_counts[${key}]:-}" ]]; then
    echo "[semantic-labeling-governance][violation] new naming debt path detected: ${key} current=${current}" >&2
    (( violations += 1 ))
    continue
  fi
  baseline="${baseline_counts[${key}]}"
  if (( current > baseline )); then
    echo "[semantic-labeling-governance][violation] naming debt expanded: ${key} current=${current} baseline=${baseline}" >&2
    (( violations += 1 ))
  fi
done

mapfile -t mapping_duplicates < <(rg -n -- '^\s*(legacy_aliases|context_assembler_stage_mapping):\s*$' "${scan_targets[@]}" || true)
if (( ${#mapping_duplicates[@]} > 0 )); then
  echo "[semantic-labeling-governance][violation] duplicate mapping definitions found outside canonical source:" >&2
  for line in "${mapping_duplicates[@]}"; do
    echo "  ${line}" >&2
  done
  (( violations += ${#mapping_duplicates[@]} ))
fi

echo "[semantic-labeling-governance] summary:"
echo "  legacy-axx-content=${totals[legacy-axx-content]:-0}"
echo "  legacy-context-stage-wording-content=${totals[legacy-context-stage-wording-content]:-0}"
echo "  legacy-axx-path=${totals[legacy-axx-path]:-0}"
echo "  legacy-context-stage-wording-path=${totals[legacy-context-stage-wording-path]:-0}"
echo "  baseline_rows=$(( $(wc -l < "${BASELINE_FILE}") - 1 ))"

if (( violations > 0 )); then
  echo "[semantic-labeling-governance] failed: violations=${violations}" >&2
  exit 1
fi

echo "[semantic-labeling-governance] passed"
